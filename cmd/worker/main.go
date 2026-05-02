package main

import (
	"context"
	"log"

	"github.com/OrlandoHdz/kubo/internal/database"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/internal/services"
)

func main() {
	ctx := context.Background()

	// 1. Inicializar la conexión a la base de datos
	// Usamos la configuración existente
	pool, err := database.NuevoPool(ctx, "configs/db/database.yaml")
	if err != nil {
		log.Fatalf("Fallo crítico en base de datos: %v", err)
	}
	defer pool.Close()

	// 2. Instanciar queries y el servicio
	queries := db.New(pool)
	clienteService := services.NewClientesIntegracionService(queries)

	// 3. Ejecutar la sincronización
	// Ruta al archivo DBF de SAI ERP (basado en tu ejemplo previo)
	dbfPath := "/Users/orlando/Proyectos/Orlando/KUBO/BaseDatos_SAI/CLIENTES/CLIENTES.DBF"
	
	log.Printf("Iniciando proceso de sincronización desde: %s", dbfPath)
	
	err = clienteService.SincronizarClientesDesdeDBF(ctx, dbfPath)
	if err != nil {
		log.Fatalf("Error durante la sincronización: %v", err)
	}

	log.Println("Sincronización completada exitosamente.")
}
