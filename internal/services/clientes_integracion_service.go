package services

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/OrlandoHdz/kubo/internal/db"
	dbf "github.com/SebastiaanKlippert/go-foxpro-dbf"
	"github.com/jackc/pgx/v5/pgtype"
)

type ClientesIntegracionService struct {
	queries *db.Queries
}

func NewClientesIntegracionService(q *db.Queries) *ClientesIntegracionService {
	return &ClientesIntegracionService{queries: q}
}

func (s *ClientesIntegracionService) SincronizarClientesDesdeDBF(ctx context.Context, dbfPath string) error {
	// 1. Abrimos el archivo DBF
	openedDbf, err := dbf.OpenFile(dbfPath, new(dbf.Win1250Decoder))
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo DBF: %v", err)
	}
	defer openedDbf.Close()

	totalRegistros := openedDbf.NumRecords()
	log.Printf("Iniciando sincronización. Registros totales en DBF: %d\n", totalRegistros)

	contadorSincronizados := 0
	var i uint32
	for i = 0; i < totalRegistros; i++ {
		deleted, err := openedDbf.DeletedAt(i)
		if err != nil {
			continue
		}

		if !deleted {
			registro, err := openedDbf.RecordToMap(i)
			if err != nil {
				log.Printf("Error al leer registro %d: %v", i, err)
				continue
			}

			// Solo loguear el primer registro para ver los campos
			if i == 0 {
				log.Printf("Campos encontrados en el primer registro: %v", registro)
			}

			params := s.mapRegistroToParams(registro)

			_, err = s.queries.UpsertClienteIntegracion(ctx, params)
			if err != nil {
				log.Printf("Error al insertar/actualizar cliente %v: %v", params.CveCte, err)
				continue
			}
			contadorSincronizados++
		}
	}

	log.Printf("¡Sincronización terminada! Se procesaron %d clientes.\n", contadorSincronizados)
	return nil
}

func (s *ClientesIntegracionService) mapRegistroToParams(reg map[string]interface{}) db.UpsertClienteIntegracionParams {
	return db.UpsertClienteIntegracionParams{
		CveCte:    toInt4(reg["CVE_CTE"]),
		NomCte:    toText(reg["NOM_CTE"]),
		DirCte:    toText(reg["DIR_CTE"]),
		ColCte:    toText(reg["COL_CTE"]),
		CdCte:     toText(reg["CD_CTE"]),
		EdoCte:    toText(reg["EDO_CTE"]),
		RfcCte:    toText(reg["RFC_CTE"]),
		CpCte:     toText(reg["CP_CTE"]),
		Contacto:  toText(reg["CONTACTO"]),
		Tel1Cte:   toText(reg["TEL1_CTE"]),
		LimCre:    toNumeric(reg["LIM_CRE"]),
		DiaCre:    toInt4(reg["DIA_CRE"]),
		CveAge:    toInt4(reg["CVE_AGE"]),
		LadaCte:   toText(reg["LADA_CTE"]),
		CveZon:    toInt4(reg["CVE_ZON"]),
		CveSub:    toInt4(reg["CVE_SUB"]),
		CveCan:    toInt4(reg["CVE_CAN"]),
		Cuentacon: toText(reg["CUENTACON"]),
		Contado:   toInt4(reg["CONTADO"]),
		PaisCte:   toText(reg["PAIS_CTE"]),
		EmailCte:  toText(reg["EMAIL_CTE"]),
	}
}

func toInt4(val interface{}) pgtype.Int4 {
	if val == nil {
		return pgtype.Int4{Valid: false}
	}
	
	var i int32
	switch v := val.(type) {
	case int32:
		i = v
	case int:
		i = int32(v)
	case int64:
		i = int32(v)
	case float64:
		i = int32(v)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(v))
		i = int32(parsed)
	default:
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: i, Valid: true}
}

func toText(val interface{}) pgtype.Text {
	if val == nil {
		return pgtype.Text{Valid: false}
	}
	str := strings.TrimSpace(fmt.Sprintf("%v", val))
	return pgtype.Text{String: str, Valid: true}
}

func toNumeric(val interface{}) pgtype.Numeric {
	if val == nil {
		return pgtype.Numeric{Valid: false}
	}
	var f float64
	switch v := val.(type) {
	case float64:
		f = v
	case int32:
		f = float64(v)
	case int64:
		f = float64(v)
	case int:
		f = float64(v)
	case string:
		f, _ = strconv.ParseFloat(strings.TrimSpace(v), 64)
	default:
		return pgtype.Numeric{Valid: false}
	}

	num := pgtype.Numeric{}
	num.Int = big.NewInt(int64(f * 100))
	num.Exp = -2
	num.Valid = true
	return num
}
