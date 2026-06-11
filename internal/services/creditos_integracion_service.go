package services

import (
	"context"
	"fmt"
	"log"

	"github.com/OrlandoHdz/kubo/internal/db"
	dbf "github.com/SebastiaanKlippert/go-foxpro-dbf"
)

type CreditosIntegracionService struct {
	queries *db.Queries
}

func NewCreditosIntegracionService(q *db.Queries) *CreditosIntegracionService {
	return &CreditosIntegracionService{queries: q}
}

// SincronizarCreditosDesdeDBF syncs credit records (notas de crédito) from a DBF file to Postgres.
func (s *CreditosIntegracionService) SincronizarCreditosDesdeDBF(ctx context.Context, dbfPath string) error {
	openedDbf, err := dbf.OpenFile(dbfPath, new(dbf.Win1250Decoder))
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo DBF de créditos: %v", err)
	}
	defer openedDbf.Close()

	totalRegistros := openedDbf.NumRecords()
	log.Printf("Iniciando sincronización de créditos. Registros totales en DBF: %d\n", totalRegistros)

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
			log.Printf("Error al leer registro de crédito %d: %v", i, err)
			continue
		}

		if i == 0 {
			log.Printf("Campos encontrados en el primer registro de crédito: %v", registro)
		}

		params := s.mapRegistroToParams(registro)

		_, err = s.queries.UpsertCredito(ctx, params)
		if err != nil {
			log.Printf("Error al insertar/actualizar crédito %s (cve_factu=%v): %v", params.NoFac.String, params.CveFactu.String, err)
			continue
		}
		contadorSincronizados++
	}

	log.Printf("¡Sincronización de créditos terminada! Se procesaron %d registros.\n", contadorSincronizados)
	return nil
}

func (s *CreditosIntegracionService) mapRegistroToParams(reg map[string]interface{}) db.UpsertCreditoParams {
	// Notar que el campo A—O (Año) en el mapa DBF puede venir con caracteres extraños
	var aqoVal interface{} = reg["AQO"]
	if aqoVal == nil {
		aqoVal = reg["A—O"]
	}
	if aqoVal == nil {
		for k, v := range reg {
			if len(k) == 3 && k[0] == 'A' && (k[2] == 'O' || k[2] == 'Q') {
				aqoVal = v
				break
			}
		}
	}

	return db.UpsertCreditoParams{
		CveDda:    toPGText(reg["CVE_DDA"]),
		NoNota:    toPGText(reg["NO_NOTA"]),
		TipNot:    toPGText(reg["TIP_NOT"]),
		Fecha:     toTimestamp(reg["FECHA"]),
		NoFac:     toPGText(reg["NO_FAC"]),
		NoCliente: toInt4(reg["NO_CLIENTE"]),
		NoAgente:  toInt4(reg["NO_AGENTE"]),
		NoEstado:  toPGText(reg["NO_ESTADO"]),
		TotImp:    toNumeric(reg["TOT_IMP"]),
		Subtotal:  toNumeric(reg["SUBTOTAL"]),
		DescNot:   toNumeric(reg["DESC_NOT"]),
		TotNota:   toNumeric(reg["TOT_NOTA"]),
		Num:       toInt4(reg["NUM"]),
		Saldo:     toNumeric(reg["SALDO"]),
		TotDes:    toNumeric(reg["TOT_DES"]),
		Lugar:     toPGText(reg["LUGAR"]),
		CveFactu:  toPGText(reg["CVE_FACTU"]),
		CveMon:    toInt4(reg["CVE_MON"]),
		TipCam:    toNumeric(reg["TIP_CAM"]),
		CveSuc:    toPGText(reg["CVE_SUC"]),
		Mes:       toPGText(reg["MES"]),
		Aqo:       toPGText(aqoVal),
		Usuario:   toInt4(reg["USUARIO"]),
		Trans:     toInt4(reg["TRANS"]),
	}
}
