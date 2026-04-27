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

	// 2. Crear Admin (Staff interno)
	admin, err := queries.CrearUsuario(ctx, db.CrearUsuarioParams{
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

	// 3. Crear Cliente Industrial (Prevención de Riesgo [cite: 27, 30])
	cliente, err := queries.CrearCliente(ctx, db.CrearClienteParams{
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
		log.Fatalf("Error en seed: %v", err)
	}

	log.Printf("Seed completado. Cliente %s creado por admin %d", cliente.NombreComercial, admin.ID)
}
