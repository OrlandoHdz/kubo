package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// En producción, esto debe venir de una variable de entorno
var secretKey = []byte("tu_clave_secreta_super_segura_2026")

type CustomClaims struct {
	UsuarioID int32  `json:"usuario_id"`
	Email     string `json:"email"`
	Rol       string `json:"rol"`
	jwt.RegisteredClaims
}

// GenerarToken crea un nuevo JWT firmado para un usuario
func GenerarToken(id int32, email string, rol string) (string, error) {
	claims := CustomClaims{
		id,
		email,
		rol,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 3)), // Expira en 3h
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ValidarToken comprueba si el token es válido y retorna sus datos
func ValidarToken(tokenString string) (*CustomClaims, error) {
	// 1. Obtener la clave secreta desde el entorno
	// jwtSecret := []byte(os.Getenv("tu_clave_secreta_super_segura_2026"))
	jwtSecret := []byte("tu_clave_secreta_super_segura_2026")

	// 2. Parsear el token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validamos que el método de firma sea el esperado (HMAC)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de firma inesperado")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	// 3. Extraer y validar los claims
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("token inválido")
}
