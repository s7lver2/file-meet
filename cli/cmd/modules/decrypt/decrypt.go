package decrypt

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// PORT debe ser el mismo que usa tu backend (42532)
const PORT = 42532

// DecryptClient llama al endpoint del backend para desencriptar el archivo
func Decrypt(filename string, code string) error {
	port := PORT
	// Opcional: leer de variable de entorno si existe
	if p := os.Getenv("MEET_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	url := fmt.Sprintf("http://localhost:%d/files/decrypt/%s?code=%s", port, filename, code)

	client := &http.Client{
		Timeout: 30 * time.Second, // más generoso para extracciones grandes
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("✗ No se pudo conectar al backend local")
		fmt.Println("  Asegúrate de que el servidor esté corriendo con:")
		fmt.Println("     meet start   (o el comando que uses para levantar el backend)")
		fmt.Printf("  Error de conexión: %v\n", err)
		return fmt.Errorf("fallo de conexión al backend: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("no se pudo leer la respuesta del servidor: %w", err)
	}

	body := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		var errMsg string

		// Intentamos parsear el JSON de error que envía el backend
		var errData map[string]interface{}
		if jsonErr := json.Unmarshal(bodyBytes, &errData); jsonErr == nil {
			if msg, ok := errData["error"].(string); ok && msg != "" {
				errMsg = msg
			}
		}

		// Si no pudimos parsear, usamos el body crudo
		if errMsg == "" {
			errMsg = strings.TrimSpace(body)
			if errMsg == "" {
				errMsg = "(sin mensaje detallado del servidor)"
			}
		}

		switch resp.StatusCode {
		case http.StatusBadRequest:
			fmt.Println("✗ Solicitud inválida:", errMsg)
		case http.StatusNotFound:
			fmt.Printf("✗ Archivo '%s' no encontrado en el servidor\n", filename)
		case http.StatusForbidden:
			fmt.Println("✗ Código incorrecto o passphrase no coincide")
			fmt.Println("  Verifica que el código sea correcto y que la passphrase")
			fmt.Println("  en config.ini del servidor coincida con la usada al cifrar.")
		case http.StatusInternalServerError:
			fmt.Println("✗ Error interno del servidor (500)")
			fmt.Printf("  Detalle: %s\n", errMsg)
			fmt.Println("  Posibles causas:")
			fmt.Println("   • Falta passphrase en config.ini")
			fmt.Println("   • Error al extraer el ZIP")
			fmt.Println("   • Problema de permisos en disco")
		default:
			fmt.Printf("✗ Error inesperado del servidor (Status %d): %s\n", resp.StatusCode, errMsg)
		}

		return fmt.Errorf("error del servidor: %d - %s", resp.StatusCode, errMsg)
	}

	// Respuesta OK → parseamos el JSON de éxito
	var data map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return fmt.Errorf("respuesta OK pero JSON inválido: %w", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("¡ÉXITO! Archivo desencriptado y extraído correctamente")

	// Mostramos los campos más útiles que devuelve el backend
	if msg, ok := data["message"].(string); ok && msg != "" {
		fmt.Println(msg)
	}

	if path, ok := data["extracted_to"].(string); ok && path != "" {
		fmt.Printf("→ Carpeta de extracción: %s\n", path)
	}

	fmt.Println(strings.Repeat("=", 60) + "\n")
	return nil
}