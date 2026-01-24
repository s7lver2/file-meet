package discovery

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
)

var (
	config          *viper.Viper
)

const (
	OLD_HOST_THRESHOLD = 48 * time.Hour // 48 horas
)

var (
	projectRoot = func() string {
		// Ajusta según dónde esté tu main o cli
		// Esto asume que discover.go está en client/modules/
		p, _ := os.Getwd()
		return filepath.Join(p, "..")
	}()
	hostsIniPath = filepath.Join(projectRoot, "hosts.ini")
)

// ────────────────────────────────────────────────
// Parte 1: Escaneo asíncrono de puertos
// ────────────────────────────────────────────────

func probePort(ctx context.Context, ip string, port int) bool {
	d := net.Dialer{Timeout: time.Duration(config.GetInt("scan.timeout_connect")) * time.Millisecond}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func findOpenHosts(networkStr string) ([]string, error) {
	_, ipNet, err := net.ParseCIDR(networkStr)
	if err != nil {
		return nil, err
	}

	var hosts []string
	for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
		if !ip.Equal(ipNet.IP) && !ip.Equal(broadcastIP(ipNet)) {
			hosts = append(hosts, ip.String())
		}
	}

	fmt.Printf("Escaneando %d hosts en %s puerto %d...\n", len(hosts), networkStr, config.GetInt("scan.port"))
	sem := semaphore.NewWeighted(int64(config.GetInt("scan.port")))
	ctx := context.Background()

	var wg sync.WaitGroup
	openCh := make(chan string, len(hosts))

	for _, ip := range hosts {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				return
			}
			defer sem.Release(1)

			if probePort(ctx, ip, config.GetInt("scan.port")) {
				fmt.Printf("  ABIERTO → %s:%d\n", ip, config.GetInt("scan.port"))
				openCh <- ip
			}
		}(ip)
	}

	go func() {
		wg.Wait()
		close(openCh)
	}()

	var openIPs []string
	for ip := range openCh {
		openIPs = append(openIPs, ip)
	}

	fmt.Printf("Encontradas %d IPs con puerto abierto.\n", len(openIPs))
	return openIPs, nil
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func broadcastIP(n *net.IPNet) net.IP {
	bc := make(net.IP, len(n.IP))
	copy(bc, n.IP)
	for i := range bc {
		bc[i] |= ^n.Mask[i]
	}
	return bc
}

// ────────────────────────────────────────────────
// Parte 2: Obtener datos vía HTTP y actualizar .ini
// ────────────────────────────────────────────────

type MeetData struct {
	Hostname   string
	Passphrase string
	Address    string
}

