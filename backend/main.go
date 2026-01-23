package main

import (
	"log"
	"os"
	"path/filepath"

	"meet-backend/routes/filesget"
	"meet-backend/routes/filesdecrypt"
	"meet-backend/modules/network"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var (
	tempDir         = filepath.Join(".", "temp_downloads")
	config          *viper.Viper
	allowedSelfHosts []string
	prIP            string
	// ALLOWED_IPS     []string   // si lo necesitas más adelante
)

func main() {
	// Crear carpeta temporal si no existe
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Fatalf("No se pudo crear temp_downloads: %v", err)
	}

	// Cargar configuración (equivalente a tu config.ini)
	config = viper.New()
	config.SetConfigName("config")
	config.SetConfigType("toml")
	config.AddConfigPath("../")           // mismo nivel que el binario o ajusta según tu estructura
	config.AddConfigPath(".")             // fallback
	config.AddConfigPath(filepath.Join("../config/meet/config"))
	config.AddConfigPath(filepath.Join("../config"))

	config.SetDefault("meet.hostname", "localhost")
    config.SetDefault("meet.passphrase", "")
    config.SetDefault("security.allowed_hosts", []string{"127.0.0.1", "localhost"})
    config.SetDefault("server.port", 42532)
    config.SetDefault("server.temp_dir", "temp_downloads")

	if err := config.ReadInConfig(); err != nil {
		log.Printf("No se pudo leer config.ini → usando valores por defecto: %v", err)
	}

	// Cargar allowed hosts (ajusta según cómo lo tengas en tu módulo)
	allowedSelfHosts = network.LoadAllowedHosts(config) 

	// Obtener IP (puedes mejorar esta lógica)
	prIP = network.GetPrivateIP()

	r := gin.Default()
	r.SetTrustedProxies(nil) // ← importante en producción

	r.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{"Status": "Up"})
    })

    r.GET("/meet", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "address":       prIP,
            "hostname":      config.GetString("meet.hostname"),
            "passphrase":    config.GetString("meet.passphrase"),
            "allowed_hosts": allowedSelfHosts,
        })
    })

	r.POST("/files/get", func(c *gin.Context) {
        filesGet.ReceiveFile(c, tempDir)
    })

    r.GET("/files/decrypt/:filename", func(c *gin.Context) {
        filesDecrypt.DecryptFile(c, tempDir, config)
    })

	port := ":42532" // puedes leerlo de config si quieres
	log.Printf("Iniciando servidor en http://%s%s", prIP, port)
	if err := r.Run(port); err != nil {
		log.Fatalf("Fallo al iniciar servidor: %v", err)
	}
}