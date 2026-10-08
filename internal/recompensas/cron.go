package recompensas

import (
	"context"
	"log"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
)

// Scheduler programa y ejecuta los trabajos periódicos del módulo de recompensas.
type Scheduler struct {
	service *RecompensasService
}

func NewScheduler(service *RecompensasService) *Scheduler {
	return &Scheduler{service: service}
}

// Start lanza los cron jobs en goroutines de fondo.
func (s *Scheduler) Start() {
	log.Println("[Recompensas] Iniciando scheduler de recompensas...")

	// Evaluación de niveles: cada día a las 3:00 AM
	go s.scheduleDailyLevelEvaluation()

	// Reinicio anual: 1 de enero a las 00:00:00
	go s.scheduleAnnualPointsReset()

	log.Println("[Recompensas] Scheduler iniciado.")
}

// scheduleDailyLevelEvaluation ejecuta la evaluación de niveles cada 24 horas.
func (s *Scheduler) scheduleDailyLevelEvaluation() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		delay := next.Sub(now)

		log.Printf("[Recompensas] Próxima evaluación de niveles en %v", delay)
		time.Sleep(delay)

		log.Println("[Recompensas] Ejecutando evaluación de niveles...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		s.service.EvaluarNiveles(ctx)
		cancel()
	}
}

// scheduleAnnualPointsReset ejecuta el reinicio de puntos el 1 de enero a las 00:00.
func (s *Scheduler) scheduleAnnualPointsReset() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		if now.After(next) {
			next = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
		}
		delay := next.Sub(now)

		log.Printf("[Recompensas] Próximo reinicio anual de puntos en %v", delay)
		time.Sleep(delay)

		log.Println("[Recompensas] Ejecutando reinicio anual de puntos...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		s.service.ReiniciarPuntosAnual(ctx)
		cancel()
	}
}

// HookPedidoCompletado se llama como goroutine separada después de crear un pedido.
// Si puntosOtorgados > 0, usa ese valor en lugar de calcularlo.
func HookPedidoCompletado(queries *db.Queries, clienteID int32, pedidoID int32, totalOrden float64, userID int32, puntosOtorgados int32) {
	service := NewRecompensasService(queries)
	service.AcumularPuntosPorPedido(context.Background(), clienteID, pedidoID, totalOrden, userID, puntosOtorgados)
}
