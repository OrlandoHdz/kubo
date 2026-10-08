package recompensas

import (
	"context"
	"log"
	"math"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// RecompensasService maneja la lógica de negocio del módulo de recompensas.
type RecompensasService struct {
	queries *db.Queries
}

func NewRecompensasService(q *db.Queries) *RecompensasService {
	return &RecompensasService{queries: q}
}

// ----- Acumulación de puntos por compra -----

// AcumularPuntosPorPedido calcula y asigna puntos después de un pedido completado.
// Se llama de forma no-bloqueante (goroutine) desde el handler de pedidos.
// Si puntosOtorgados > 0, usa ese valor en lugar de calcularlo (viene del frontend).
func (s *RecompensasService) AcumularPuntosPorPedido(ctx context.Context, clienteID int32, pedidoID int32, totalOrden float64, userID int32, puntosOtorgados int32) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Recompensas] panic en AcumularPuntosPorPedido (pedido=%d): %v", pedidoID, r)
		}
	}()

	var puntosGanados int32

	if puntosOtorgados > 0 {
		// Usar los puntos ya calculados por el frontend
		puntosGanados = puntosOtorgados
	} else {
		// Calcular puntos según parámetros (fallback para pedidos antiguos o directos)
		params, err := s.queries.GetParametrosRecompensa(ctx)
		if err != nil {
			log.Printf("[Recompensas] error obteniendo parámetros para pedido %d: %v", pedidoID, err)
			return
		}

		montoPorPunto, _ := numericToFloat64(params.MontoCompraPorPunto)
		if montoPorPunto <= 0 {
			return
		}

		puntosGanados = int32(math.Floor(totalOrden / montoPorPunto))
	}

	if puntosGanados <= 0 {
		return
	}

	// Registrar transacción
	_, err := s.queries.CreateTransaccionPuntos(ctx, db.CreateTransaccionPuntosParams{
		ClienteID: clienteID,
		Tipo:      "ACUMULADO",
		Puntos:    puntosGanados,
		PedidoID:  pgtype.Int4{Int32: pedidoID, Valid: true},
		CreatedBy: pgtype.Int4{Int32: userID, Valid: userID != 0},
	})
	if err != nil {
		log.Printf("[Recompensas] error creando transacción ACUMULADO para pedido %d: %v", pedidoID, err)
		return
	}

	// Actualizar saldo del cliente
	err = s.queries.UpdateClienteSaldoPuntos(ctx, db.UpdateClienteSaldoPuntosParams{
		ID:          clienteID,
		SaldoPuntos: puntosGanados,
	})
	if err != nil {
		log.Printf("[Recompensas] error actualizando saldo puntos cliente %d: %v", clienteID, err)
		return
	}

	log.Printf("[Recompensas] cliente %d ganó %d puntos por pedido %d ($%.2f)", clienteID, puntosGanados, pedidoID, totalOrden)
}

// ----- Evaluación de niveles -----

// EvaluarNiveles recorre todos los clientes con nivel no-fijado y recalcula
// su nivel según el promedio de compras trimestral y periodo de gracia.
func (s *RecompensasService) EvaluarNiveles(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Recompensas] panic en EvaluarNiveles: %v", r)
		}
	}()

	clientes, err := s.queries.ListarClientesConNivel(ctx)
	if err != nil {
		log.Printf("[Recompensas] error listando clientes con nivel: %v", err)
		return
	}

	niveles, err := s.queries.ListNivelesRecompensa(ctx)
	if err != nil {
		log.Printf("[Recompensas] error listando niveles: %v", err)
		return
	}

	for _, c := range clientes {
		clienteID := c.ID

		// Calcular total de compras en últimos 90 días
		totalResult, err := s.queries.GetTotalCompras90Dias(ctx, pgtype.Int4{Int32: clienteID, Valid: true})
		if err != nil {
			log.Printf("[Recompensas] error calculando compras 90 días cliente %d: %v", clienteID, err)
			continue
		}
		totalCompras, _ := numericToFloat64(totalResult)
		promedioTrimestral := totalCompras / 3.0

		// Determinar el nivel correspondiente
		nivelCorrespondiente := int32(0) // Nivel 0 por defecto
		for _, nivel := range niveles {
			req, _ := numericToFloat64(nivel.PromedioCompraTrimestral)
			if promedioTrimestral >= req {
				nivelCorrespondiente = nivel.NumeroNivel
			}
		}

		// Si el nivel actual del cliente es diferente al calculado, actualizar
		nivelActual := c.NivelNumero.Int32
		if nivelActual != nivelCorrespondiente {
			nivelTarget, err := s.queries.GetNivelRecompensaByNumero(ctx, nivelCorrespondiente)
			if err != nil {
				log.Printf("[Recompensas] nivel %d no encontrado para cliente %d: %v", nivelCorrespondiente, clienteID, err)
				continue
			}

			_, err = s.queries.UpdateClienteNivelAutomatico(ctx, db.UpdateClienteNivelAutomaticoParams{
				ID:                clienteID,
				NivelRecompensaID: pgtype.Int4{Int32: nivelTarget.ID, Valid: true},
			})
			if err != nil {
				log.Printf("[Recompensas] error actualizando nivel cliente %d: %v", clienteID, err)
				continue
			}
			log.Printf("[Recompensas] cliente %d actualizado a nivel %d (%s)", clienteID, nivelCorrespondiente, nivelTarget.Nombre)
		}
	}

	log.Printf("[Recompensas] evaluación de niveles completada para %d clientes", len(clientes))
}

// ----- Reinicio anual de puntos -----

// ReiniciarPuntosAnual pone en cero el saldo de puntos de todos los clientes
// con saldo > 0 y registra las transacciones de REINICIO_ANUAL.
func (s *RecompensasService) ReiniciarPuntosAnual(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Recompensas] panic en ReiniciarPuntosAnual: %v", r)
		}
	}()

	clientes, err := s.queries.GetClientesConSaldoMayorCero(ctx)
	if err != nil {
		log.Printf("[Recompensas] error listando clientes con saldo > 0: %v", err)
		return
	}

	contador := 0
	for _, c := range clientes {
		saldo := c.SaldoPuntos

		// Registrar transacción de reinicio
		_, err = s.queries.CreateTransaccionPuntos(ctx, db.CreateTransaccionPuntosParams{
			ClienteID: c.ID,
			Tipo:      "REINICIO_ANUAL",
			Puntos:    -saldo,
			CreatedBy: pgtype.Int4{Valid: false},
		})
		if err != nil {
			log.Printf("[Recompensas] error registrando reinicio puntos cliente %d: %v", c.ID, err)
			continue
		}

		// Resetear saldo
		err = s.queries.UpdateClienteSaldoPuntos(ctx, db.UpdateClienteSaldoPuntosParams{
			ID:          c.ID,
			SaldoPuntos: -saldo,
		})
		if err != nil {
			log.Printf("[Recompensas] error reseteando saldo cliente %d: %v", c.ID, err)
			continue
		}

		contador++
		log.Printf("[Recompensas] reiniciado saldo de %d puntos para cliente %d", saldo, c.ID)
	}

	log.Printf("[Recompensas] reinicio anual completado: %d clientes reiniciados", contador)
}

// ----- Utilidades -----

func numericToFloat64(n pgtype.Numeric) (float64, error) {
	if !n.Valid || n.Int == nil {
		return 0, nil
	}
	f, err := n.Float64Value()
	if err != nil {
		return 0, err
	}
	if !f.Valid {
		return 0, nil
	}
	return f.Float64, nil
}
