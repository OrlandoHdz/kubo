package main

import (
	"context"
	"log"

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

	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true, // Cambiar a false en producción
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:   []string{"Content-Length"},
		MaxAge:          12,
	}))

	// 3. Cargar Rutas
	routes.SetupRoutes(r, queries)

	log.Println("🚀 Kubo API corriendo en :8080")
	r.Run(":8080")
}
