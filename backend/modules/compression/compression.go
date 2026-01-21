package compression

import (
	"fmt"
	"os"
	"strings"
	"path/filepath"
	"io"
	
	"github.com/alexmullins/zip"
)

func UnzipWithPassword(src, dest, password string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Verificar si es directorio usando FileInfo()
		if f.FileInfo().IsDir() {
			continue
		}

		// Construir ruta segura (evitar path traversal)
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("ruta ilegal (posible path traversal): %s", fpath)
		}

		// Crear directorios padres
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return fmt.Errorf("no se pudo crear directorio %q: %w", filepath.Dir(fpath), err)
		}

		// Si el archivo está encriptado, establecer la contraseña
		// (solo se necesita si f.IsEncrypted() == true)
		if f.IsEncrypted() {
			f.FileHeader.SetPassword(password)   // ← string, no []byte
		}

		// Abrir el contenido del archivo (aquí se desencripta si corresponde)
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("no se pudo abrir el contenido de %q: %w", f.Name, err)
		}

		// Crear archivo en disco
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("no se pudo crear archivo destino %q: %w", fpath, err)
		}

		// Copiar datos
		_, copyErr := io.Copy(outFile, rc)

		// Cerrar recursos (importante hacerlo siempre)
		outFile.Close()
		rc.Close()

		if copyErr != nil {
			return fmt.Errorf("error al escribir %q: %w", f.Name, copyErr)
		}
	}

	return nil
}