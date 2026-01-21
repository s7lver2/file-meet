package filesGet

import (
	"fmt"
	"net/http"
	"time"
	"log"
	"path/filepath"
	"strconv"
	"os"
	"io"

	"github.com/gin-gonic/gin"
	_ "meet-backend/modules/network"
)

func ReceiveFile(c *gin.Context, tempDir string) {
	var payload struct {
		DownloadURL string `json:"download_url" binding:"required"`
		Filename    string `json:"filename" binding:"required"`
		Passphrase  string `json:"passphrase_enc,omitempty"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Faltan parámetros: download_url o filename"})
		return
	}

	log.Printf("POST /files/get → %s → %s", payload.DownloadURL, payload.Filename)

	client := &http.Client{Timeout: 40 * time.Second}
	resp, err := client.Get(payload.DownloadURL)
	if err != nil {
		log.Printf("Error al descargar: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("No se pudo conectar: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Descarga falló con status %d", resp.StatusCode)})
		return
	}

	zipPath := filepath.Join(tempDir, payload.Filename)
	f, err := os.Create(zipPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear archivo"})
		return
	}
	defer f.Close()

	size, err := io.Copy(f, resp.Body)
	if err != nil {
		log.Printf("Error al guardar archivo: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar archivo"})
		return
	}

	// Crear archivo .info con timestamp
	infoPath := zipPath + ".info"
	os.WriteFile(infoPath, []byte(strconv.FormatInt(time.Now().Unix(), 10)), 0644)

	log.Printf("Descarga OK → %s  (%.1f MB)", zipPath, float64(size)/1024/1024)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Archivo recibido y guardado como %s", payload.Filename),
		"path":    zipPath,
	})
}