package services

import (
	"strings"
	"testing"
)

func TestResolverColumnasCatalogo(t *testing.T) {
	encabezado := []string{
		"NO.", "CODIGO", "Product Name", "volumen", "tipo", "STOCK",
		"CANTIDAD VOL", "PRESENTACION", "rev vol.", "reventa", "usuario", "mostrador",
	}

	columnas, advertencias, err := resolverColumnasCatalogo(encabezado)
	if err != nil {
		t.Fatalf("resolverColumnasCatalogo() error = %v", err)
	}

	if columnas.codigo != 1 || columnas.productName != 2 || columnas.volumen != 3 ||
		columnas.tipo != 4 || columnas.presentacion != 7 || columnas.usuario != 10 ||
		columnas.mostrador != 11 {
		t.Fatalf("índices de columnas inesperados: %+v", columnas)
	}
	if len(advertencias) != 1 || !strings.Contains(advertencias[0], "desc_prodtitulo") {
		t.Fatalf("se esperaba advertencia por desc_prodtitulo, got %v", advertencias)
	}
}

func TestResolverColumnasCatalogoFaltantes(t *testing.T) {
	_, _, err := resolverColumnasCatalogo([]string{"CODIGO", "Product Name"})
	if err == nil {
		t.Fatal("se esperaba error por columnas faltantes")
	}
	for _, columna := range []string{"usuario", "mostrador", "volumen", "tipo", "presentacion"} {
		if !strings.Contains(err.Error(), columna) {
			t.Errorf("el error no menciona la columna %q: %v", columna, err)
		}
	}
}

func TestMapearFilaCatalogo(t *testing.T) {
	columnas := columnasCatalogo{
		codigo:         1,
		productName:    2,
		descProdTitulo: 3,
		volumen:        3,
		tipo:           4,
		presentacion:   7,
		usuario:        10,
		mostrador:      11,
	}
	fila := []string{
		"28", "AR32", "AISLADOR 32 TWECO WELDSKILL", "AISLADOR", "TWECO", "500",
		"20", "10", "13.01", "17.74", "27.30", "40.95",
	}

	datos, err := mapearFilaCatalogo(fila, columnas)
	if err != nil {
		t.Fatalf("mapearFilaCatalogo() error = %v", err)
	}

	if datos.codigo != "AR32" || datos.titulo != "AISLADOR 32 TWECO WELDSKILL" {
		t.Fatalf("datos base inesperados: %+v", datos)
	}
	if datos.descripcion != "AISLADOR" || datos.volumen != "AISLADOR" || datos.tipo != "TWECO" {
		t.Fatalf("mapeo de textos inesperado: %+v", datos)
	}
	if datos.precioDistribuidor != 27.30 || datos.precioMostrador != 40.95 {
		t.Fatalf("precios inesperados: %+v", datos)
	}
	if datos.multiplos != 10 {
		t.Fatalf("múltiplos inesperados: %d", datos.multiplos)
	}
}

func TestMapearFilaCatalogoInvalida(t *testing.T) {
	columnas := columnasCatalogo{
		codigo: 0, productName: 1, descProdTitulo: -1,
		volumen: 2, tipo: 3, presentacion: 4, usuario: 5, mostrador: 6,
	}

	casos := []struct {
		nombre string
		fila   []string
	}{
		{"sin código", []string{"", "Producto", "VOL", "TIPO", "1", "10", "20"}},
		{"sin nombre", []string{"AR1", "", "VOL", "TIPO", "1", "10", "20"}},
		{"precio usuario inválido", []string{"AR1", "Producto", "VOL", "TIPO", "1", "abc", "20"}},
		{"precio mostrador inválido", []string{"AR1", "Producto", "VOL", "TIPO", "1", "10", "-1"}},
		{"presentación inválida", []string{"AR1", "Producto", "VOL", "TIPO", "1.5", "10", "20"}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := mapearFilaCatalogo(caso.fila, columnas); err == nil {
				t.Fatal("se esperaba error de parseo")
			}
		})
	}
}

func TestParsePrecioCatalogo(t *testing.T) {
	casos := map[string]float64{
		"27.305301799368266": 27.305301799368266,
		"$1,234.56":          1234.56,
		" 10 ":               10,
		"0":                  0,
	}
	for entrada, esperado := range casos {
		obtenido, err := parsePrecioCatalogo(entrada)
		if err != nil {
			t.Fatalf("parsePrecioCatalogo(%q) error = %v", entrada, err)
		}
		if obtenido != esperado {
			t.Errorf("parsePrecioCatalogo(%q) = %v, esperado %v", entrada, obtenido, esperado)
		}
	}

	for _, invalido := range []string{"", "abc", "-5", "NaN", "+Inf"} {
		if _, err := parsePrecioCatalogo(invalido); err == nil {
			t.Errorf("parsePrecioCatalogo(%q) debió fallar", invalido)
		}
	}
}

func TestParseEnteroCatalogo(t *testing.T) {
	obtenido, err := parseEnteroCatalogo("10")
	if err != nil {
		t.Fatalf("parseEnteroCatalogo() error = %v", err)
	}
	if obtenido != 10 {
		t.Fatalf("parseEnteroCatalogo() = %d, esperado 10", obtenido)
	}

	for _, invalido := range []string{"", "0", "-1", "1.5", "abc"} {
		if _, err := parseEnteroCatalogo(invalido); err == nil {
			t.Errorf("parseEnteroCatalogo(%q) debió fallar", invalido)
		}
	}
}
