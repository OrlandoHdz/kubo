package utils

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ToNumeric convierte cualquier valor (string, int, float) a pgtype.Numeric.
// Ideal para Lineas de Crédito y Montos Mínimos.
func ToNumeric(val interface{}) pgtype.Numeric {
	var n pgtype.Numeric
	// Scan maneja la conversión interna de forma segura
	err := n.Scan(fmt.Sprintf("%v", val))
	if err != nil {
		return pgtype.Numeric{Valid: false}
	}
	return n
}

// ToTimestamp convierte un time.Time de Go a pgtype.Timestamp de Postgres.
// Útil para created_at, updated_at y deleted_at.
func ToTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{
		Time:  t,
		Valid: true,
	}
}

// ToDateNow es un helper rápido para obtener el timestamp del momento actual.
func ToDateNow() pgtype.Timestamp {
	return ToTimestamp(time.Now())
}

// NumericToFloat64 convierte un pgtype.Numeric a float64.
func NumericToFloat64(n pgtype.Numeric) (float64, error) {
	if !n.Valid {
		return 0, fmt.Errorf("valor numeric no es válido")
	}
	f8, err := n.Float64Value()
	if err != nil {
		return 0, err
	}
	return f8.Float64, nil
}
