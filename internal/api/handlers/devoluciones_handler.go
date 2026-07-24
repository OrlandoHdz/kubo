package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/email"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type DevolucionesHandler struct {
	queries  *db.Queries
	ossCfg   *utils.ConfigOSS
	emailCfg *email.Config
}

func NewDevolucionesHandler(q *db.Queries, o *utils.ConfigOSS, e *email.Config) *DevolucionesHandler {
	return &DevolucionesHandler{queries: q, ossCfg: o, emailCfg: e}
}

func (h *DevolucionesHandler) Crear(c *gin.Context) {
	clienteIDStr := c.PostForm("cliente_id")
	pedidoFolio := c.PostForm("pedido_folio")
	tipo := c.PostForm("tipo")
	numerosParte := c.PostForm("numeros_parte")
	cantidades := c.PostForm("cantidades")
	notaCliente := c.PostForm("nota_cliente")

	if clienteIDStr == "" || pedidoFolio == "" || tipo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Faltan campos obligatorios (cliente_id, pedido_folio, tipo)"})
		return
	}
	clienteID, err := strconv.Atoi(clienteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cliente_id inválido"})
		return
	}

	if tipo != "Devolucion" && tipo != "Garantia" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tipo debe ser 'Devolucion' o 'Garantia'"})
		return
	}

	var evidenciasURLs []string
	files := c.Request.MultipartForm.File["evidencias"]
	for _, file := range files {
		url, err := h.ossCfg.SubirEvidencia(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al subir evidencia a OSS: " + err.Error()})
			return
		}
		evidenciasURLs = append(evidenciasURLs, url)
	}

	folio := fmt.Sprintf("DVG-%d", time.Now().Unix())

	usuarioID := int32(0)
	if uid, ok := c.Get("userID"); ok {
		if v, ok2 := uid.(int32); ok2 {
			usuarioID = v
		}
	}

	devolucion, err := h.queries.CrearDevolucion(c.Request.Context(), db.CrearDevolucionParams{
		Folio:        folio,
		ClienteID:    pgtype.Int4{Int32: int32(clienteID), Valid: true},
		PedidoFolio:  pedidoFolio,
		Tipo:         tipo,
		NumerosParte: numerosParte,
		Cantidades:   cantidades,
		NotaCliente:  notaCliente,
		Evidencias:   strings.Join(evidenciasURLs, "\n"),
		CreatedBy:    pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear solicitud: " + err.Error()})
		return
	}

	if h.emailCfg != nil {
		go h.notificarDevolucionCreada(context.Background(), devolucion.ID)
	}

	c.JSON(http.StatusCreated, devolucion)
}

func (h *DevolucionesHandler) notificarDevolucionCreada(ctx context.Context, devolucionID int32) {
	log.Printf("Email: notificarDevolucionCreada iniciado para devolucion %d", devolucionID)

	dv, err := h.queries.GetDevolucion(ctx, devolucionID)
	if err != nil {
		log.Printf("Email: error al obtener devolucion %d: %v", devolucionID, err)
		return
	}
	if !dv.ClienteID.Valid {
		return
	}

	cliente, err := h.queries.GetCliente(ctx, dv.ClienteID.Int32)
	if err != nil {
		log.Printf("Email: error al obtener cliente %d: %v", dv.ClienteID.Int32, err)
		return
	}

	usuarios, err := h.queries.ListarUsuariosPorCliente(ctx, dv.ClienteID)
	if err != nil || len(usuarios) == 0 {
		return
	}

	dvData := email.DevolucionData{
		Folio:       dv.Folio,
		Tipo:        dv.Tipo,
		PedidoFolio: dv.PedidoFolio,
		NumerosParte:  dv.NumerosParte,
		Cantidades:    dv.Cantidades,
		NotaCliente:   dv.NotaCliente,
		Estatus:       dv.Estatus,
		ClienteName:   cliente.NombreComercial,
	}

	h.emailCfg.SendDevolucionNotification(dvData, usuarios[0].Email)
}

