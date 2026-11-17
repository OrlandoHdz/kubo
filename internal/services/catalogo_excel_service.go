package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"
)

const (
	catalogoUnidadMedida = "PIEZA"
	catalogoCategoria    = "MIG"
)

// CatalogoExcelService importa productos de un archivo Excel al catálogo
// (productos_padre + productos_variantes).
type CatalogoExcelService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewCatalogoExcelService(q *db.Queries, pool *pgxpool.Pool) *CatalogoExcelService {
	return &CatalogoExcelService{queries: q, pool: pool}
}

// FilaIgnorada describe una fila que no pudo importarse.
type FilaIgnorada struct {
	Fila   int    `json:"fila"`
	Codigo string `json:"codigo,omitempty"`
	Motivo string `json:"motivo"`
}

// CatalogoImportResumen es el resultado de una importación de catálogo.
type CatalogoImportResumen struct {
	FilasProcesadas       int            `json:"filas_procesadas"`
	Insertadas            int            `json:"insertadas"`
	Actualizadas          int            `json:"actualizadas"`
	OmitidasNoIntegracion int            `json:"omitidas_no_integracion"`
	OmitidasParseo        int            `json:"omitidas_parseo"`
	OmitidasConflicto     int            `json:"omitidas_conflicto"`
	Errores               int            `json:"errores"`
	Ignoradas             []FilaIgnorada `json:"ignoradas,omitempty"`
	ReporteIgnorados      string         `json:"reporte_ignorados,omitempty"`
	Advertencias          []string       `json:"advertencias,omitempty"`
}

type columnasCatalogo struct {
	codigo         int
	productName    int
	descProdTitulo int
	usuario        int
	mostrador      int
	volumen        int
	tipo           int
	presentacion   int
}

type filaCatalogo struct {
	codigo             string
	titulo             string
	descripcion        string
	precioDistribuidor float64
	precioMostrador    float64
	volumen            string
	tipo               string
	multiplos          int32
}

type resultadoFila int

const (
	resultadoInsertada resultadoFila = iota
	resultadoActualizada
	resultadoNoIntegracion
	resultadoParseo
	resultadoConflicto
	resultadoError
)

type errorFila struct {
	resultado resultadoFila
	motivo    string
}

func (e *errorFila) Error() string { return e.motivo }

