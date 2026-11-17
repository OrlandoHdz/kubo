package chat

// BackorderResumen es la cabecera de un backorder visible por el cliente.
type BackorderResumen struct {
	ID                int32   `json:"-"`
	Folio             string  `json:"folio"`
	Estado            string  `json:"estado"`
	Fecha             string  `json:"fecha"`
	Total             float64 `json:"total"`
	Moneda            string  `json:"moneda"`
	Guia              string  `json:"guia,omitempty"`
	PedidoOrigenFolio string  `json:"pedido_origen_folio,omitempty"`
}

type BackorderItem struct {
	SKU            string  `json:"sku"`
	Nombre         string  `json:"nombre"`
	Cantidad       int32   `json:"cantidad"`
	PrecioUnitario float64 `json:"precio_unitario"`
	Disponible     bool    `json:"disponible"`
}

// DevolucionResumen expone solo datos de seguimiento, no notas internas.
type DevolucionResumen struct {
	Folio              string `json:"folio"`
	PedidoFolio        string `json:"pedido_folio"`
	Tipo               string `json:"tipo"`
	NumerosParte       string `json:"numeros_parte,omitempty"`
	Cantidades         string `json:"cantidades,omitempty"`
	Estatus            string `json:"estatus"`
	FechaCreacion      string `json:"fecha_creacion"`
	FechaActualizacion string `json:"fecha_actualizacion,omitempty"`
}

type NivelRecompensa struct {
	ID                       int32   `json:"-"`
	Nombre                   string  `json:"nombre"`
	NumeroNivel              int32   `json:"numero_nivel"`
	PromedioCompraTrimestral float64 `json:"promedio_compra_trimestral"`
	PorcentajeDescuento      float64 `json:"porcentaje_descuento"`
}

type TransaccionPuntos struct {
	Tipo        string `json:"tipo"`
	Puntos      int32  `json:"puntos"`
	Fecha       string `json:"fecha"`
	PedidoFolio string `json:"pedido_folio,omitempty"`
}

type RecompensasCliente struct {
	SaldoPuntos             int32               `json:"saldo_puntos"`
	MontoCompraPorPunto     float64             `json:"monto_compra_por_punto"`
	Compra90Dias            float64             `json:"compra_90dias"`
	PromedioTrimestral      float64             `json:"promedio_trimestral"`
	NivelActual             *NivelRecompensa    `json:"nivel_actual,omitempty"`
	NivelManualFijado       bool                `json:"nivel_manual_fijado"`
	FechaActualizacionNivel string              `json:"fecha_actualizacion_nivel,omitempty"`
	SiguienteNivel          *NivelRecompensa    `json:"siguiente_nivel,omitempty"`
	FaltanteParaSiguiente   float64             `json:"faltante_promedio_para_siguiente,omitempty"`
	Niveles                 []NivelRecompensa   `json:"niveles"`
	UltimasTransacciones    []TransaccionPuntos `json:"ultimas_transacciones,omitempty"`
}

type NivelesRecompensaResult struct {
	Niveles             []NivelRecompensa `json:"niveles"`
	MontoCompraPorPunto float64           `json:"monto_compra_por_punto"`
}
