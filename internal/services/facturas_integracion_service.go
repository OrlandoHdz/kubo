package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/OrlandoHdz/kubo/internal/db"
	dbf "github.com/SebastiaanKlippert/go-foxpro-dbf"
	"github.com/jackc/pgx/v5/pgtype"
)

type FacturasIntegracionService struct {
	queries *db.Queries
}

func NewFacturasIntegracionService(q *db.Queries) *FacturasIntegracionService {
	return &FacturasIntegracionService{queries: q}
}

func (s *FacturasIntegracionService) SincronizarFacturasDesdeDBF(ctx context.Context, dbfPath string) error {
	// 1. Abrimos el archivo DBF
	openedDbf, err := dbf.OpenFile(dbfPath, new(dbf.Win1250Decoder))
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo DBF de facturas: %v", err)
	}
	defer openedDbf.Close()

	totalRegistros := openedDbf.NumRecords()
	log.Printf("Iniciando sincronización de facturas. Registros totales en DBF: %d\n", totalRegistros)

	// // --- CAMBIO PARA PRUEBAS: Limitar a 15 registros ---
	// limitePrueba := uint32(15)
	// if totalRegistros > limitePrueba {
	// 	totalRegistros = limitePrueba
	// }
	// log.Printf("MODO PRUEBA: Iniciando sincronización limitada a los primeros %d registros.\n", totalRegistros)
	// // ----------------------------------------------------

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
				if strings.Contains(err.Error(), "REIMPRE") {
					log.Printf("Warning: REIMPRE field invalid at record %d – constructing partial map", i)
					// Build map manually, setting REIMPRE to nil
					registro = make(map[string]interface{})
					fields := openedDbf.Fields()
					rawRecord, recErr := openedDbf.RecordAt(i)
					if recErr != nil {
						log.Printf("Error retrieving raw record %d after REIMPRE parse failure: %v", i, recErr)
						continue
					}
					recordSlice := rawRecord.FieldSlice()
					for idx, field := range fields {
						if field.FieldName() == "REIMPRE" {
							registro[field.FieldName()] = nil
							continue
						}
						if idx < len(recordSlice) {
							registro[field.FieldName()] = recordSlice[idx]
						}
					}
				} else {
					log.Printf("Error al leer registro de factura %d: %v", i, err)
					continue
				}
			}

			// Solo loguear el primer registro para ver los campos
			if i == 0 {
				log.Printf("Campos encontrados en el primer registro de factura: %v", registro)
			}

			params := s.mapRegistroToParams(registro)

			_, err = s.queries.UpsertFacturaIntegracion(ctx, params)
			if err != nil {
				log.Printf("Error al insertar/actualizar factura %s: %v", params.NoFac, err)
				continue
			}
			contadorSincronizados++
		}
	}

	log.Printf("¡Sincronización de facturas terminada! Se procesaron %d facturas.\n", contadorSincronizados)
	return nil
}