// ImportCatalogFromExcel procesa la primera hoja del archivo indicado.
// Cada fila se ejecuta en su propia transacción; un error en una fila no
// revierte las filas ya importadas.
func (s *CatalogoExcelService) ImportCatalogFromExcel(ctx context.Context, path string) (*CatalogoImportResumen, error) {
	if s == nil || s.queries == nil || s.pool == nil {
		return nil, errors.New("servicio de importación de catálogo no inicializado")
	}

	archivo, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir el archivo Excel: %w", err)
	}
	defer archivo.Close()

	hojas := archivo.GetSheetList()
	if len(hojas) == 0 {
		return nil, errors.New("el archivo Excel no contiene hojas")
	}

	filas, err := archivo.GetRows(hojas[0])
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer la hoja %q: %w", hojas[0], err)
	}
	if len(filas) == 0 {
		return nil, errors.New("la hoja del Excel está vacía")
	}

	indiceEncabezado, columnas, advertencias, err := detectarEncabezadoCatalogo(filas)
	if err != nil {
		return nil, err
	}

	resumen := &CatalogoImportResumen{Advertencias: advertencias}
	for indice, fila := range filas[indiceEncabezado+1:] {
		if filaVacia(fila) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return resumen, fmt.Errorf("importación cancelada: %w", err)
		}

		resumen.FilasProcesadas++
		numeroFila := indiceEncabezado + indice + 2 // índices base 0 + número de fila de Excel base 1

		datos, err := mapearFilaCatalogo(fila, columnas)
		if err != nil {
			resumen.OmitidasParseo++
			resumen.agregarIgnorada(numeroFila, valorColumna(fila, columnas.codigo), err.Error())
			continue
		}

		resultado, err := s.importarFilaCatalogo(ctx, datos)
		if err != nil {
			var filaErr *errorFila
			if errors.As(err, &filaErr) {
				switch filaErr.resultado {
				case resultadoNoIntegracion:
					resumen.OmitidasNoIntegracion++
				case resultadoParseo:
					resumen.OmitidasParseo++
				case resultadoConflicto:
					resumen.OmitidasConflicto++
				default:
					resumen.Errores++
				}
				resumen.agregarIgnorada(numeroFila, datos.codigo, filaErr.motivo)
				log.Printf("catálogo Excel: fila %d (código %q) omitida: %s", numeroFila, datos.codigo, filaErr.motivo)
				continue
			}
			return resumen, fmt.Errorf("error inesperado importando la fila %d: %w", numeroFila, err)
		}

		switch resultado {
		case resultadoInsertada:
			resumen.Insertadas++
		case resultadoActualizada:
			resumen.Actualizadas++
		}
	}

	if len(resumen.Ignoradas) > 0 {
		rutaReporte, err := escribirReporteIgnorados(path, resumen.Ignoradas)
		if err != nil {
			resumen.Advertencias = append(resumen.Advertencias, "no se pudo escribir el reporte de ignorados: "+err.Error())
		} else {
			resumen.ReporteIgnorados = rutaReporte
		}
	}

	log.Printf(
		"catálogo Excel: %d filas procesadas, %d insertadas, %d actualizadas, %d sin integración, %d con errores de parseo, %d conflictos, %d errores",
		resumen.FilasProcesadas,
		resumen.Insertadas,
		resumen.Actualizadas,
		resumen.OmitidasNoIntegracion,
		resumen.OmitidasParseo,
		resumen.OmitidasConflicto,
		resumen.Errores,
	)

	return resumen, nil
}

func (r *CatalogoImportResumen) agregarIgnorada(fila int, codigo, motivo string) {
	r.Ignoradas = append(r.Ignoradas, FilaIgnorada{
		Fila:   fila,
		Codigo: strings.TrimSpace(codigo),
		Motivo: motivo,
	})
}

