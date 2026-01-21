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
        log.Println("Advertencia: viper config es nil → usando defaults")
        return []string{"127.0.0.1", "::1", "localhost"}
    }

    raw := cfg.GetString("meet.allowed_hosts")
    if raw == "" {
        log.Println("Advertencia: meet.allowed_hosts no encontrado → defaults")
        return []string{"127.0.0.1", "::1", "localhost"}
    }

    parts := strings.Split(raw, ",")
    seen := make(map[string]struct{})
    var hosts []string

    for _, p := range parts {
        trimmed := strings.TrimSpace(p)
        if trimmed == "" {
            continue
        }
        if _, exists := seen[trimmed]; !exists {
            seen[trimmed] = struct{}{}
            hosts = append(hosts, trimmed)
        }
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