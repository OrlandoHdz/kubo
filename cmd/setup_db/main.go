package main

import (
	"context"
	"log"

	"github.com/OrlandoHdz/kubo/internal/database"
	"github.com/OrlandoHdz/kubo/internal/db"
)

func main() {
	ctx := context.Background()

	// Inicializar la conexión usando el nuevo helper
	pool, err := database.NuevoPool(ctx, "configs/db/database.yaml")
	if err != nil {
		log.Fatalf("Fallo crítico en base de datos: %v", err)
	}
	defer pool.Close()

	// Instanciar tus queries generadas por sqlc
	queries := db.New(pool)

	// Ejemplo: Listar clientes activos para el panel administrativo [cite: 224]
	clientes, err := queries.ListarClientesActivos(ctx)
	if err != nil {
		log.Printf("Error al consultar clientes: %v", err)
	}

	log.Printf("Conexión exitosa. Clientes cargados: %d", len(clientes))
}