func (s *CatalogoExcelService) importarFilaCatalogo(ctx context.Context, datos filaCatalogo) (resultadoFila, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return resultadoError, &errorFila{resultado: resultadoError, motivo: "no se pudo iniciar la transacción: " + err.Error()}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.queries.WithTx(tx)

	productoIntegracion, err := qtx.GetProductoByClave(ctx, pgtype.Text{String: datos.codigo, Valid: true})
	if errors.Is(err, pgx.ErrNoRows) {
		productoIntegracion, err = qtx.GetProductoByClaveCaseInsensitive(ctx, datos.codigo)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return resultadoNoIntegracion, &errorFila{
			resultado: resultadoNoIntegracion,
			motivo:    "el código no existe en productos_integracion.cve_prod",
		}
	}
	if err != nil {
		return resultadoError, &errorFila{resultado: resultadoError, motivo: "error consultando productos_integracion: " + err.Error()}
	}

	cveCanonico := strings.TrimSpace(productoIntegracion.CveProd.String)
	if cveCanonico == "" {
		cveCanonico = datos.codigo
	}

	padre, err := qtx.GetProductoPadreByCveProd(ctx, pgtype.Text{String: cveCanonico, Valid: true})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		eliminado, errEliminado := qtx.ExisteProductoPadreEliminadoPorCve(ctx, pgtype.Text{String: cveCanonico, Valid: true})
		if errEliminado != nil {
			return resultadoError, &errorFila{resultado: resultadoError, motivo: "error validando producto eliminado: " + errEliminado.Error()}
		}
		if eliminado {
			return resultadoConflicto, &errorFila{
				resultado: resultadoConflicto,
				motivo:    "existe un producto padre eliminado con este cve_prod_integracion",
			}
		}

		descripcion := datos.descripcion
		if descripcion == "" {
			descripcion = strings.TrimSpace(productoIntegracion.DescProd.String)
		}
		if descripcion == "" {
			descripcion = datos.titulo
		}

		padre, err = qtx.CrearProductoPadre(ctx, db.CrearProductoPadreParams{
			CveProdIntegracion: pgtype.Text{String: cveCanonico, Valid: true},
			Descripcion:        textoCatalogo(descripcion, 200),
			Titulo:             textoCatalogo(datos.titulo, 400),
		})
		if err != nil {
			return resultadoError, &errorFila{resultado: resultadoError, motivo: "error creando producto padre: " + err.Error()}
		}
	case err != nil:
		return resultadoError, &errorFila{resultado: resultadoError, motivo: "error consultando producto padre: " + err.Error()}
	}

	variante, err := qtx.GetVarianteBySKU(ctx, datos.codigo)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		_, err = qtx.CrearVariante(ctx, db.CrearVarianteParams{
			PadreID:            pgtype.Int4{Int32: padre.ID, Valid: true},
			Sku:                datos.codigo,
			PrecioDistribuidor: utils.ToNumeric(datos.precioDistribuidor),
			PrecioLista:        utils.ToNumeric(datos.precioMostrador),
			PrecioPublico:      utils.ToNumeric(datos.precioMostrador),
			StockActual:        0,
			UnidadMedida:       catalogoUnidadMedida,
			LeadTimeDias:       pgtype.Int4{Int32: 2, Valid: true},
			Categoria:          pgtype.Text{String: catalogoCategoria, Valid: true},
			Subgrupo:           textoCatalogo(datos.volumen, 50),
			Modelo:             textoCatalogo(datos.tipo, 50),
			Multiplos:          pgtype.Int4{Int32: datos.multiplos, Valid: true},
			PermitirBackorder:  pgtype.Bool{Bool: true, Valid: true},
		})
		if err != nil {
			return resultadoError, &errorFila{resultado: resultadoError, motivo: "error creando variante: " + err.Error()}
		}
		if err := tx.Commit(ctx); err != nil {
			return resultadoError, &errorFila{resultado: resultadoError, motivo: "error confirmando la transacción: " + err.Error()}
		}
		return resultadoInsertada, nil

	case err != nil:
		return resultadoError, &errorFila{resultado: resultadoError, motivo: "error consultando variante: " + err.Error()}
	}

	if !variante.PadreID.Valid || variante.PadreID.Int32 != padre.ID {
		return resultadoConflicto, &errorFila{
			resultado: resultadoConflicto,
			motivo:    "el SKU ya existe pero está vinculado a otro producto padre",
		}
	}

	_, err = qtx.ActualizarVariante(ctx, db.ActualizarVarianteParams{
		ID:                 variante.ID,
		PadreID:            pgtype.Int4{Int32: padre.ID, Valid: true},
		Sku:                datos.codigo,
		Medida:             variante.Medida,
		PrecioDistribuidor: utils.ToNumeric(datos.precioDistribuidor),
		PrecioLista:        utils.ToNumeric(datos.precioMostrador),
		PrecioPublico:      utils.ToNumeric(datos.precioMostrador),
		StockActual:        variante.StockActual,
		UnidadMedida:       catalogoUnidadMedida,
		LeadTimeDias:       variante.LeadTimeDias,
		Especificaciones:   variante.Especificaciones,
		Categoria:          pgtype.Text{String: catalogoCategoria, Valid: true},
		Subgrupo:           textoCatalogo(datos.volumen, 50),
		Modelo:             textoCatalogo(datos.tipo, 50),
		Tipo:               variante.Tipo,
		Marca:              variante.Marca,
		Multiplos:          pgtype.Int4{Int32: datos.multiplos, Valid: true},
		PermitirBackorder:  variante.PermitirBackorder,
	})
	if err != nil {
		return resultadoError, &errorFila{resultado: resultadoError, motivo: "error actualizando variante: " + err.Error()}
	}
	if err := tx.Commit(ctx); err != nil {
		return resultadoError, &errorFila{resultado: resultadoError, motivo: "error confirmando la transacción: " + err.Error()}
	}

	return resultadoActualizada, nil
}

