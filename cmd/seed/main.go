package main

import (
	"context"
	"log"

	"github.com/OrlandoHdz/kubo/internal/auth"
	"github.com/OrlandoHdz/kubo/internal/database"
	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/jackc/pgx/v5/pgtype"
)

func main() {
	ctx := context.Background()
	pool, err := database.NuevoPool(ctx, "configs/db/database.yaml")

	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	queries := db.New(pool)

	// 1. Hashear contraseña
	hash, _ := auth.HashPassword("Tigres.2026")

	// 2. Crear Admin (Staff interno) si no existe
	var admin db.Usuario
	admin, err = queries.GetUsuarioByEmail(ctx, "orlando.hdz@gmail.com")
	if err != nil {
		admin, err = queries.CrearUsuario(ctx, db.CrearUsuarioParams{
			Email:        "orlando.hdz@gmail.com",
			PasswordHash: hash,
			Rol:          "Admin",
			IsActive:     pgtype.Bool{Bool: true, Valid: true},
			ClienteID:    pgtype.Int4{Valid: false}, // NULL: Staff interno [cite: 231]
			CreatedBy:    pgtype.Int4{Valid: false},
		})
		if err != nil {
			log.Fatal("Error creando admin: ", err)
		}
		log.Println("Admin creado: ", admin.ID)
	} else {
		log.Println("Admin ya existe: ", admin.ID)
	}

	// 3. Crear Cliente Industrial (Prevención de Riesgo [cite: 27, 30]) si no existe
	cliente, err := queries.GetClienteByRFC(ctx, "DEL010101ABC")
	if err != nil {
		cliente, err = queries.CrearCliente(ctx, db.CrearClienteParams{
			NombreComercial: "Constructora Delta",
			RazonSocial:     "Delta S.A. de C.V.",
			Rfc:             "DEL010101ABC",
			Estado:          "Activo",

			// Usamos la utilidad centralizada
			MontoMinimoCompra:     utils.ToNumeric(500.00),
			LineaCreditoTotal:     utils.ToNumeric(100000),
			LineaCreditoUtilizada: utils.ToNumeric(0),
			DiasCredito:           30,
			PermitirPagoCredito:   true,
			MetodoPagoPreferente:  "Transferencia",

			CreatedBy: pgtype.Int4{Int32: admin.ID, Valid: true},
		})

		if err != nil {
			log.Fatalf("Error creando cliente: %v", err)
		}
		log.Println("Cliente creado: ", cliente.NombreComercial)
	} else {
		log.Println("Cliente ya existe: ", cliente.NombreComercial)
	}

	// 4. Crear Productos Padre y Variantes
	productosSemilla := []struct {
		Nombre      string
		Descripcion string
		Categoria   string
		Marca       string
		Variantes   []struct {
			SKU    string
			Medida string
			Precio float64
			Stock  int32
		}
	}{
		{
			Nombre:      "Tornillo Hexagonal G5",
			Descripcion: "Tornillo hexagonal grado 5, acabado galvanizado",
			Categoria:   "Fijación",
			Marca:       "Kubo Fasteners",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "THG5-14-1", Medida: "1/4\" x 1\"", Precio: 2.50, Stock: 1000},
				{SKU: "THG5-14-2", Medida: "1/4\" x 2\"", Precio: 3.80, Stock: 800},
				{SKU: "THG5-12-1", Medida: "1/2\" x 1\"", Precio: 5.20, Stock: 500},
			},
		},
		{
			Nombre:      "Guantes de Nitrilo",
			Descripcion: "Guantes de nitrilo sin polvo, caja con 100 piezas",
			Categoria:   "Seguridad e Higiene",
			Marca:       "Kubo Safety",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "GN-CH", Medida: "Chico", Precio: 185.00, Stock: 200},
				{SKU: "GN-MD", Medida: "Mediano", Precio: 185.00, Stock: 350},
				{SKU: "GN-GD", Medida: "Grande", Precio: 185.00, Stock: 400},
			},
		},
		{
			Nombre:      "Arandela Plana Galvanizada",
			Descripcion: "Arandela plana para distribución de carga, acabado galvanizado",
			Categoria:   "Fijación",
			Marca:       "Kubo Fasteners",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "APG-14", Medida: "1/4\"", Precio: 0.45, Stock: 5000},
				{SKU: "APG-38", Medida: "3/8\"", Precio: 0.85, Stock: 3000},
			},
		},
		{
			Nombre:      "Electrodo 6013 1/8",
			Descripcion: "Electrodo para soldadura de acero al carbón, excelente acabado",
			Categoria:   "Soldadura",
			Marca:       "Kubo Welds",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "E6013-18", Medida: "1/8\" x 14\"", Precio: 1250.00, Stock: 50},
			},
		},
		{
			Nombre:      "Cinta de Aislar Super 33",
			Descripcion: "Cinta de PVC de grado profesional para aislamiento eléctrico",
			Categoria:   "Cintas",
			Marca:       "3M / Kubo",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "CA-S33", Medida: "19mm x 20m", Precio: 85.00, Stock: 150},
			},
		},
		{
			Nombre:      "Casco de Seguridad Ala Ancha",
			Descripcion: "Casco de seguridad con suspensión de 4 puntos, color blanco",
			Categoria:   "Seguridad e Higiene",
			Marca:       "Kubo Safety",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "CS-AA-B", Medida: "Ajustable", Precio: 245.00, Stock: 80},
			},
		},
		{
			Nombre:      "Pija Multiusos Negra",
			Descripcion: "Pija punta de broca para madera o tabla roca",
			Categoria:   "Fijación",
			Marca:       "Kubo Fasteners",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "PMN-6-1", Medida: "#6 x 1\"", Precio: 0.35, Stock: 10000},
				{SKU: "PMN-8-2", Medida: "#8 x 2\"", Precio: 0.65, Stock: 5000},
			},
		},
		{
			Nombre:      "Lentes de Seguridad Claros",
			Descripcion: "Lentes de policarbonato con recubrimiento anti-empañante",
			Categoria:   "Seguridad e Higiene",
			Marca:       "Kubo Safety",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "LS-CL", Medida: "Unitalla", Precio: 45.00, Stock: 300},
			},
		},
		{
			Nombre:      "Tuerca Hexagonal Galvanizada",
			Descripcion: "Tuerca hexagonal estándar, acabado galvanizado",
			Categoria:   "Fijación",
			Marca:       "Kubo Fasteners",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "THG-14", Medida: "1/4\"", Precio: 0.55, Stock: 4000},
				{SKU: "THG-12", Medida: "1/2\"", Precio: 1.20, Stock: 2000},
			},
		},
		{
			Nombre:      "Cinta Canela de Empaque",
			Descripcion: "Cinta adhesiva de polipropileno para sellado de cajas",
			Categoria:   "Cintas",
			Marca:       "Kubo Tapes",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "CC-48", Medida: "48mm x 150m", Precio: 55.00, Stock: 600},
			},
		},
		{
			Nombre:      "Chaleco Reflejante",
			Descripcion: "Chaleco de alta visibilidad con bandas reflejantes",
			Categoria:   "Seguridad e Higiene",
			Marca:       "Kubo Safety",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "CR-NAR", Medida: "Unitalla / Naranja", Precio: 95.00, Stock: 120},
				{SKU: "CR-VER", Medida: "Unitalla / Verde", Precio: 95.00, Stock: 120},
			},
		},
		{
			Nombre:      "Electrodo 7018 5/32",
			Descripcion: "Electrodo de bajo hidrógeno para soldaduras estructurales",
			Categoria:   "Soldadura",
			Marca:       "Lincoln / Kubo",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "E7018-532", Medida: "5/32\"", Precio: 1580.00, Stock: 30},
			},
		},
		{
			Nombre:      "Flexómetro Profesional",
			Descripcion: "Cinta métrica de acero con carcasa resistente a impactos",
			Categoria:   "Herramientas",
			Marca:       "Stanley / Kubo",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "FX-5M", Medida: "5 Metros", Precio: 165.00, Stock: 45},
				{SKU: "FX-8M", Medida: "8 Metros", Precio: 245.00, Stock: 30},
			},
		},
		{
			Nombre:      "Careta para Soldar Electrónica",
			Descripcion: "Careta fotosensible con ajuste de sombra automático",
			Categoria:   "Soldadura",
			Marca:       "Kubo Welds",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "CSE-AUTO", Medida: "Sombra 9-13", Precio: 1890.00, Stock: 15},
			},
		},
		{
			Nombre:      "Cinta Doble Capa",
			Descripcion: "Cinta adhesiva de espuma doble cara para montajes",
			Categoria:   "Cintas",
			Marca:       "Kubo Tapes",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "CDC-1", Medida: "1 pulgada x 5m", Precio: 120.00, Stock: 100},
			},
		},
		{
			Nombre:      "Pinzas de Corte Diagonal",
			Descripcion: "Pinzas de corte para alambre y cables eléctricos",
			Categoria:   "Herramientas",
			Marca:       "Kubo Tools",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "PC-7", Medida: "7 Pulgadas", Precio: 220.00, Stock: 40},
			},
		},
		{
			Nombre:      "Taquete de Plástico con Tornillo",
			Descripcion: "Kit de taquete y tornillo para fijación en muro",
			Categoria:   "Fijación",
			Marca:       "Kubo Fasteners",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "TK-14", Medida: "1/4\"", Precio: 1.20, Stock: 2000},
				{SKU: "TK-38", Medida: "3/8\"", Precio: 2.10, Stock: 1500},
			},
		},
		{
			Nombre:      "Cepillo de Alambre",
			Descripcion: "Cepillo con cerdas de acero para limpieza de soldadura",
			Categoria:   "Soldadura",
			Marca:       "Kubo Welds",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "CE-AL", Medida: "Madera / Acero", Precio: 38.00, Stock: 200},
			},
		},
		{
			Nombre:      "Tapones Auditivos Desechables",
			Descripcion: "Tapones de espuma suave expandible para protección auditiva",
			Categoria:   "Seguridad e Higiene",
			Marca:       "Kubo Safety",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "TA-DES", Medida: "Unitalla", Precio: 12.00, Stock: 1000},
			},
		},
		{
			Nombre:      "Alambre para Microalambre ER70S-6",
			Descripcion: "Rollo de alambre sólido para proceso MIG",
			Categoria:   "Soldadura",
			Marca:       "Kubo Welds",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "MA-035", Medida: "0.035\" / 15kg", Precio: 1150.00, Stock: 25},
			},
		},
		{
			Nombre:      "Cinta para Ductos (Duct Tape)",
			Descripcion: "Cinta de tela reforzada multiusos de alta resistencia",
			Categoria:   "Cintas",
			Marca:       "Kubo Tapes",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "DT-GRI", Medida: "2 pulg x 30m", Precio: 145.00, Stock: 85},
			},
		},
		{
			Nombre:      "Martillo de Bola",
			Descripcion: "Martillo con mango de fibra de vidrio y cabeza de acero",
			Categoria:   "Herramientas",
			Marca:       "Kubo Tools",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "MB-16", Medida: "16 Onzas", Precio: 310.00, Stock: 20},
			},
		},
		{
			Nombre:      "Bota de Seguridad con Casco",
			Descripcion: "Calzado de seguridad industrial con casquillo de acero",
			Categoria:   "Seguridad e Higiene",
			Marca:       "Kubo Safety",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "BS-26", Medida: "Talla 26", Precio: 890.00, Stock: 15},
				{SKU: "BS-27", Medida: "Talla 27", Precio: 890.00, Stock: 20},
				{SKU: "BS-28", Medida: "Talla 28", Precio: 890.00, Stock: 15},
			},
		},
		{
			Nombre:      "Disco de Corte 4.5\"",
			Descripcion: "Disco abrasivo para corte de metal y acero inoxidable",
			Categoria:   "Soldadura",
			Marca:       "Kubo Abrasives",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "DC-45", Medida: "4.5\" x 0.045\"", Precio: 28.00, Stock: 500},
			},
		},
		{
			Nombre:      "Perno Estructural A325",
			Descripcion: "Perno de alta resistencia para estructuras metálicas",
			Categoria:   "Fijación",
			Marca:       "Kubo Fasteners",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "PE-58-2", Medida: "5/8\" x 2\"", Precio: 18.50, Stock: 400},
				{SKU: "PE-34-3", Medida: "3/4\" x 3\"", Precio: 32.00, Stock: 250},
			},
		},
		{
			Nombre:      "Mascarilla N95",
			Descripcion: "Respirador para partículas libre de mantenimiento",
			Categoria:   "Seguridad e Higiene",
			Marca:       "Kubo Safety",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "M-N95", Medida: "Desechable", Precio: 35.00, Stock: 500},
			},
		},
		{
			Nombre:      "Cinta de Enmascarar (Masking)",
			Descripcion: "Cinta de papel para pintura y uso general",
			Categoria:   "Cintas",
			Marca:       "Kubo Tapes",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "MT-1", Medida: "1 Pulgada", Precio: 42.00, Stock: 200},
			},
		},
		{
			Nombre:      "Juego de Llaves Combinadas",
			Descripcion: "Set de llaves de acero cromo vanadio (12 piezas)",
			Categoria:   "Herramientas",
			Marca:       "Kubo Tools",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "JL-12P", Medida: "8-19mm", Precio: 980.00, Stock: 10},
			},
		},
		{
			Nombre:      "Boquilla para Corte Oxicorte",
			Descripcion: "Boquilla de repuesto para soplete de corte",
			Categoria:   "Soldadura",
			Marca:       "Victor / Kubo",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "BC-1", Medida: "#1 Acetileno", Precio: 245.00, Stock: 60},
			},
		},
		{
			Nombre:      "Antorcha MIG 250A",
			Descripcion: "Antorcha completa para proceso de soldadura MIG",
			Categoria:   "Soldadura",
			Marca:       "Kubo Welds",
			Variantes: []struct {
				SKU    string
				Medida string
				Precio float64
				Stock  int32
			}{
				{SKU: "AMIG-250", Medida: "15 Pies", Precio: 2850.00, Stock: 8},
			},
		},
	}

	for _, p := range productosSemilla {
		padre, err := queries.GetProductoPadreByNombre(ctx, p.Nombre)
		if err != nil {
			padre, err = queries.CrearProductoPadre(ctx, db.CrearProductoPadreParams{
				NombreTecnico: p.Nombre,
				Descripcion:   pgtype.Text{String: p.Descripcion, Valid: p.Descripcion != ""},
				Categoria:     p.Categoria,
				Marca:         pgtype.Text{String: p.Marca, Valid: p.Marca != ""},
				CreatedBy:     pgtype.Int4{Int32: admin.ID, Valid: true},
			})
			if err != nil {
				log.Printf("Error creando producto %s: %v", p.Nombre, err)
				continue
			}
			log.Println("Producto padre creado: ", p.Nombre)
		} else {
			log.Println("Producto padre ya existe: ", p.Nombre)
		}

		for _, v := range p.Variantes {
			_, err := queries.GetVarianteBySKU(ctx, v.SKU)
			if err != nil {
				_, err = queries.CrearVariante(ctx, db.CrearVarianteParams{
					PadreID:          pgtype.Int4{Int32: padre.ID, Valid: true},
					Sku:              v.SKU,
					Medida:           pgtype.Text{String: v.Medida, Valid: v.Medida != ""},
					PrecioLista:      utils.ToNumeric(v.Precio),
					StockActual:      v.Stock,
					UnidadMedida:     "Pieza",
					LeadTimeDias:     pgtype.Int4{Int32: 2, Valid: true},
					Especificaciones: pgtype.Text{Valid: false},
					CreatedBy:        pgtype.Int4{Int32: admin.ID, Valid: true},
				})
				if err != nil {
					log.Printf("  Error creando variante %s: %v", v.SKU, err)
					continue
				}
				log.Println("  Variante creada: ", v.SKU)
			} else {
				log.Println("  Variante ya existe: ", v.SKU)
			}
		}
	}

	log.Println("✅ Seeding de productos completado con éxito")
}
