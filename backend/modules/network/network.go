package network

import (
	_ "fmt"
	"strings"
	"log"
	"io"
	"net/http"

	"github.com/spf13/viper"
)

func LoadAllowedHosts(cfg *viper.Viper) []string {
    if cfg == nil {
        return []string{"127.0.0.1", "::1", "localhost"}
    }

    // Intentamos leer como Slice (el formato nativo de TOML para [a, b])
    hosts := cfg.GetStringSlice("security.allowed_hosts")
    
    // Si GetStringSlice falló, intentamos GetString por si acaso es un string simple
    if len(hosts) == 0 {
        raw := cfg.GetString("security.allowed_hosts")
        if raw != "" {
            parts := strings.Split(raw, ",")
            for _, p := range parts {
                hosts = append(hosts, strings.TrimSpace(p))
            }
        }
    }

    if len(hosts) == 0 {
        log.Println("Advertencia: security.allowed_hosts no encontrado o vacío → defaults")
        return []string{"127.0.0.1", "::1", "localhost"}
    }

    return hosts
}

func GetOutboundIP() string {
	conn, err := http.Get("https://api.ipify.org")
	if err != nil || conn == nil {
		return "127.0.0.1"
	}
	defer conn.Body.Close()

	ip, _ := io.ReadAll(conn.Body)
	return strings.TrimSpace(string(ip))
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}