// detectarEncabezadoCatalogo busca, entre las primeras filas de la hoja, la que
// contiene los encabezados requeridos. Algunos archivos incluyen filas vacías o
// de título antes del encabezado real.
func detectarEncabezadoCatalogo(filas [][]string) (int, columnasCatalogo, []string, error) {
	limite := len(filas)
	if limite > 50 {
		limite = 50
	}

	var primerError error
	for indice := 0; indice < limite; indice++ {
		if filaVacia(filas[indice]) {
			continue
		}
		columnas, advertencias, err := resolverColumnasCatalogo(filas[indice])
		if err == nil {
			return indice, columnas, advertencias, nil
		}
		if primerError == nil {
			primerError = err
		}
	}

	if primerError == nil {
		primerError = errors.New("no se encontró la fila de encabezados en el Excel")
	}
	return 0, columnasCatalogo{}, nil, primerError
}

func resolverColumnasCatalogo(encabezado []string) (columnasCatalogo, []string, error) {
	columnas := columnasCatalogo{
		codigo:         -1,
		productName:    -1,
		descProdTitulo: -1,
		usuario:        -1,
		mostrador:      -1,
		volumen:        -1,
		tipo:           -1,
		presentacion:   -1,
	}

	for indice, celda := range encabezado {
		switch normalizarEncabezado(celda) {
		case "codigo", "clave", "sku":
			if columnas.codigo < 0 {
				columnas.codigo = indice
			}
		case "productname", "nombreproducto", "nombre":
			if columnas.productName < 0 {
				columnas.productName = indice
			}
		case "descprodtitulo", "descprod", "descripcion", "descripcionproducto":
			if columnas.descProdTitulo < 0 {
				columnas.descProdTitulo = indice
			}
		case "usuario":
			if columnas.usuario < 0 {
				columnas.usuario = indice
			}
		case "mostrador":
			if columnas.mostrador < 0 {
				columnas.mostrador = indice
			}
		case "volumen":
			if columnas.volumen < 0 {
				columnas.volumen = indice
			}
		case "tipo":
			if columnas.tipo < 0 {
				columnas.tipo = indice
			}
		case "presentacion":
			if columnas.presentacion < 0 {
				columnas.presentacion = indice
			}
		}
	}

	faltantes := make([]string, 0, 7)
	if columnas.codigo < 0 {
		faltantes = append(faltantes, "CODIGO")
	}
	if columnas.productName < 0 {
		faltantes = append(faltantes, "Product Name")
	}
	if columnas.usuario < 0 {
		faltantes = append(faltantes, "usuario")
	}
	if columnas.mostrador < 0 {
		faltantes = append(faltantes, "mostrador")
	}
	if columnas.volumen < 0 {
		faltantes = append(faltantes, "volumen")
	}
	if columnas.tipo < 0 {
		faltantes = append(faltantes, "tipo")
	}
	if columnas.presentacion < 0 {
		faltantes = append(faltantes, "presentacion")
	}
	if len(faltantes) > 0 {
		return columnas, nil, fmt.Errorf("faltan columnas obligatorias en el Excel: %s", strings.Join(faltantes, ", "))
	}

	var advertencias []string
	if columnas.descProdTitulo < 0 {
		advertencias = append(advertencias,
			"la columna desc_prodtitulo no existe; se usará productos_integracion.desc_prod y, si está vacío, Product Name")
	}
	return columnas, advertencias, nil
}