func (h *DevolucionesHandler) ListarPorCliente(c *gin.Context) {
	clienteIDStr := c.Param("cliente_id")
	clienteID, err := strconv.Atoi(clienteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
		return
	}

	fechaInicio, fechaFin := parseDateRange(c)
	devoluciones, err := h.queries.ListDevolucionesByClienteRango(c.Request.Context(), db.ListDevolucionesByClienteRangoParams{
		ClienteID:   pgtype.Int4{Int32: int32(clienteID), Valid: true},
		CreatedAt:   fechaInicio,
		CreatedAt_2: fechaFin,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar solicitudes: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, devoluciones)
}

func (h *DevolucionesHandler) Listar(c *gin.Context) {
	fechaInicio, fechaFin := parseDateRange(c)
	devoluciones, err := h.queries.ListDevolucionesPorRango(c.Request.Context(), db.ListDevolucionesPorRangoParams{
		CreatedAt:   fechaInicio,
		CreatedAt_2: fechaFin,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar solicitudes: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, devoluciones)
}

func (h *DevolucionesHandler) ActualizarEstatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var req struct {
		Estatus           string `json:"estatus"`
		NotaAdministrador string `json:"nota_administrador"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}
	if req.Estatus != "Aprobada" && req.Estatus != "Rechazada" && req.Estatus != "Cancelada" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "estatus debe ser 'Aprobada', 'Rechazada' o 'Cancelada'"})
		return
	}
	if req.Estatus != "Cancelada" && req.NotaAdministrador == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "La nota del administrador es obligatoria"})
		return
	}

	usuarioID := int32(0)
	if uid, ok := c.Get("userID"); ok {
		if v, ok2 := uid.(int32); ok2 {
			usuarioID = v
		}
	}

	devolucion, err := h.queries.ActualizarEstatusDevolucion(c.Request.Context(), db.ActualizarEstatusDevolucionParams{
		ID:                int32(id),
		Estatus:           req.Estatus,
		NotaAdministrador: req.NotaAdministrador,
		UpdatedBy:         pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estatus: " + err.Error()})
		return
	}

	if h.emailCfg != nil {
		go h.notificarEstadoDevolucion(context.Background(), devolucion.ID, req.Estatus, req.NotaAdministrador)
	}

	c.JSON(http.StatusOK, devolucion)
}

func (h *DevolucionesHandler) Cancelar(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	usuarioID := int32(0)
	if uid, ok := c.Get("userID"); ok {
		if v, ok2 := uid.(int32); ok2 {
			usuarioID = v
		}
	}

	devolucion, err := h.queries.ActualizarEstatusDevolucion(c.Request.Context(), db.ActualizarEstatusDevolucionParams{
		ID:                int32(id),
		Estatus:           "Cancelada",
		NotaAdministrador: "",
		UpdatedBy:         pgtype.Int4{Int32: usuarioID, Valid: usuarioID != 0},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al cancelar solicitud: " + err.Error()})
		return
	}

	if h.emailCfg != nil {
		go h.notificarEstadoDevolucion(context.Background(), devolucion.ID, "Cancelada", "")
	}

	c.JSON(http.StatusOK, devolucion)
}

func (h *DevolucionesHandler) notificarEstadoDevolucion(ctx context.Context, devolucionID int32, estatus, notaAdmin string) {
	log.Printf("Email: notificarEstadoDevolucion iniciado para devolucion %d, estatus=%s", devolucionID, estatus)

	dv, err := h.queries.GetDevolucion(ctx, devolucionID)
	if err != nil {
		log.Printf("Email: error al obtener devolucion %d: %v", devolucionID, err)
		return
	}
	if !dv.ClienteID.Valid {
		return
	}

	cliente, err := h.queries.GetCliente(ctx, dv.ClienteID.Int32)
	if err != nil {
		log.Printf("Email: error al obtener cliente %d: %v", dv.ClienteID.Int32, err)
		return
	}

	usuarios, err := h.queries.ListarUsuariosPorCliente(ctx, dv.ClienteID)
	if err != nil || len(usuarios) == 0 {
		return
	}

	dvData := email.DevolucionData{
		Folio:             dv.Folio,
		Tipo:              dv.Tipo,
		PedidoFolio:       dv.PedidoFolio,
		NumerosParte:      dv.NumerosParte,
		Cantidades:        dv.Cantidades,
		NotaCliente:       dv.NotaCliente,
		NotaAdministrador: notaAdmin,
		Estatus:           estatus,
		ClienteName:       cliente.NombreComercial,
	}

	h.emailCfg.SendDevolucionStatusNotification(dvData, usuarios[0].Email)
}
