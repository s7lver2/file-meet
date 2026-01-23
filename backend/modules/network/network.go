package network

import (
	_ "fmt"
	"strings"
	"log"
    "net"

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

func GetPrivateIP() string {
    interfaces, err := net.Interfaces()
    if err != nil {
        return "127.0.0.1"
    }

    for _, iface := range interfaces {
        // ignoramos interfaces down o loopback
        if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
            continue
        }

        addrs, err := iface.Addrs()
        if err != nil {
            continue
        }

        for _, addr := range addrs {
            var ip net.IP
            switch v := addr.(type) {
            case *net.IPNet:
                ip = v.IP
            case *net.IPAddr:
                ip = v.IP
            }

            if ip == nil || ip.IsLoopback() {
                continue
            }

            // Solo IPv4 privadas
            if ip4 := ip.To4(); ip4 != nil {
                if isPrivateIPv4(ip4) {
                    return ip4.String()
                }
            }
        }
    }

    // fallback
    return "127.0.0.1"
}

func isPrivateIPv4(ip net.IP) bool {
    // Rangos privados RFC 1918 + CGNAT + link-local
    return ip[0] == 10 ||
        (ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) ||
        (ip[0] == 192 && ip[1] == 168) ||
        (ip[0] == 100 && ip[1] >= 64 && ip[1] <= 127) || // 100.64.0.0/10 CGNAT
        (ip[0] == 169 && ip[1] == 254)                    // 169.254.0.0/16 link-local
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}