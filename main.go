package main

import (
	"log"
	"os"
	"strings"

	"pos-go/config"
	database "pos-go/database/migrations"
	"pos-go/routes"
	"pos-go/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.ConnectDatabase()
	config.InitMidtrans()
	database.SeedAdmin()

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Static("/uploads", "./uploads")

	// Izinkan frontend lokal dan domain produksi. FRONTEND_URLS dapat berisi
	// beberapa origin yang dipisahkan koma untuk deployment lain.
	allowedOrigins := []string{
		"http://localhost:5173",
		"https://pos.habibiyasin.my.id",
	}

	frontendURLs := os.Getenv("FRONTEND_URLS")
	if frontendURLs == "" {
		frontendURLs = os.Getenv("FRONTEND_URL")
	}
	for _, frontendURL := range strings.Split(frontendURLs, ",") {
		frontendURL = strings.TrimRight(strings.TrimSpace(frontendURL), "/")
		if frontendURL != "" && !containsOrigin(allowedOrigins, frontendURL) {
			allowedOrigins = append(allowedOrigins, frontendURL)
		}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	routes.AuthRoutes(r)
	routes.CategoryRoutes(r)
	routes.MenuRoutes(r)
	routes.TransactionRoutes(r)

	r.GET("/ping", func(c *gin.Context) {
		utils.SuccessResponseOK(c, "API sukses berjalan", nil)
	})

	// Render memberikan port melalui environment variable PORT.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server berjalan di port %s", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Server gagal dijalankan:", err)
	}
}

func containsOrigin(origins []string, target string) bool {
	for _, origin := range origins {
		if origin == target {
			return true
		}
	}
	return false
}
