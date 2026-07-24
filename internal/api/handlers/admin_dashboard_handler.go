package handlers

import (
	"log"
	"net/http"

	"github.com/OrlandoHdz/kubo/internal/db"
	"github.com/OrlandoHdz/kubo/pkg/utils"
	"github.com/gin-gonic/gin"
)

type AdminDashboardHandler struct {
	queries *db.Queries
}

func NewAdminDashboardHandler(q *db.Queries) *AdminDashboardHandler {
	return &AdminDashboardHandler{queries: q}
}

type AdminDashboardResponse struct {
	Inventario struct {
		TotalVariantes int64 `json:"total_variantes"`
		SinStock       int64 `json:"sin_stock"`
		StockBajo      int64 `json:"stock_bajo"`
		ConStock       int64 `json:"con_stock"`
	} `json:"inventario"`

	Pedidos struct {
		TotalPedidos     int64   `json:"total_pedidos"`
		MontoTotalOrdenes float64 `json:"monto_total_ordenes"`
	} `json:"pedidos"`

	Backorders struct {
		TotalBackorders int64   `json:"total_backorders"`
		MontoRetenido   float64 `json:"monto_retenido"`
	} `json:"backorders"`

	Devoluciones struct {
		Pendientes int64 `json:"pendientes"`
		Aprobadas  int64 `json:"aprobadas"`
		Rechazadas int64 `json:"rechazadas"`
		Canceladas int64 `json:"canceladas"`
		Total      int64 `json:"total"`
	} `json:"devoluciones"`

	Facturas struct {
		CarteraVencida      float64 `json:"cartera_vencida"`
		SaldoPendienteTotal float64 `json:"saldo_pendiente_total"`
	} `json:"facturas"`

	Ingresos struct {
		TotalIngresos float64 `json:"total_ingresos"`
		TotalPagos    int64   `json:"total_pagos"`
	} `json:"ingresos"`

	IngresosMensuales []IngresoMensual `json:"ingresos_mensuales"`
}

type IngresoMensual struct {
	Anio  int32   `json:"anio"`
	Mes   int32   `json:"mes"`
	Total float64 `json:"total"`
}

func (h *AdminDashboardHandler) ObtenerDashboard(c *gin.Context) {
	resp := AdminDashboardResponse{}

	// Inventario
	inv, err := h.queries.ObtenerStatsInventario(c.Request.Context())
	if err != nil {
		log.Printf("AdminDashboard: error inventario: %v", err)
	} else {
		resp.Inventario.TotalVariantes = inv.TotalVariantes
		resp.Inventario.SinStock = inv.SinStock
		resp.Inventario.StockBajo = inv.StockBajo
		resp.Inventario.ConStock = inv.TotalVariantes - inv.SinStock - inv.StockBajo
	}

	// Pedidos
	ped, err := h.queries.ObtenerStatsPedidos(c.Request.Context())
	if err != nil {
		log.Printf("AdminDashboard: error pedidos: %v", err)
	} else {
		resp.Pedidos.TotalPedidos = ped.TotalPedidos
		monto, _ := utils.NumericToFloat64(ped.MontoTotalOrdenes)
		resp.Pedidos.MontoTotalOrdenes = monto
	}

	// Backorders
	bo, err := h.queries.ObtenerStatsBackorders(c.Request.Context())
	if err != nil {
		log.Printf("AdminDashboard: error backorders: %v", err)
	} else {
		resp.Backorders.TotalBackorders = bo.TotalBackorders
		monto, _ := utils.NumericToFloat64(bo.MontoRetenido)
		resp.Backorders.MontoRetenido = monto
	}

	// Devoluciones
	dv, err := h.queries.ObtenerStatsDevoluciones(c.Request.Context())
	if err != nil {
		log.Printf("AdminDashboard: error devoluciones: %v", err)
	} else {
		resp.Devoluciones.Pendientes = dv.Pendientes
		resp.Devoluciones.Aprobadas = dv.Aprobadas
		resp.Devoluciones.Rechazadas = dv.Rechazadas
		resp.Devoluciones.Canceladas = dv.Canceladas
		resp.Devoluciones.Total = dv.Pendientes + dv.Aprobadas + dv.Rechazadas + dv.Canceladas
	}

	// Facturas
	fac, err := h.queries.ObtenerStatsFacturas(c.Request.Context())
	if err != nil {
		log.Printf("AdminDashboard: error facturas: %v", err)
	} else {
		cv, _ := utils.NumericToFloat64(fac.CarteraVencida)
		resp.Facturas.CarteraVencida = cv
		spt, _ := utils.NumericToFloat64(fac.SaldoPendienteTotal)
		resp.Facturas.SaldoPendienteTotal = spt
	}

	// Ingresos recientes
	ing, err := h.queries.ObtenerIngresosRecientes(c.Request.Context())
	if err != nil {
		log.Printf("AdminDashboard: error ingresos: %v", err)
	} else {
		ti, _ := utils.NumericToFloat64(ing.TotalIngresos)
		resp.Ingresos.TotalIngresos = ti
		resp.Ingresos.TotalPagos = ing.TotalPagos
	}

	// Ingresos mensuales (sparkline data)
	ingMensuales, err := h.queries.ObtenerIngresosMensuales(c.Request.Context())
	if err != nil {
		log.Printf("AdminDashboard: error ingresos mensuales: %v", err)
	} else {
		for _, im := range ingMensuales {
			total, _ := utils.NumericToFloat64(im.Total)
			resp.IngresosMensuales = append(resp.IngresosMensuales, IngresoMensual{
				Anio:  im.Anio,
				Mes:   im.Mes,
				Total: total,
			})
		}
	}

	c.JSON(http.StatusOK, resp)
}