func mapearFilaCatalogo(fila []string, columnas columnasCatalogo) (filaCatalogo, error) {
	datos := filaCatalogo{
		codigo:      strings.TrimSpace(valorColumna(fila, columnas.codigo)),
		titulo:      strings.TrimSpace(valorColumna(fila, columnas.productName)),
		descripcion: strings.TrimSpace(valorColumna(fila, columnas.descProdTitulo)),
		volumen:     strings.TrimSpace(valorColumna(fila, columnas.volumen)),
		tipo:        strings.TrimSpace(valorColumna(fila, columnas.tipo)),
	}
	if datos.codigo == "" {
		return datos, errors.New("CODIGO vacío")
	}
	if datos.titulo == "" {
		return datos, errors.New("Product Name vacío")
	}

	precioDistribuidor, err := parsePrecioCatalogo(valorColumna(fila, columnas.usuario))
	if err != nil {
		return datos, fmt.Errorf("usuario inválido: %w", err)
	}
	datos.precioDistribuidor = precioDistribuidor

	precioMostrador, err := parsePrecioCatalogo(valorColumna(fila, columnas.mostrador))
	if err != nil {
		return datos, fmt.Errorf("mostrador inválido: %w", err)
	}
	datos.precioMostrador = precioMostrador

	multiplos, err := parseEnteroCatalogo(valorColumna(fila, columnas.presentacion))
	if err != nil {
		return datos, fmt.Errorf("presentacion inválida: %w", err)
	}
	datos.multiplos = multiplos

	return datos, nil
}

func parsePrecioCatalogo(valor string) (float64, error) {
	limpio := strings.TrimSpace(valor)
	limpio = strings.ReplaceAll(limpio, "$", "")
	limpio = strings.ReplaceAll(limpio, ",", "")
	if limpio == "" {
		return 0, errors.New("valor vacío")
	}

	numero, err := strconv.ParseFloat(limpio, 64)
	if err != nil {
		return 0, fmt.Errorf("no es un número: %q", valor)
	}
	if math.IsNaN(numero) || math.IsInf(numero, 0) || numero < 0 {
		return 0, fmt.Errorf("número inválido: %q", valor)
	}
	return numero, nil
}

func parseEnteroCatalogo(valor string) (int32, error) {
	numero, err := parsePrecioCatalogo(valor)
	if err != nil {
		return 0, err
	}
	if numero != math.Trunc(numero) || numero > math.MaxInt32 {
		return 0, fmt.Errorf("se esperaba un entero: %q", valor)
	}
	if numero <= 0 {
		return 0, fmt.Errorf("debe ser mayor a cero: %q", valor)
	}
	return int32(numero), nil
}

func normalizarEncabezado(valor string) string {
	limpio := strings.TrimSpace(strings.ToLower(valor))
	reemplazos := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		".", "", "_", "", "-", "", " ", "", "\t", "",
	)
	return reemplazos.Replace(limpio)
}

func valorColumna(fila []string, indice int) string {
	if indice < 0 || indice >= len(fila) {
		return ""
	}
	return fila[indice]
}

func filaVacia(fila []string) bool {
	for _, celda := range fila {
		if strings.TrimSpace(celda) != "" {
			return false
		}
	}
	return true
}

func textoCatalogo(valor string, maximo int) pgtype.Text {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return pgtype.Text{Valid: false}
	}

	runas := []rune(valor)
	if len(runas) > maximo {
		valor = string(runas[:maximo])
	}
	return pgtype.Text{String: valor, Valid: true}
}

func escribirReporteIgnorados(rutaExcel string, ignoradas []FilaIgnorada) (string, error) {
	rutaReporte := rutaExcel + ".ignorados.log"

	archivo, err := os.Create(rutaReporte)
	if err != nil {
		return "", err
	}
	defer archivo.Close()

	if _, err := fmt.Fprintf(archivo, "Reporte de filas ignoradas: %s\n", rutaExcel); err != nil {
		return "", err
	}
	for _, ignorada := range ignoradas {
		if _, err := fmt.Fprintf(archivo, "fila=%d\tcodigo=%s\tmotivo=%s\n", ignorada.Fila, ignorada.Codigo, ignorada.Motivo); err != nil {
			return "", err
		}
	}
	return rutaReporte, nil
}
