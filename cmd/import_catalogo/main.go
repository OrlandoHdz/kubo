package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/OrlandoHdz/kubo/internal/database"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/internal/services"
)

func main() {
	rutaExcel := flag.String("archivo", "", "Ruta del archivo Excel (.xlsx) a importar")
	rutaConfig := flag.String("config", "configs/db/database.yaml", "Ruta de la configuración de la base de datos")
	flag.Parse()

	if *rutaExcel == "" {
		log.Fatal("debes indicar la ruta del Excel con -archivo")
	}

	ctx := context.Background()
	pool, err := database.NuevoPool(ctx, *rutaConfig)
	if err != nil {
		log.Fatalf("error conectando a la base de datos: %v", err)
	}
	defer pool.Close()

	servicio := services.NewCatalogoExcelService(db.New(pool), pool)
	resumen, err := servicio.ImportCatalogFromExcel(ctx, *rutaExcel)
	if err != nil {
		log.Fatalf("error importando catálogo: %v", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(resumen); err != nil {
		log.Fatalf("error imprimiendo el resumen: %v", err)
	}
}
