package send

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/alexmullins/zip"
	"gopkg.in/ini.v1"
)

const (
	PORT = 42532
	SERVE_PORT = 42532
)

var WORD_LIST = []string{
	"sol", "luna", "estrella", "rio", "montaña", "bosque", "cielo", "mar", "viento", "fuego",
}

func Send(archivo string, destino string) error {
    config, err := loadHostsIni()
    if err != nil {
        return fmt.Errorf("error cargando hosts.ini: %w", err)
    }
    section := config.Section(destino)
    if section == nil {
        return fmt.Errorf("no se encontró la sección [%s] en hosts.ini", destino)
    }
    passphrase := strings.TrimSpace(section.Key("passphrase").String())
    if passphrase == "" {
        return fmt.Errorf("no se encontró passphrase para '%s' en hosts.ini", destino)
    }
    targetHost := strings.TrimSpace(section.Key("address").String())
    if targetHost == "" {
        targetHost = strings.TrimSpace(section.Key("hostname").String())
    }
    if targetHost == "" {
        return fmt.Errorf("no se encontró 'address' ni 'hostname' para '%s'", destino)
    }

    // Generar código aleatorio de 6 dígitos
    rng := rand.New(rand.NewSource(time.Now().UnixNano()))
    code := fmt.Sprintf("%06d", rng.Intn(1000000))
    randomWord := WORD_LIST[rng.Intn(len(WORD_LIST))]
    password := passphrase + code

    fmt.Println(strings.Repeat("=", 60))
    fmt.Printf("  PALABRA CLAVE DEL ARCHIVO: %s\n", randomWord)
    fmt.Println("  (úsala en el comando decrypt en el receptor para identificarlo)")
    fmt.Printf("  CÓDIGO SECRETO (6 dígitos) PARA EL RECEPTOR: %s\n", code)
    fmt.Println(strings.Repeat("=", 60) + "\n")

    // ────────────────────────────────────────────────
    // Preparar archivo ZIP cifrado
    // ────────────────────────────────────────────────
    archivoPath := filepath.Clean(archivo)
    if _, err := os.Stat(archivoPath); os.IsNotExist(err) {
        return fmt.Errorf("el archivo no existe: %s", archivoPath)
    }

    tmpDir, err := os.MkdirTemp("", "meet-send-*")
    if err != nil {
        return fmt.Errorf("error creando directorio temporal: %w", err)
    }
    defer os.RemoveAll(tmpDir)

    zipName := randomWord + ".zip"
    zipPath := filepath.Join(tmpDir, zipName)

    zipFile, err := os.Create(zipPath)
    if err != nil {
        return fmt.Errorf("error creando el ZIP: %w", err)
    }

    zw := zip.NewWriter(zipFile)

    encryptedWriter, err := zw.Encrypt(filepath.Base(archivoPath), password)
    if err != nil {
        zw.Close()
        zipFile.Close()
        return fmt.Errorf("error configurando cifrado AES: %w", err)
    }

    src, err := os.Open(archivoPath)
    if err != nil {
        zw.Close()
        zipFile.Close()
        return fmt.Errorf("error abriendo archivo original: %w", err)
    }
    defer src.Close()

    _, err = io.Copy(encryptedWriter, src)
    if err != nil {
        zw.Close()
        zipFile.Close()
        return fmt.Errorf("error copiando contenido al ZIP cifrado: %w", err)
    }

    if err := zw.Close(); err != nil {
        zipFile.Close()
        return fmt.Errorf("error cerrando ZIP writer: %w", err)
    }
    if err := zipFile.Close(); err != nil {
        return fmt.Errorf("error cerrando archivo ZIP: %w", err)
    }

    fmt.Printf("✓ Archivo comprimido y cifrado: %s\n", zipPath)

    // ────────────────────────────────────────────────
    // El resto del código (servidor temporal, POST, espera) queda igual
    // ────────────────────────────────────────────────

    localIP := getLocalIP()
    downloadURL := fmt.Sprintf("http://%s:%d/%s", localIP, SERVE_PORT, zipName)

    mux := http.NewServeMux()
    mux.HandleFunc("/"+zipName, func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, zipPath)
        fmt.Printf("[!] Archivo %s solicitado desde %s\n", zipName, r.RemoteAddr)
    })

    srv := &http.Server{
        Addr:    fmt.Sprintf(":%d", SERVE_PORT),
        Handler: mux,
    }

    go func() {
        fmt.Printf("Servidor temporal escuchando en: %s\n", downloadURL)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            fmt.Printf("Error en servidor temporal: %v\n", err)
        }
    }()

    time.Sleep(400 * time.Millisecond)

    targetURL := fmt.Sprintf("http://%s:%d/files/get", targetHost, PORT)
    payload := map[string]string{
        "download_url": downloadURL,
        "filename":     zipName,
    }
    jsonData, _ := json.Marshal(payload)

    client := &http.Client{Timeout: 12 * time.Second}
    resp, err := client.Post(targetURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Printf("⚠️  No se pudo contactar al receptor (%s): %v\n", targetHost, err)
    } else {
        defer resp.Body.Close()
        body, _ := io.ReadAll(resp.Body)
        if resp.StatusCode == http.StatusOK {
            fmt.Printf("✓ Receptor %s notificado correctamente\n", targetHost)
        } else {
            fmt.Printf("✗ Receptor respondió con error %d: %s\n", resp.StatusCode, strings.TrimSpace(string(body)))
        }
    }

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    fmt.Println("\nEsperando a que el receptor descargue el archivo...")
    fmt.Println("  (se cerrará automáticamente en 10 minutos o con Ctrl+C)")

    select {
    case <-sigChan:
        fmt.Println("\nDetenido por el usuario")
    case <-time.After(10 * time.Minute):
        fmt.Println("\nTiempo máximo alcanzado (10 min)")
    }

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
    defer cancel()
    if err := srv.Shutdown(shutdownCtx); err != nil {
        fmt.Printf("Error durante shutdown del servidor temporal: %v\n", err)
    }

    return nil
}

func loadHostsIni() (*ini.File, error) {
	wd, _ := os.Getwd()

	// Intentamos en diferentes ubicaciones comunes
	locations := []string{
		filepath.Join(wd, "hosts.ini"),
		filepath.Join(wd, "..", "hosts.ini"),
		filepath.Join(wd, "../config", "hosts.ini"),
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return ini.Load(path)
		}
	}

	return nil, fmt.Errorf("no se encontró hosts.ini en las ubicaciones habituales")
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}