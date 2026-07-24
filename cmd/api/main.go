package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/OrlandoHdz/kubo/internal/api/routes"
	"github.com/OrlandoHdz/kubo/internal/database"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/email"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
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
	r.MaxMultipartMemory = 100 << 20 // 100 MB

	// Configuración de CORS simplificada y abierta para desarrollo
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:   []string{"Content-Length"},
		MaxAge:          12 * time.Hour,
	}))

	// 3. Cargar Config OSS
	ossCfg, err := utils.NewConfigOSS("configs/cloud/alibaba.yaml")
	if err != nil {
		log.Fatalf("Error cargando config OSS: %v", err)
	}

	// 4. Cargar Config Email
	emailCfg := &email.Config{}
	if data, err := os.ReadFile("configs/cloud/email.yaml"); err == nil {
		if err := yaml.Unmarshal(data, emailCfg); err != nil {
			log.Printf("Error parseando config email: %v (correo deshabilitado)", err)
			emailCfg = nil
		}
	} else {
		log.Printf("Archivo email.yaml no encontrado: %v (correo deshabilitado)", err)
		emailCfg = nil
	}

	// 5. Cargar Rutas
	routes.SetupRoutes(r, queries, pool, ossCfg, emailCfg)

	log.Println("🚀 Kubo API corriendo en :8080")
	r.Run(":8080")
}
