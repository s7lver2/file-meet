package decrypt

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PORT debe ser el mismo que usa tu backend (42532)
const PORT = 42532

// DecryptClient llama al endpoint del backend para desencriptar el archivo
func Decrypt(filename string, code string) error {
	// Construir la URL con el puerto y parámetros
	url := fmt.Sprintf("http://localhost:%d/files/decrypt/%s?code=%s", PORT, filename, code)

	// Configurar cliente con timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Realizar la petición GET
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("✗ No se pudo conectar al backend local.")
		fmt.Println("   Asegúrate de que el servidor esté corriendo con 'meet start'")
		return err
	}
	defer resp.Body.Close()

	// Leer el cuerpo de la respuesta
	body, _ := io.ReadAll(resp.Body)

	// Manejar errores según el Status Code (equivalente a r.raise_for_status)
	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusForbidden: // 403
			fmt.Println("✗ Código incorrecto o passphrase no coincide")
		case http.StatusNotFound: // 404
			fmt.Printf("✗ Archivo '%s' no encontrado en temp_downloads\n", filename)
		default:
			fmt.Printf("✗ Error del servidor (Status %d): %s\n", resp.StatusCode, string(body))
		}
		return fmt.Errorf("error del servidor: %d", resp.StatusCode)
	}

	// Decodificar JSON de éxito
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("error al leer respuesta JSON: %v", err)
	}

	// Imprimir mensaje de éxito (equivalente a click.echo)
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("¡ÉXITO! Archivo desencriptado correctamente")
	
	msg := data["message"]
	if msg == nil {
		msg = "Extraído en carpeta 'extracted'"
	}
	fmt.Printf("%v\n", msg)
	fmt.Println(strings.Repeat("=", 60) + "\n")

	return nil
}