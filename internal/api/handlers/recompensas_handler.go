package handlers

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/internal/recompensas"
	"github.com/OrlandoHdz/kubo/pkg/email"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecompensasHandler struct {
	queries  *db.Queries
	pool     *pgxpool.Pool
	service  *recompensas.RecompensasService
	emailCfg *email.Config
}

func NewRecompensasHandler(q *db.Queries) *RecompensasHandler {
	return &RecompensasHandler{
		queries: q,
		service: recompensas.NewRecompensasService(q),
	}
}

func NewRecompensasHandlerWithPool(q *db.Queries, pool *pgxpool.Pool) *RecompensasHandler {
	return &RecompensasHandler{
		queries: q,
		pool:    pool,
		service: recompensas.NewRecompensasService(q),
	}
}

func NewRecompensasHandlerWithPoolAndEmail(q *db.Queries, pool *pgxpool.Pool, emailCfg *email.Config) *RecompensasHandler {
	return &RecompensasHandler{
		queries:  q,
		pool:     pool,
		service:  recompensas.NewRecompensasService(q),
		emailCfg: emailCfg,
	}
}

// ===== PARÁMETROS =====

func (h *RecompensasHandler) GetParametros(c *gin.Context) {
	params, err := h.queries.GetParametrosRecompensa(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo parámetros: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, params)
}

