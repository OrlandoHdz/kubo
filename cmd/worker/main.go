package main

import (
	"context"
	"flag"
	"log"

	"github.com/OrlandoHdz/kubo/internal/database"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/internal/services"
)

func main() {
	ctx := context.Background()

	// Parse command‑line flags for DBF file locations
	clientesPath := flag.String("clientes", "", "Path to the CLIENTES DBF file")
	facturasPath := flag.String("facturas", "", "Path to the FACTURAS DBF file")
	existenciasPath := flag.String("existencias", "", "Path to the EXISTENCIAS DBF file")
	flag.Parse()

	// 1. Inicializar la conexión a la base de datos
	pool, err := database.NuevoPool(ctx, "configs/db/database.yaml")
	if err != nil {
		log.Fatalf("Fallo crítico en base de datos: %v", err)
	}
	defer pool.Close()

	// 2. Instanciar queries y los servicios
	queries := db.New(pool)
	clienteService := services.NewClientesIntegracionService(queries)
	existenciasService := services.NewExistenciasIntegracionService(queries)
	facturaService := services.NewFacturasIntegracionService(queries)

	// Track whether we performed any synchronization
	synced := false

	// 3. Sincronizar clientes si se provee la ruta
	if *clientesPath != "" {
		log.Printf("Iniciando sincronización de clientes desde: %s", *clientesPath)
		if err := clienteService.SincronizarClientesDesdeDBF(ctx, *clientesPath); err != nil {
			log.Fatalf("Error al sincronizar clientes: %v", err)
		}
		synced = true
	}

	// 4. Sincronizar facturas si se provee la ruta
	if *facturasPath != "" {
		log.Printf("Iniciando sincronización de facturas desde: %s", *facturasPath)
		if err := facturaService.SincronizarFacturasDesdeDBF(ctx, *facturasPath); err != nil {
			log.Fatalf("Error al sincronizar facturas: %v", err)
		}
		synced = true
	}

	// 5. Sincronizar existencias si se provee la ruta
	if *existenciasPath != "" {
		log.Printf("Iniciando sincronización de existencias desde: %s", *existenciasPath)
		if err := existenciasService.SincronizarExistenciasDesdeDBF(ctx, *existenciasPath); err != nil {
			log.Fatalf("Error al sincronizar existencias: %v", err)
		}
		synced = true
	}

	if !synced {
		log.Fatalf("Se debe proporcionar al menos una de las rutas DBF mediante -clientes, -facturas o -existencias")
	}

	log.Println("Sincronización completada exitosamente.")
}
