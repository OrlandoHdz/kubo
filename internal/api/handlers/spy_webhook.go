package handlers

import (
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SpyWebhook atrapa cualquier petición POST y registra sus encabezados y cuerpo en la consola.
func SpyWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Error leyendo el body del webhook: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se pudo leer el cuerpo de la petición"})
		return
	}

	log.Println("=== INICIO DE SPY WEBHOOK ===")
	log.Println("Method:", c.Request.Method)
	log.Println("URL:", c.Request.URL.String())

	log.Println("Headers:")
	for k, v := range c.Request.Header {
		log.Printf("  %s: %v\n", k, v)
	}

	log.Printf("Body:\n%s\n", string(body))
	log.Println("=== FIN DE SPY WEBHOOK ===")

	c.JSON(http.StatusOK, gin.H{
		"status":  "recibido",
		"message": "Webhook atrapado correctamente",
	})
}
