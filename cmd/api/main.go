package main

import (
	"context"
	"log"
	"time"

	"github.com/OrlandoHdz/kubo/internal/api/routes"
	"github.com/OrlandoHdz/kubo/internal/database"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Setup DB
	ctx := context.Background()
	pool, err := database.NuevoPool(ctx, "configs/db/database.yaml")
	if err != nil {
		log.Fatalf("Error de conexión DB: %v", err)
	}
	defer pool.Close()
	queries := db.New(pool)

	// 2. Setup Server
	r := gin.Default()

	// Configuración de CORS simplificada y abierta para desarrollo
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:   []string{"Content-Length"},
		MaxAge:          12 * time.Hour,
	}))

	// 3. Cargar Rutas
	routes.SetupRoutes(r, queries)

	log.Println("🚀 Kubo API corriendo en :8080")
	r.Run(":8080")
}