func fetchMeetData(ip string) (*MeetData, error) {
	url := fmt.Sprintf("http://%s:%d/meet", ip, config.GetInt("scan.port"))
	client := &http.Client{Timeout: time.Duration(config.GetInt("scan.timeout_request")) * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Muy simple: parseamos JSON manualmente (puedes usar encoding/json)
	// Aquí asumimos respuesta como {"hostname": "...", "passphrase": "...", "address": "..."}
	data := make(map[string]string)
	// Para producción → json.Unmarshal(body, &data)
	// Por simplicidad y para evitar dependencias extra, parseamos rudimentario
	content := string(body)
	for _, line := range strings.Split(content, ",") {
		kv := strings.SplitN(line, ":", 2)
		if len(kv) == 2 {
			key := strings.Trim(strings.Trim(kv[0], `"{`), " ")
			val := strings.Trim(strings.Trim(kv[1], `"}`), " ")
			data[key] = val
		}
	}

	required := []string{"hostname", "passphrase", "address"}
	for _, k := range required {
		if _, ok := data[k]; !ok {
			return nil, fmt.Errorf("faltan campos requeridos")
		}
	}

	return &MeetData{
		Hostname:   strings.TrimSpace(data["hostname"]),
		Passphrase: strings.TrimSpace(data["passphrase"]),
		Address:    strings.TrimSpace(data["address"]),
	}, nil
}

func updateHostsIni(newEntries map[string]*MeetData) error {
	fmt.Printf("[DEBUG] updateHostsIni llamado con %d entradas\n", len(newEntries))
	var cfg *ini.File
    var err error

    _, err = os.Stat(hostsIniPath)
	if err != nil {
    	if os.IsNotExist(err) {
        	cfg = ini.Empty()
        	fmt.Println("hosts.ini no existe → creando nuevo")
    	} else {
        	// Aquí se atasca: imprime el error real
        	fmt.Printf("[ERROR CRÍTICO] No se puede acceder a hosts.ini en %s\n", hostsIniPath)
        	fmt.Printf("[ERROR DETALLE] %v\n", err)
        	return fmt.Errorf("error verificando hosts.ini: %w", err)
    	}
	} else {
    	fmt.Printf("[INFO] Archivo existe → intentando cargar: %s\n", hostsIniPath)
    	cfg, err = ini.Load(hostsIniPath)
    	if err != nil {
        	fmt.Printf("[ERROR AL CARGAR] No se pudo leer hosts.ini\n")
        	fmt.Printf("[ERROR DETALLE] %v\n", err)
        	return fmt.Errorf("error cargando hosts.ini: %w", err)
    	}
    	fmt.Printf("Leyendo hosts.ini existente (%d secciones)\n", len(cfg.Sections()))
	}
	now := time.Now().Unix()
	updated := 0
	added := 0

	for ip, data := range newEntries {
		hostname := data.Hostname
		if hostname == "" {
			hostname = "Host_" + strings.ReplaceAll(ip, ".", "_")
		}
		section := cfg.Section(hostname)

		wasNew := !section.HasKey("ip")
		if wasNew {
			added++
			fmt.Printf("  Añadiendo nuevo: %s (%s)\n", hostname, ip)
		} else {
			updated++
			fmt.Printf("  Actualizando %s (%s)\n", hostname, ip)
		}

		section.Key("hostname").SetValue(hostname)
		section.Key("passphrase").SetValue(data.Passphrase)
		section.Key("address").SetValue(data.Address)
		section.Key("ip").SetValue(ip)
		section.Key("last_seen").SetValue(strconv.FormatInt(now, 10))
	}

	// Limpieza de hosts antiguos
	toRemove := []string{}
    for _, sec := range cfg.Sections() {
        if sec.Name() == "Local" {
            continue
        }
        lastSeenStr := sec.Key("last_seen").String()
        if lastSeenStr == "" {
            continue
        }
        lastSeen, _ := strconv.ParseInt(lastSeenStr, 10, 64)
        if time.Unix(lastSeen, 0).Add(time.Duration(config.GetInt("scan.old_host_threshold")) * time.Hour).Before(time.Now()) {
            toRemove = append(toRemove, sec.Name())
        }
    }

	for _, name := range toRemove {
        cfg.DeleteSection(name)
        fmt.Printf("  Eliminado host antiguo: %s\n", name)
    }

	if added > 0 || updated > 0 || len(toRemove) > 0 {
        fmt.Printf("[DEBUG] Guardando archivo en: %s\n", hostsIniPath)
        err = cfg.SaveTo(hostsIniPath)
        if err != nil {
            fmt.Printf("[ERROR AL GUARDAR] %v\n", err)
            return err
        }
        fmt.Printf("\nhosts.ini actualizado correctamente:\n")
        fmt.Printf("  Nuevos: %d\n", added)
        fmt.Printf("  Actualizados: %d\n", updated)
        fmt.Printf("  Eliminados antiguos: %d\n", len(toRemove))
        total := len(cfg.Sections())
        if cfg.HasSection("Local") {
            total--
        }
        fmt.Printf("  Total hosts remotos: %d\n", total)
    } else {
        fmt.Println("\nNo hubo cambios en hosts.ini")
    }

    return nil
}

// ────────────────────────────────────────────────
// Función principal
// ────────────────────────────────────────────────

func DiscoverAndUpdate(config *viper.Viper) error {
	start := time.Now()

	openIPs, err := findOpenHosts(config.GetString("scan.range"))
	if err != nil {
		return err
	}
	if len(openIPs) == 0 {
		fmt.Println("No se encontraron hosts con puerto abierto.")
		return nil
	}

	discovered := make(map[string]*MeetData)

	for _, ip := range openIPs {
		fmt.Printf("  Consultando /meet en %s...\n", ip)
		data, err := fetchMeetData(ip)
		if err != nil {
			fmt.Printf("  No se pudo obtener datos válidos de %s: %v\n", ip, err)
			continue
		}
		discovered[ip] = data
	}

	if len(discovered) == 0 {
    fmt.Println("Ningún host respondió con datos válidos en /meet")
    return nil
	}

	wd, _ := os.Getwd()   // ignora el error explícitamente

	fmt.Printf("[DEBUG] Directorio de trabajo actual: %s\n", wd)
	fmt.Printf("[DEBUG] Ruta calculada para hosts.ini: %s\n", hostsIniPath)

	err = updateHostsIni(discovered)
	if err != nil {
    	return err
	}

	fmt.Printf("Tiempo total: %.2f segundos\n", time.Since(start).Seconds())

	return nil
}

func main() {
	start := time.Now()
	err := DiscoverAndUpdate()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Tiempo total: %.2f segundos\n", time.Since(start).Seconds())
}