func (h *RecompensasHandler) UpdateParametros(c *gin.Context) {
	var input struct {
		MontoCompraPorPunto float64 `json:"monto_compra_por_punto" binding:"required"`
		ValorPuntoPesos     float64 `json:"valor_punto_pesos" binding:"required"`
		UpdatedBy           int32   `json:"updated_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	params, err := h.queries.GetParametrosRecompensa(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo parámetros: " + err.Error()})
		return
	}

	actualizado, err := h.queries.UpdateParametrosRecompensa(c.Request.Context(), db.UpdateParametrosRecompensaParams{
		MontoCompraPorPunto: toNumericForce(input.MontoCompraPorPunto),
		ValorPuntoPesos:     toNumericForce(input.ValorPuntoPesos),
		UpdatedBy:           pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
		ID:                  params.ID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error actualizando parámetros: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, actualizado)
}

// ===== NIVELES DE RECOMPENSA =====

func (h *RecompensasHandler) ListarNiveles(c *gin.Context) {
	niveles, err := h.queries.ListNivelesRecompensa(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error listando niveles: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, niveles)
}

func (h *RecompensasHandler) CrearNivel(c *gin.Context) {
	var input struct {
		Nombre                   string  `json:"nombre" binding:"required"`
		NumeroNivel              *int32  `json:"numero_nivel"`
		PromedioCompraTrimestral float64 `json:"promedio_compra_trimestral"`
		PorcentajeDescuento      float64 `json:"porcentaje_descuento"`
		ImagenInsigniaURL        string  `json:"imagen_insignia_url"`
		CreatedBy                int32   `json:"created_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	if input.NumeroNivel == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "numero_nivel es requerido"})
		return
	}

	nivel, err := h.queries.CreateNivelRecompensa(c.Request.Context(), db.CreateNivelRecompensaParams{
		Nombre:                   input.Nombre,
		NumeroNivel:              *input.NumeroNivel,
		PromedioCompraTrimestral: toNumericForce(input.PromedioCompraTrimestral),
		PorcentajeDescuento:      toNumericForce(input.PorcentajeDescuento),
		ImagenInsigniaUrl:        pgtype.Text{String: input.ImagenInsigniaURL, Valid: input.ImagenInsigniaURL != ""},
		CreatedBy:                pgtype.Int4{Int32: input.CreatedBy, Valid: input.CreatedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando nivel: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, nivel)
}

func (h *RecompensasHandler) ActualizarNivel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var input struct {
		Nombre                   string  `json:"nombre" binding:"required"`
		NumeroNivel              *int32  `json:"numero_nivel"`
		PromedioCompraTrimestral float64 `json:"promedio_compra_trimestral"`
		PorcentajeDescuento      float64 `json:"porcentaje_descuento"`
		ImagenInsigniaURL        string  `json:"imagen_insignia_url"`
		UpdatedBy                int32   `json:"updated_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	if input.NumeroNivel == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "numero_nivel es requerido"})
		return
	}

	nivel, err := h.queries.UpdateNivelRecompensa(c.Request.Context(), db.UpdateNivelRecompensaParams{
		ID:                       int32(id),
		Nombre:                   input.Nombre,
		NumeroNivel:              *input.NumeroNivel,
		PromedioCompraTrimestral: toNumericForce(input.PromedioCompraTrimestral),
		PorcentajeDescuento:      toNumericForce(input.PorcentajeDescuento),
		ImagenInsigniaUrl:        pgtype.Text{String: input.ImagenInsigniaURL, Valid: input.ImagenInsigniaURL != ""},
		UpdatedBy:                pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error actualizando nivel: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, nivel)
}

func (h *RecompensasHandler) EliminarNivel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var input struct{ DeletedBy int32 }
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	err = h.queries.SoftDeleteNivelRecompensa(c.Request.Context(), db.SoftDeleteNivelRecompensaParams{
		ID: int32(id), DeletedBy: pgtype.Int4{Int32: input.DeletedBy, Valid: input.DeletedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error eliminando nivel: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Nivel eliminado"})
}

// ===== TRANSACCIONES DE PUNTOS =====

// ForzarEvaluacionNiveles ejecuta manualmente la evaluación de niveles (solo para pruebas)
func (h *RecompensasHandler) ForzarEvaluacionNiveles(c *gin.Context) {
	log.Println("[Recompensas] Evaluación de niveles forzada manualmente por API")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	h.service.EvaluarNiveles(ctx)

	c.JSON(http.StatusOK, gin.H{
		"message": "Evaluación de niveles ejecutada correctamente",
	})
}

func (h *RecompensasHandler) ListarTransaccionesCliente(c *gin.Context) {
	clienteIDStr := c.Param("cliente_id")
	clienteID, err := strconv.Atoi(clienteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	transacciones, err := h.queries.ListTransaccionesByCliente(c.Request.Context(), int32(clienteID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error listando transacciones: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, transacciones)
}

// ===== INFO RECOMPENSAS CLIENTE =====

func (h *RecompensasHandler) GetInfoRecompensasCliente(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	info, err := h.queries.GetClienteConNivel(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cliente no encontrado"})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *RecompensasHandler) AsignarNivelManual(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	var input struct {
		NivelRecompensaID int32 `json:"nivel_recompensa_id" binding:"required"`
		UpdatedBy         int32 `json:"updated_by"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	cliente, err := h.queries.UpdateClienteNivelManual(c.Request.Context(), db.UpdateClienteNivelManualParams{
		ID:                int32(id),
		NivelRecompensaID: pgtype.Int4{Int32: input.NivelRecompensaID, Valid: true},
		UpdatedBy:         pgtype.Int4{Int32: input.UpdatedBy, Valid: input.UpdatedBy != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error asignando nivel: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, cliente)
}

func (h *RecompensasHandler) Canjear(c *gin.Context) {
	var input struct {
		ClienteID int32 `json:"cliente_id" binding:"required"`
		UsuarioID int32 `json:"usuario_id" binding:"required"`
		Items     []struct {
			VarianteID int32 `json:"variante_id" binding:"required"`
			Cantidad   int32 `json:"cantidad" binding:"required"`
		} `json:"items" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	// Calcular total de puntos y recolectar variantes
	type varInfo struct {
		Variante db.ProductosRecompensaVariante
		Cantidad int32
	}
	var totalPuntos int32 = 0
	var infos []varInfo
	for _, it := range input.Items {
		if it.Cantidad <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cantidad inválida"})
			return
		}
		variante, err := h.queries.GetVarianteRecompensa(c.Request.Context(), it.VarianteID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Variante no encontrada: " + strconv.Itoa(int(it.VarianteID))})
			return
		}
		totalPuntos += variante.PrecioPuntos * it.Cantidad
		infos = append(infos, varInfo{Variante: variante, Cantidad: it.Cantidad})
		permitirBO := false
		if variante.PermitirBackorder.Valid {
			permitirBO = variante.PermitirBackorder.Bool
		}
		if !permitirBO && it.Cantidad > variante.StockActual {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stock insuficiente para variante " + variante.Sku})
			return
		}
	}
	if totalPuntos <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Total de puntos inválido"})
		return
	}
	clienteSaldo, err := h.queries.GetClienteSaldoPuntos(c.Request.Context(), input.ClienteID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cliente no encontrado"})
		return
	}
	if clienteSaldo.SaldoPuntos < totalPuntos {
		c.JSON(http.StatusForbidden, gin.H{"error": "Puntos insuficientes", "saldo": clienteSaldo.SaldoPuntos, "requerido": totalPuntos})
		return
	}
	saldoRestante := clienteSaldo.SaldoPuntos - totalPuntos
	// Si no hay pool, fallback a solo descontar puntos sin pedido
	if h.pool == nil {
		_, err = h.queries.CreateTransaccionPuntos(c.Request.Context(), db.CreateTransaccionPuntosParams{
			ClienteID: input.ClienteID,
			Tipo:      "CANJEADO",
			Puntos:    -totalPuntos,
			PedidoID:  pgtype.Int4{Valid: false},
			CreatedBy: pgtype.Int4{Int32: input.UsuarioID, Valid: input.UsuarioID != 0},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando transacción: " + err.Error()})
			return
		}
		err = h.queries.UpdateClienteSaldoPuntos(c.Request.Context(), db.UpdateClienteSaldoPuntosParams{
			ID:          input.ClienteID,
			SaldoPuntos: -totalPuntos,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error actualizando saldo: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Canje realizado con éxito", "puntos_canjeados": totalPuntos, "saldo_restante": saldoRestante})
		return
	}
	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iniciando transacción: " + err.Error()})
		return
	}
	defer tx.Rollback(c.Request.Context())
	qtx := h.queries.WithTx(tx)
	// Actualizar saldo y crear transacción
	err = qtx.UpdateClienteSaldoPuntos(c.Request.Context(), db.UpdateClienteSaldoPuntosParams{
		ID:          input.ClienteID,
		SaldoPuntos: -totalPuntos,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error actualizando saldo: " + err.Error()})
		return
	}
	folio := fmt.Sprintf("REC-%d", time.Now().Unix())
	pedido, err := qtx.CrearPedido(c.Request.Context(), db.CrearPedidoParams{
		Folio:               folio,
		ClienteID:           pgtype.Int4{Int32: input.ClienteID, Valid: true},
		UsuarioID:           pgtype.Int4{Int32: input.UsuarioID, Valid: true},
		Estado:              "CONFIRMADO",
		MetodoPago:          "Puntos",
		Subtotal:            utils.ToNumeric(0),
		DescuentoAplicado:   utils.ToNumeric(0),
		DescuentoPorcentaje: utils.ToNumeric(0),
		Iva:                 utils.ToNumeric(0),
		TotalOrden:          utils.ToNumeric(0),
		PuntosOtorgados:     -totalPuntos,
		NivelRecompensaID:   pgtype.Int4{Valid: false},
		CreatedBy:           pgtype.Int4{Int32: input.UsuarioID, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando pedido: " + err.Error()})
		return
	}
	_, err = qtx.CreateTransaccionPuntos(c.Request.Context(), db.CreateTransaccionPuntosParams{
		ClienteID: input.ClienteID,
		Tipo:      "CANJEADO",
		Puntos:    -totalPuntos,
		PedidoID:  pgtype.Int4{Int32: pedido.ID, Valid: true},
		CreatedBy: pgtype.Int4{Int32: input.UsuarioID, Valid: input.UsuarioID != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creando transacción: " + err.Error()})
		return
	}
	// Crear detalles del pedido
	for _, info := range infos {
		varianteID := info.Variante.ID
		// Intentar mapear a variante regular por SKU para que el detalle se vea en pedidos normales
		if regVar, err := qtx.GetVarianteBySKU(c.Request.Context(), info.Variante.Sku); err == nil {
			varianteID = regVar.ID
		}
		_, err = qtx.CrearPedidoDetalle(c.Request.Context(), db.CrearPedidoDetalleParams{
			PedidoID:               pgtype.Int4{Int32: pedido.ID, Valid: true},
			VarianteID:             pgtype.Int4{Int32: varianteID, Valid: true},
			Cantidad:               info.Cantidad,
			PrecioUnitarioAplicado: utils.ToNumeric(float64(info.Variante.PrecioPuntos)),
			CreatedBy:              pgtype.Int4{Int32: input.UsuarioID, Valid: true},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error creando detalle para variante %s: %s", info.Variante.Sku, err.Error())})
			return
		}
		// Descontar stock_actual de la variante de recompensa
		nuevoStock := info.Variante.StockActual - info.Cantidad
		if nuevoStock < 0 {
			nuevoStock = 0
		}
		_ = qtx.ActualizarStockVarianteRecompensa(c.Request.Context(), db.ActualizarStockVarianteRecompensaParams{
			ID:          info.Variante.ID,
			StockActual: nuevoStock,
			UpdatedBy:   pgtype.Int4{Int32: input.UsuarioID, Valid: true},
		})
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error confirmando canje: " + err.Error()})
		return
	}
	// Enviar correo de confirmación (igual que pedido normal pero con puntos)
	if h.emailCfg != nil {
		go func(pedido db.Pedido, infos []varInfo, totalPuntos int32, saldoRestante int32) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cliente, err := h.queries.GetCliente(ctx, input.ClienteID)
			if err != nil {
				log.Printf("Email recompensa: error obteniendo cliente %d: %v", input.ClienteID, err)
				return
			}
			usuarios, err := h.queries.ListarUsuariosPorCliente(ctx, pgtype.Int4{Int32: input.ClienteID, Valid: true})
			if err != nil || len(usuarios) == 0 {
				log.Printf("Email recompensa: cliente %d sin usuarios", input.ClienteID)
				return
			}
			var items []email.OrderItemData
			for _, info := range infos {
				ptsUnit := fmt.Sprintf("%d", info.Variante.PrecioPuntos)
				ptsImporte := fmt.Sprintf("%d", info.Variante.PrecioPuntos*info.Cantidad)
				desc := ""
				if info.Variante.Medida.Valid {
					desc = info.Variante.Medida.String
				}
				if desc == "" && pedido.ID != 0 {
					// intentar obtener nombre técnico del padre
					if padre, err := h.queries.GetProductoRecompensaPadre(ctx, info.Variante.PadreID); err == nil {
						desc = padre.Descripcion.String
						if padre.NombreTecnico != "" {
							desc = padre.NombreTecnico
						}
					}
				}
				items = append(items, email.OrderItemData{
					SKU:         info.Variante.Sku,
					Descripcion: desc,
					Cantidad:    info.Cantidad,
					Precio:      ptsUnit,
					Importe:     ptsImporte,
				})
			}
			orderData := email.OrderData{
				Folio:          pedido.Folio,
				Fecha:          pedido.CreatedAt.Time.Format("02/01/2006 15:04"),
				MetodoPago:     "Puntos",
				ClienteName:    cliente.NombreComercial,
				ClienteID:      input.ClienteID,
				Status:         pedido.Estado,
				IsRecompensa:   true,
				PuntosGastados: fmt.Sprintf("%d", totalPuntos),
				PuntosSaldo:    fmt.Sprintf("%d", saldoRestante),
				Items:          items,
			}
			h.emailCfg.SendOrderNotification(orderData, usuarios[0].Email)
		}(pedido, infos, totalPuntos, saldoRestante)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Canje realizado con éxito", "puntos_canjeados": totalPuntos, "saldo_restante": saldoRestante, "pedido": pedido})
}

// ===== Utilidades =====

func toNumericForce(v float64) pgtype.Numeric {
	n := pgtype.Numeric{}
	n.Int = intToBig(v)
	n.Exp = -2
	n.Valid = true
	return n
}

func intToBig(f float64) *big.Int {
	return big.NewInt(int64(f * 100))
}
