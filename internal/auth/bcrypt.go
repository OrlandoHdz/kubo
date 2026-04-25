package auth

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword convierte una contraseña en texto plano a un hash de bcrypt
func HashPassword(password string) (string, error) {
	// Usamos un costo de 12 para un buen balance entre seguridad y velocidad
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("error al cifrar la contraseña: %w", err)
	}
	return string(bytes), nil
}

// CheckPasswordHash compara una contraseña en texto plano con el hash guardado en la BD
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
