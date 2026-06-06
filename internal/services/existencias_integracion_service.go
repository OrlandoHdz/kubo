package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	dbf "github.com/SebastiaanKlippert/go-foxpro-dbf"
	"github.com/jackc/pgx/v5/pgtype"
)

type ExistenciasIntegracionService struct {
	queries *db.Queries
}

func NewExistenciasIntegracionService(q *db.Queries) *ExistenciasIntegracionService {
	return &ExistenciasIntegracionService{queries: q}
}

// SincronizarExistenciasDesdeDBF syncs existence records from a DBF file to Postgres.
func (s *ExistenciasIntegracionService) SincronizarExistenciasDesdeDBF(ctx context.Context, dbfPath string) error {
	openedDbf, err := dbf.OpenFile(dbfPath, new(dbf.Win1250Decoder))
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo DBF de existencias: %v", err)
	}
	defer openedDbf.Close()

	totalRegistros := openedDbf.NumRecords()
	log.Printf("Iniciando sincronización de existencias. Registros totales en DBF: %d\n", totalRegistros)

	// Optional limit for testing – remove or adjust as needed
	limitePrueba := uint32(0) // 0 = no limit
	if limitePrueba > 0 && totalRegistros > limitePrueba {
		totalRegistros = limitePrueba
		log.Printf("Modo prueba: limitando a %d registros.\n", totalRegistros)
	}

	contador := 0
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
			log.Printf("Error leyendo registro %d: %v", i, err)
			continue
		}
		if i == 0 {
			log.Printf("Campos del primer registro de existencias: %v", registro)
		}
		params := s.mapRegistroToParams(registro)
		_, err = s.queries.UpsertExistenciaIntegracion(ctx, params)
		if err != nil {
			log.Printf("Error upsert existencia (cve_prod=%s, lugar=%s): %v", params.CveProd, params.Lugar, err)
			continue
		}
		contador++
	}
	log.Printf("¡Sincronización de existencias finalizada! %d registros procesados.\n", contador)
	return nil
}

func (s *ExistenciasIntegracionService) mapRegistroToParams(reg map[string]interface{}) db.UpsertExistenciaIntegracionParams {
	// Helper conversion functions similar to facturas service
	toText := func(v interface{}) string {
		if v == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	toNumeric := func(v interface{}) pgtype.Numeric {
		var n pgtype.Numeric
		if v == nil {
			return n // Invalid by default
		}
		// Use Scan to populate the Numeric from various possible DBF value types.
		if err := n.Scan(strings.TrimSpace(fmt.Sprintf("%v", v))); err != nil {
			return pgtype.Numeric{Valid: false}
		}
		return n
	}
	toTimestamp := func(v interface{}) pgtype.Timestamp {
		if v == nil {
			return pgtype.Timestamp{Valid: false}
		}
		switch t := v.(type) {
		case time.Time:
			utc := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			return pgtype.Timestamp{Time: utc, Valid: true}
		case string:
			str := strings.TrimSpace(t)
			if str == "" {
				return pgtype.Timestamp{Valid: false}
			}
			layouts := []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00", "2006-01-02"}
			for _, l := range layouts {
				if parsed, err := time.Parse(l, str); err == nil {
					return pgtype.Timestamp{Time: parsed, Valid: true}
				}
			}
			return pgtype.Timestamp{Valid: false}
		default:
			return pgtype.Timestamp{Valid: false}
		}
	}

	return db.UpsertExistenciaIntegracionParams{
		CseProd:    toTextt(reg["CSE_PROD"]),
		CveProd:    toText(reg["CVE_PROD"]),
		Lugar:      toText(reg["LUGAR"]),
		Existencia: toNumeric(reg["EXISTENCIA"]),
		MedProd:    toNumeric(reg["MED_PROD"]),
		FechUmod:   toTimestamp(reg["FECH_UMOD"]),
		InvIni:     pgtype.Int8{Int64: parseInt64(reg["INV_INI"]), Valid: true},
		Lote:       toTextt(reg["LOTE"]),
		FechLote:   toTimestamp(reg["FECH_LOTE"]),
		RefLote:    toTextt(reg["REF_LOTE"]),
		CostoProm:  toNumeric(reg["COSTO_PROM"]),
		NewMed:     toTextt(reg["NEW_MED"]),
		Costuepeps: toNumeric(reg["COSTUEPEPS"]),
		Costoprom2: toNumeric(reg["COSTOPROM2"]),
	}
}

// func parseInt64(v interface{}) int64 {
// 	if v == nil {
// 		return 0
// 	}
// 	switch val := v.(type) {
// 	case int64:
// 		return val
// 	case int:
// 		return int64(val)
// 	case string:
// 		i, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
// 		if err != nil {
// 			return 0
// 		}
// 		return i
// 	default:
// 		return 0
// 	}
// }

func toTextt(val interface{}) pgtype.Text {
	if val == nil {
		return pgtype.Text{Valid: false}
	}
	str := strings.TrimSpace(fmt.Sprintf("%v", val))
	return pgtype.Text{String: str, Valid: true}
}

// func toText(val interface{}) pgtype.Text {
// 	if val == nil {
// 		return pgtype.Text{Valid: false}
// 	}
// 	str := strings.TrimSpace(fmt.Sprintf("%v", val))
// 	return pgtype.Text{String: str, Valid: true}
// }