func (s *FacturasIntegracionService) mapRegistroToParams(reg map[string]interface{}) db.UpsertFacturaIntegracionParams {
	// Notar que el campo A—O en el mapa DBF puede venir con caracteres extraños como "A\x14O" o similar.
	// Buscaremos dinámicamente si contiene "A" y "O" con caracteres intermedios.
	var anioVal interface{} = reg["A—O"]
	if anioVal == nil {
		// Búsqueda fallback por si el encoding cambia el caracter del medio
		for k, v := range reg {
			if len(k) == 3 && k[0] == 'A' && k[2] == 'O' {
				anioVal = v
				break
			}
		}
	}

	return db.UpsertFacturaIntegracionParams{
		NoFac:     toRequiredText(reg["NO_FAC"]),
		NoPed:     toText(reg["NO_PED"]),
		CveCte:    toInt4(reg["CVE_CTE"]),
		CveAge:    toInt4(reg["CVE_AGE"]),
		FaltaFac:  toTimestamp(reg["FALTA_FAC"]),
		StatusFac: toText(reg["STATUS_FAC"]),
		SubtFac:   toNumeric(reg["SUBT_FAC"]),
		Descuento: toNumeric(reg["DESCUENTO"]),
		Descue:    toNumeric(reg["DESCUE"]),
		TotalFac:  toNumeric(reg["TOTAL_FAC"]),
		SaldoFac:  toNumeric(reg["SALDO_FAC"]),
		FPago:     toTimestamp(reg["F_PAGO"]),
		Contrarec: toText(reg["CONTRAREC"]),
		Lugar:     toText(reg["LUGAR"]),
		CveFactu:  toText(reg["CVE_FACTU"]),
		CveMon:    toText(reg["CVE_MON"]),
		TipCam:    toNumeric(reg["TIP_CAM"]),
		SaldoFac2: toNumeric(reg["SALDO_FAC2"]),
		Staley:    toText(reg["STALEY"]),
		CveSuc:    toInt4(reg["CVE_SUC"]),
		Mes:       toInt4(reg["MES"]),
		Anio:      toInt4(anioVal),
		Usuario:   toText(reg["USUARIO"]),
		Trans:     toText(reg["TRANS"]),
		Staley2:   toText(reg["STALEY2"]),
		PedInt:    toText(reg["PED_INT"]),
		ComCob:    toNumeric(reg["COM_COB"]),
		Cierre:    toTimestamp(reg["CIERRE"]),
		UTipCam:   toNumeric(reg["U_TIP_CAM"]),
		CveAge2:   toInt4(reg["CVE_AGE2"]),
		ComCob2:   toNumeric(reg["COM_COB2"]),
		DinCom:    toNumeric(reg["DIN_COM"]),
		DinCom2:   toNumeric(reg["DIN_COM2"]),
		FechSal:   toTimestamp(reg["FECH_SAL"]),
		FechEmb:   toTimestamp(reg["FECH_EMB"]),
		CveFlet:   toText(reg["CVE_FLET"]),
		TotFlet:   toNumeric(reg["TOT_FLET"]),
		TotEnv:    toNumeric(reg["TOT_ENV"]),
		PrecFlet:  toNumeric(reg["PREC_FLET"]),
		PrvReal:   toText(reg["PRV_REAL"]),
		TimpCar:   toNumeric(reg["TIMP_CAR"]),
		Numpol:    toText(reg["NUMPOL"]),
		Numpolcan: toText(reg["NUMPOLCAN"]),
		ImpCarac:  toNumeric(reg["IMP_CARAC"]),
		Pesotot:   toNumeric(reg["PESOTOT"]),
		RetivaFac: toNumeric(reg["RETIVA_FAC"]),
		Deposito:  toNumeric(reg["DEPOSITO"]),
		SucDepo:   toText(reg["SUC_DEPO"]),
		Status2:   toText(reg["STATUS_2"]),
		Cvedirent: toText(reg["CVEDIRENT"]),
		Reimpre:   toInt4(reg["REIMPRE"]),
		Descue2:   toNumeric(reg["DESCUE2"]),
		Descue3:   toNumeric(reg["DESCUE3"]),
		Descue4:   toNumeric(reg["DESCUE4"]),
		NoPeda:    toText(reg["NO_PEDA"]),
		CveSuca:   toText(reg["CVE_SUCA"]),
		CveFactua: toText(reg["CVE_FACTUA"]),
		CveEntre:  toText(reg["CVE_ENTRE"]),
		Ieps:      toNumeric(reg["IEPS"]),
	}
}

func toRequiredText(val interface{}) string {
	if val == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", val))
}

// func toTimestamp(val interface{}) pgtype.Timestamp {
// 	if val == nil {
// 		return pgtype.Timestamp{Valid: false}
// 	}
// 	switch v := val.(type) {
// 	case time.Time:
// 		// Convertir a UTC y establecer la hora a medianoche
// 		utcTime := time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.UTC)
// 		return pgtype.Timestamp{Time: utcTime, Valid: true}
// 	case string:
// 		vStr := strings.TrimSpace(v)
// 		if vStr == "" {
// 			return pgtype.Timestamp{Valid: false}
// 		}
// 		formats := []string{
// 			"2006-01-02 15:04:05",
// 			"2006-01-02T15:04:05Z07:00",
// 			"2006-01-02",
// 		}
// 		for _, f := range formats {
// 			if parsed, err := time.Parse(f, vStr); err == nil {
// 				return pgtype.Timestamp{Time: parsed, Valid: true}
// 			}
// 		}
// 	}
// 	return pgtype.Timestamp{Valid: false}
// }

// toInt4 converts various numeric types to pgtype.Int4, returning invalid on parse errors.
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
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return pgtype.Int4{Valid: false}
		}
		i = int32(parsed)
	default:
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: i, Valid: true}
}
