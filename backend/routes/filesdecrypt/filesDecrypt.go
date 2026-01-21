package filesDecrypt

import (
	"fmt"
	"os"
	"net/http"
	"log"
	"path/filepath"
	"time"
	"strings"

	"github.com/gin-gonic/gin"
	"meet-backend/modules/compression"
)

func DecryptFile(c *gin.Context, tempDir string) {
	filename := c.Param("filename")
	code := c.Query("code")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetro obligatorio: ?code=xxx"})
		return
	}

	zipPath := filepath.Join(tempDir, filename)
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Archivo no encontrado en temp_downloads"})
		return
	}

	// ──── passphrase del RECEPTOR (este servidor) ────
	localPassphrase := c.GetString("meet.passphrase")
	if localPassphrase == "" {
		log.Println("ERROR: No hay passphrase definida en config.ini → [meet] passphrase")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Configuración incompleta: falta passphrase en este servidor",
		})
		return
	}

	// Contraseña completa que se va a probar
	attemptedPassword := localPassphrase + code
	log.Printf("Intentando desencriptar %s con contraseña de %d caracteres", filename, len(attemptedPassword))

	// Carpeta donde se extraerá (única por archivo para evitar colisiones)
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	extractDir := filepath.Join(tempDir, "extracted-"+baseName+"-"+time.Now().Format("20060102-150405"))

	if err := os.MkdirAll(extractDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No se pudo crear directorio de extracción"})
		return
	}

	// Intento real de extracción con contraseña
	err := compression.UnzipWithPassword(zipPath, extractDir, attemptedPassword)
	if err != nil {
		if isWrongPasswordError(err) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Código incorrecto → la combinación passphrase local + código no abre el archivo",
			})
			return
		}

		log.Printf("Error al extraer ZIP: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Fallo al desencriptar/extraer: %v", err),
		})
		return
	}

	// Éxito → limpieza
	os.Remove(zipPath)
	os.Remove(zipPath + ".info")
	log.Printf("Éxito: %s extraído en %s", filename, extractDir)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Archivo desencriptado y extraído correctamente (usando passphrase local del servidor)",
		"extracted_to": extractDir,
	})
}

// Helper para detectar error de contraseña incorrecta
// (depende de la librería de zip que uses)
func isWrongPasswordError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "password") ||
		strings.Contains(msg, "incorrect") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "bad password") ||
		strings.Contains(msg, "wrong password")
}

// unzipWithPassword extrae un zip protegido con contraseña
