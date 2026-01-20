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
)

var WORD_LIST = []string{
	"sol", "luna", "estrella", "rio", "montaña", "bosque", "cielo", "mar", "viento", "fuego",
}

func Send(archivo string, destino string) error {
	config, err := loadHostsIni()
	if err != nil {
		return fmt.Errorf("error cargando hosts.ini: %v", err)
	}

	section := config.Section(destino)
	passphrase := strings.TrimSpace(section.Key("passphrase").String())
	if passphrase == "" {
		return fmt.Errorf("no se encontró passphrase para '%s'", destino)
	}

	targetHost := strings.TrimSpace(section.Key("address").String())
	if targetHost == "" {
		targetHost = strings.TrimSpace(section.Key("hostname").String())
	}
	if targetHost == "" {
		return fmt.Errorf("no se encontró 'address' o 'hostname' para '%s'", destino)
	}

	// Initialize random seed (important for older Go versions, good practice)
	rand.Seed(time.Now().UnixNano())

	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	randomWord := WORD_LIST[rand.Intn(len(WORD_LIST))]
	password := passphrase + code

	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("   PALABRA CLAVE DEL ARCHIVO: %s\n", randomWord)
	fmt.Printf("   (úsala en el comando decrypt en el receptor)\n")
	fmt.Printf("   ¡CÓDIGO SECRETO (6 dígitos) PARA EL RECEPTOR: %s\n", code)
	fmt.Println(strings.Repeat("=", 60) + "\n")

	archivoPath := filepath.Clean(archivo)
	zipName := randomWord + ".zip"

	tmpDir, err := os.MkdirTemp("", "file-meet-send-*")
	if err != nil {
		return fmt.Errorf("error creando dir temporal: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, zipName)
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("❌ Error creando el archivo ZIP: %v", err)
	}

	zw := zip.NewWriter(zipFile)

	// FIX: Use zw.Encrypt instead of CreateHeader + SetPassword
	w, err := zw.Encrypt(filepath.Base(archivoPath), password)
	if err != nil {
		zipFile.Close()
		return fmt.Errorf("error configurando cifrado: %v", err)
	}

	src, err := os.Open(archivoPath)
	if err != nil {
		zipFile.Close()
		return fmt.Errorf("❌ Error abriendo el archivo original: %v", err)
	}
	
	_, err = io.Copy(w, src)
	src.Close() // Close source as soon as possible
	if err != nil {
		zipFile.Close()
		return fmt.Errorf("❌ Error copiando contenido al ZIP: %v", err)
	}

	// CRITICAL: Close writers so the file is flushed to disk before serving
	zw.Close()
	zipFile.Close()

	fmt.Printf("✓ Archivo comprimido y cifrado: %s\n", zipPath)

	localIP := getLocalIP()
	servePort := 8080
	downloadURL := fmt.Sprintf("http://%s:%d/%s", localIP, servePort, zipName)

	mux := http.NewServeMux()
	mux.HandleFunc("/"+zipName, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipPath)
		fmt.Printf("\n[!] Archivo %s solicitado por %s\n", zipName, r.RemoteAddr)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", servePort),
		Handler: mux,
	}

	go func() {
		fmt.Printf("Servidor temporal activo en %s\n", downloadURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error en servidor temporal: %v\n", err)
		}
	}()

	// Small sleep to ensure server is listening
	time.Sleep(500 * time.Millisecond)

	// Enviar POST al destino
	targetURL := fmt.Sprintf("http://%s:%d/files/get", targetHost, PORT)
	payload := map[string]string{
		"download_url": downloadURL,
		"filename":     zipName,
		"code_hint":    "El código de 6 dígitos fue compartido por el emisor",
	}

	jsonPayload, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(targetURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("⚠️  No se pudo notificar al receptor: %v\n", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			fmt.Printf("✓ Notificación enviada exitosamente a %s\n", targetHost)
		} else {
			fmt.Printf("✗ El receptor respondió con error (Status: %d)\n", resp.StatusCode)
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\nEsperando descarga... (El servidor se cerrará en 10 min o con Ctrl+C)")

	select {
	case <-sigChan:
		fmt.Println("\nDeteniendo manualmente...")
	case <-time.After(10 * time.Minute):
		fmt.Println("\nTiempo de espera agotado.")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func loadHostsIni() (*ini.File, error) {
	wd, _ := os.Getwd()
	// Try current dir first, then parent
	path := filepath.Join(wd, "hosts.ini")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(wd, "..", "hosts.ini")
	}
	return ini.Load(path)
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