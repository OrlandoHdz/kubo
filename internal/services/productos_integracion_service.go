package services

import (
	"context"
	"fmt"
	"log"

	"github.com/OrlandoHdz/kubo/internal/db"
	dbf "github.com/SebastiaanKlippert/go-foxpro-dbf"
)

type ProductosIntegracionService struct {
	queries *db.Queries
}

func NewProductosIntegracionService(q *db.Queries) *ProductosIntegracionService {
	return &ProductosIntegracionService{queries: q}
}

// SincronizarProductosDesdeDBF syncs product records from a DBF file to Postgres.
func (s *ProductosIntegracionService) SincronizarProductosDesdeDBF(ctx context.Context, dbfPath string) error {
	openedDbf, err := dbf.OpenFile(dbfPath, new(dbf.Win1250Decoder))
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo DBF de productos: %v", err)
	}
	defer openedDbf.Close()

	totalRegistros := openedDbf.NumRecords()
	log.Printf("Iniciando sincronización de productos. Registros totales en DBF: %d\n", totalRegistros)

	contadorSincronizados := 0
	var i uint32
	for i = 0; i < totalRegistros; i++ {
		deleted, err := openedDbf.DeletedAt(i)
		if err != nil {
			continue
		}
		if deleted {
			continue
		}

		registro, err := openedDbf.RecordToMap(i)
		if err != nil {
			log.Printf("Error al leer registro de producto %d: %v", i, err)
			continue
		}

		if i == 0 {
			log.Printf("Campos encontrados en el primer registro de producto: %v", registro)
		}

		params := s.mapRegistroToParams(registro)

		_, err = s.queries.UpsertProducto(ctx, params)
		if err != nil {
			log.Printf("Error al insertar/actualizar producto %s: %v", params.CveProd.String, err)
			continue
		}
		contadorSincronizados++
	}

	log.Printf("¡Sincronización de productos terminada! Se procesaron %d registros.\n", contadorSincronizados)
	return nil
}

func (s *ProductosIntegracionService) mapRegistroToParams(reg map[string]interface{}) db.UpsertProductoParams {
	return db.UpsertProductoParams{
		CseProd:   toPGText(reg["CSE_PROD"]),
		CveProd:   toPGText(reg["CVE_PROD"]),
		NomProd:   toPGText(reg["NOM_PROD"]),
		DescProd:  toPGText(reg["DESC_PROD"]),
		UniMed:    toPGText(reg["UNI_MED"]),
		UniMedP:   toPGText(reg["UNI_MED_P"]),
		CostoProd: toNumeric(reg["COSTO_PROD"]),
		FActCto:   toTimestamp(reg["F_ACT_CTO"]),
		FActPre:   toTimestamp(reg["F_ACT_PRE"]),
		CtoEnt:    toNumeric(reg["CTO_ENT"]),
		FecEnt:    toTimestamp(reg["FEC_ENT"]),
		FecAnt:    toTimestamp(reg["FEC_ANT"]),
	}
}
