package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/OrlandoHdz/kubo/internal/chat/chatdb"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgReadStore struct {
	pool *pgxpool.Pool
	q    *chatdb.Queries
}

func newPGReadStore(pool *pgxpool.Pool) *pgReadStore {
	return &pgReadStore{pool: pool, q: chatdb.New(pool)}
}

func (s *pgReadStore) BuscarProductos(ctx context.Context, in BuscarProductosInput) ([]ProductoResumen, error) {
	params := chatdb.BuscarProductosParams{
		EsDistribuidor: in.EsDistribuidor,
		Consulta:       in.Consulta,
		Proceso:        textValue(in.Proceso),
		Filtros:        in.Filtros,
		AmperajeMin:    float8Value(in.AmperajeMin),
		AmperajeMax:    float8Value(in.AmperajeMax),
		EnStock:        boolValue(in.EnStock),
	}

	rows, err := s.q.BuscarProductos(ctx, params)
	if err != nil {
		return nil, err
	}

	out := make([]ProductoResumen, 0, len(rows))
	for _, row := range rows {
		out = append(out, ProductoResumen{
			ID:               row.ID,
			PadreID:          row.PadreID,
			SKU:              row.Sku,
			Nombre:           row.Nombre,
			Imagen:           row.ImagenUrl,
			Precio:           row.Precio,
			Moneda:           row.Moneda,
			Proceso:          row.Proceso,
			Atributos:        rawJSON(row.Atributos),
			Stock:            row.Stock,
			EnStock:          row.EnStock,
			Marca:            row.Marca,
			Categoria:        row.Categoria,
			Descripcion:      row.Descripcion,
			Especificaciones: row.Especificaciones,
		})
	}
	return out, nil
}

func (s *pgReadStore) ObtenerProductoPorSKU(ctx context.Context, sku string, esDistribuidor bool) (*ProductoDetalle, error) {
	row, err := s.q.ObtenerProductoPorSKU(ctx, chatdb.ObtenerProductoPorSKUParams{
		EsDistribuidor: esDistribuidor,
		Sku:            sku,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &ProductoDetalle{
		ProductoResumen: ProductoResumen{
			ID:               row.ID,
			PadreID:          row.PadreID,
			SKU:              row.Sku,
			Nombre:           row.Nombre,
			Imagen:           row.ImagenUrl,
			Precio:           row.Precio,
			Moneda:           row.Moneda,
			Proceso:          row.Proceso,
			Atributos:        rawJSON(row.Atributos),
			Stock:            row.Stock,
			EnStock:          row.EnStock,
			Marca:            row.Marca,
			Categoria:        row.Categoria,
			Descripcion:      truncateText(row.Descripcion, maxDetailChars),
			Especificaciones: row.Especificaciones,
		},
		DescripcionCorta:  truncateText(row.DescripcionCorta, maxDetailChars),
		FichaTecnica:      row.FichaTecnica,
		Medida:            row.Medida,
		UnidadMedida:      row.UnidadMedida,
		Subgrupo:          row.Subgrupo,
		Modelo:            row.Modelo,
		Tipo:              row.Tipo,
		Multiplos:         row.Multiplos,
		LeadTimeDias:      row.LeadTimeDias,
		PermitirBackorder: row.PermitirBackorder,
	}, nil
}

func (s *pgReadStore) ObtenerContexto(ctx context.Context, usuarioID int32) (ContextoCliente, error) {
	row, err := s.q.ObtenerContextoCliente(ctx, usuarioID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ContextoCliente{}, nil
	}
	if err != nil {
		return ContextoCliente{}, err
	}

	var clienteID *int32
	if row.ClienteID.Valid {
		v := row.ClienteID.Int32
		clienteID = &v
	}
	return ContextoCliente{
		ClienteID:      clienteID,
		EsDistribuidor: row.TienePrecioDistribuidor,
	}, nil
}

func (s *pgReadStore) withClienteTx(ctx context.Context, clienteID int32, fn func(q *chatdb.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)
	if err := qtx.SetCustomerContext(ctx, strconv.FormatInt(int64(clienteID), 10)); err != nil {
		return err
	}
	if err := fn(qtx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *pgReadStore) ConCliente(ctx context.Context, clienteID int32, fn func(PedidosQuerier) error) error {
	return s.withClienteTx(ctx, clienteID, func(q *chatdb.Queries) error {
		return fn(&pgPedidosQuerier{q: q, clienteID: clienteID})
	})
}

func (s *pgReadStore) ConClienteBackorders(ctx context.Context, clienteID int32, fn func(BackordersQuerier) error) error {
	return s.withClienteTx(ctx, clienteID, func(q *chatdb.Queries) error {
		return fn(&pgBackordersQuerier{q: q, clienteID: clienteID})
	})
}

func (s *pgReadStore) ConClienteDevoluciones(ctx context.Context, clienteID int32, fn func(DevolucionesQuerier) error) error {
	return s.withClienteTx(ctx, clienteID, func(q *chatdb.Queries) error {
		return fn(&pgDevolucionesQuerier{q: q, clienteID: clienteID})
	})
}

func (s *pgReadStore) ConClienteRecompensas(ctx context.Context, clienteID int32, fn func(RecompensasQuerier) error) error {
	return s.withClienteTx(ctx, clienteID, func(q *chatdb.Queries) error {
		return fn(&pgRecompensasQuerier{q: q, clienteID: clienteID})
	})
}

type pgPedidosQuerier struct {
	q         *chatdb.Queries
	clienteID int32
}

func (p *pgPedidosQuerier) ObtenerPedido(ctx context.Context, folio string, clienteID int32) (*PedidoResumen, error) {
	if clienteID != p.clienteID {
		return nil, fmt.Errorf("cliente inconsistente en la consulta de pedido")
	}
	row, err := p.q.ObtenerPedidoCliente(ctx, chatdb.ObtenerPedidoClienteParams{
		Folio:     folio,
		ClienteID: p.clienteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &PedidoResumen{
		ID:              row.ID,
		Folio:           row.Folio,
		Estado:          row.Estado,
		FechaPedido:     row.FechaPedido,
		Total:           row.TotalOrden,
		Moneda:          "MXN",
		Guia:            row.Guia,
		Paqueteria:      row.Paqueteria,
		EntregaEstimada: row.FechaEntregaEstimada,
		HasBackorder:    row.HasBackorder,
	}, nil
}

func (p *pgPedidosQuerier) ListarItems(ctx context.Context, pedidoID int32, clienteID int32) ([]PedidoItem, error) {
	if clienteID != p.clienteID {
		return nil, fmt.Errorf("cliente inconsistente en la consulta de items")
	}
	rows, err := p.q.ListarPedidoItems(ctx, chatdb.ListarPedidoItemsParams{
		PedidoID:  pedidoID,
		ClienteID: p.clienteID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]PedidoItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, PedidoItem{
			SKU:            row.Sku,
			Nombre:         row.Nombre,
			Cantidad:       row.Cantidad,
			PrecioUnitario: row.PrecioUnitarioAplicado,
			Enviado:        row.ShippedQuantity,
			Backorder:      row.BackorderQuantity,
		})
	}
	return out, nil
}

func (p *pgPedidosQuerier) ListarPedidos(ctx context.Context, clienteID int32, limite int) ([]PedidoResumen, error) {
	if clienteID != p.clienteID {
		return nil, fmt.Errorf("cliente inconsistente en el listado de pedidos")
	}
	rows, err := p.q.ListarPedidosCliente(ctx, chatdb.ListarPedidosClienteParams{
		ClienteID: p.clienteID,
		Limite:    int32(limite),
	})
	if err != nil {
		return nil, err
	}
	out := make([]PedidoResumen, 0, len(rows))
	for _, row := range rows {
		out = append(out, PedidoResumen{
			Folio:           row.Folio,
			Estado:          row.Estado,
			FechaPedido:     row.FechaPedido,
			Total:           row.TotalOrden,
			Moneda:          "MXN",
			Guia:            row.Guia,
			Paqueteria:      row.Paqueteria,
			EntregaEstimada: row.FechaEntregaEstimada,
			HasBackorder:    row.HasBackorder,
		})
	}
	return out, nil
}

type pgBackordersQuerier struct {
	q         *chatdb.Queries
	clienteID int32
}

func (p *pgBackordersQuerier) ObtenerBackorder(ctx context.Context, folio string, clienteID int32) (*BackorderResumen, error) {
	if clienteID != p.clienteID {
		return nil, fmt.Errorf("cliente inconsistente en la consulta de backorder")
	}
	row, err := p.q.ObtenerBackorderCliente(ctx, chatdb.ObtenerBackorderClienteParams{
		Folio:     folio,
		ClienteID: p.clienteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &BackorderResumen{
		ID:                row.ID,
		Folio:             row.Folio,
		Estado:            row.EstadoBackorder,
		Fecha:             row.FechaBackorder,
		Total:             row.TotalOrden,
		Moneda:            "MXN",
		Guia:              row.GuiaBackorder,
		PedidoOrigenFolio: row.PedidoOrigenFolio,
	}, nil
}

func (p *pgBackordersQuerier) ListarBackorderItems(ctx context.Context, backorderID int32, clienteID int32) ([]BackorderItem, error) {
	if clienteID != p.clienteID {
		return nil, fmt.Errorf("cliente inconsistente en la consulta de items de backorder")
	}
	rows, err := p.q.ListarBackorderItems(ctx, chatdb.ListarBackorderItemsParams{
		BackorderID: backorderID,
		ClienteID:   p.clienteID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]BackorderItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, BackorderItem{
			SKU:            row.Sku,
			Nombre:         row.Nombre,
			Cantidad:       row.Cantidad,
			PrecioUnitario: row.PrecioUnitarioAplicado,
			Disponible:     row.Disponible,
		})
	}
	return out, nil
}

func (p *pgBackordersQuerier) ListarBackorders(ctx context.Context, clienteID int32, limite int) ([]BackorderResumen, error) {
	if clienteID != p.clienteID {
		return nil, fmt.Errorf("cliente inconsistente en el listado de backorders")
	}
	rows, err := p.q.ListarBackordersCliente(ctx, chatdb.ListarBackordersClienteParams{
		ClienteID: p.clienteID,
		Limite:    int32(limite),
	})
	if err != nil {
		return nil, err
	}
	out := make([]BackorderResumen, 0, len(rows))
	for _, row := range rows {
		out = append(out, BackorderResumen{
			Folio:             row.Folio,
			Estado:            row.EstadoBackorder,
			Fecha:             row.FechaBackorder,
			Total:             row.TotalOrden,
			Moneda:            "MXN",
			Guia:              row.GuiaBackorder,
			PedidoOrigenFolio: row.PedidoOrigenFolio,
		})
	}
	return out, nil
}

type pgDevolucionesQuerier struct {
	q         *chatdb.Queries
	clienteID int32
}

func (p *pgDevolucionesQuerier) ObtenerDevolucion(ctx context.Context, folio string, clienteID int32) (*DevolucionResumen, error) {
	if clienteID != p.clienteID {
		return nil, fmt.Errorf("cliente inconsistente en la consulta de devolución")
	}
	row, err := p.q.ObtenerDevolucionCliente(ctx, chatdb.ObtenerDevolucionClienteParams{
		Folio:     folio,
		ClienteID: p.clienteID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	resumen := newDevolucionResumen(
		row.Folio, row.PedidoFolio, row.Tipo, row.NumerosParte, row.Cantidades,
		row.Estatus, row.FechaCreacion, row.FechaActualizacion,
	)
	return &resumen, nil
}

func (p *pgDevolucionesQuerier) ListarDevoluciones(ctx context.Context, clienteID int32, limite int) ([]DevolucionResumen, error) {
	if clienteID != p.clienteID {
		return nil, fmt.Errorf("cliente inconsistente en el listado de devoluciones")
	}
	rows, err := p.q.ListarDevolucionesCliente(ctx, chatdb.ListarDevolucionesClienteParams{
		ClienteID: p.clienteID,
		Limite:    int32(limite),
	})
	if err != nil {
		return nil, err
	}
	out := make([]DevolucionResumen, 0, len(rows))
	for _, row := range rows {
		out = append(out, newDevolucionResumen(
			row.Folio, row.PedidoFolio, row.Tipo, row.NumerosParte, row.Cantidades,
			row.Estatus, row.FechaCreacion, row.FechaActualizacion,
		))
	}
	return out, nil
}

func newDevolucionResumen(
	folio, pedidoFolio, tipo, numerosParte, cantidades, estatus, fechaCreacion, fechaActualizacion string,
) DevolucionResumen {
	return DevolucionResumen{
		Folio:              folio,
		PedidoFolio:        pedidoFolio,
		Tipo:               tipo,
		NumerosParte:       truncateText(numerosParte, 600),
		Cantidades:         truncateText(cantidades, 600),
		Estatus:            estatus,
		FechaCreacion:      fechaCreacion,
		FechaActualizacion: fechaActualizacion,
	}
}

type pgRecompensasQuerier struct {
	q         *chatdb.Queries
	clienteID int32
}

func (p *pgRecompensasQuerier) ObtenerRecompensasCliente(ctx context.Context, clienteID int32) (*RecompensasCliente, error) {
	if clienteID != p.clienteID {
		return nil, fmt.Errorf("cliente inconsistente en la consulta de recompensas")
	}
	row, err := p.q.ObtenerRecompensasCliente(ctx, p.clienteID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	out := &RecompensasCliente{
		SaldoPuntos:             row.SaldoPuntos,
		MontoCompraPorPunto:     row.MontoCompraPorPunto,
		Compra90Dias:            row.Compra90dias,
		PromedioTrimestral:      row.PromedioTrimestral,
		NivelManualFijado:       row.NivelManualFijado,
		FechaActualizacionNivel: row.FechaActualizacionNivel,
	}
	if row.NivelID.Valid && row.NivelNombre != "" {
		out.NivelActual = &NivelRecompensa{
			ID:                       row.NivelID.Int32,
			Nombre:                   row.NivelNombre,
			NumeroNivel:              row.NivelNumero,
			PromedioCompraTrimestral: row.NivelPromedioTrimestral,
			PorcentajeDescuento:      row.NivelDescuento,
		}
	}
	return out, nil
}

func (p *pgRecompensasQuerier) ListarNivelesRecompensa(ctx context.Context) ([]NivelRecompensa, error) {
	rows, err := p.q.ListarNivelesRecompensa(ctx)
	if err != nil {
		return nil, err
	}
	return mapNiveles(rows), nil
}

func (p *pgRecompensasQuerier) ListarTransaccionesPuntos(ctx context.Context, clienteID int32, limite int) ([]TransaccionPuntos, error) {
	if clienteID != p.clienteID {
		return nil, fmt.Errorf("cliente inconsistente en el listado de puntos")
	}
	rows, err := p.q.ListarTransaccionesPuntos(ctx, chatdb.ListarTransaccionesPuntosParams{
		ClienteID: p.clienteID,
		Limite:    int32(limite),
	})
	if err != nil {
		return nil, err
	}
	out := make([]TransaccionPuntos, 0, len(rows))
	for _, row := range rows {
		out = append(out, TransaccionPuntos{
			Tipo:        row.Tipo,
			Puntos:      row.Puntos,
			Fecha:       row.Fecha,
			PedidoFolio: row.PedidoFolio,
		})
	}
	return out, nil
}

func (s *pgReadStore) ListarNivelesRecompensa(ctx context.Context) (NivelesRecompensaResult, error) {
	levels, err := s.q.ListarNivelesRecompensa(ctx)
	if err != nil {
		return NivelesRecompensaResult{}, err
	}
	params, err := s.q.ObtenerParametrosRecompensa(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		params = 0
	} else if err != nil {
		return NivelesRecompensaResult{}, err
	}
	return NivelesRecompensaResult{
		Niveles:             mapNiveles(levels),
		MontoCompraPorPunto: params,
	}, nil
}

func mapNiveles(rows []chatdb.ChatNivelesRecompensa) []NivelRecompensa {
	out := make([]NivelRecompensa, 0, len(rows))
	for _, row := range rows {
		out = append(out, NivelRecompensa{
			ID:                       row.ID,
			Nombre:                   row.Nombre,
			NumeroNivel:              row.NumeroNivel,
			PromedioCompraTrimestral: row.PromedioCompraTrimestral,
			PorcentajeDescuento:      row.PorcentajeDescuento,
		})
	}
	return out
}

type pgAuditStore struct {
	q *chatdb.Queries
}

func newPGAuditStore(pool *pgxpool.Pool) *pgAuditStore {
	return &pgAuditStore{q: chatdb.New(pool)}
}

func (a *pgAuditStore) CrearConversacion(ctx context.Context, id string, usuarioID, clienteID *int32) error {
	return a.q.CrearConversacion(ctx, chatdb.CrearConversacionParams{
		ID:        id,
		UsuarioID: int4Value(usuarioID),
		ClienteID: int4Value(clienteID),
	})
}

func (a *pgAuditStore) CrearMensaje(
	ctx context.Context,
	conversationID string,
	usuarioID *int32,
	role, content string,
	toolCalls json.RawMessage,
	inputTokens, outputTokens int32,
) error {
	return a.q.CrearMensaje(ctx, chatdb.CrearMensajeParams{
		ConversationID: conversationID,
		UsuarioID:      int4Value(usuarioID),
		Role:           role,
		Content:        content,
		ToolCalls:      toolCalls,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
	})
}

func (a *pgAuditStore) CrearEscalamiento(ctx context.Context, conversationID string, usuarioID *int32, motivo, nombre, email, telefono string) error {
	return a.q.CrearEscalamiento(ctx, chatdb.CrearEscalamientoParams{
		ConversationID:   conversationID,
		UsuarioID:        int4Value(usuarioID),
		Motivo:           motivo,
		ContactoNombre:   nombre,
		ContactoEmail:    email,
		ContactoTelefono: telefono,
	})
}

func textValue(v string) pgtype.Text {
	return pgtype.Text{String: v, Valid: v != ""}
}

func float8Value(v *float64) pgtype.Float8 {
	if v == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *v, Valid: true}
}

func boolValue(v *bool) pgtype.Bool {
	if v == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *v, Valid: true}
}

func int4Value(v *int32) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *v, Valid: true}
}

func rawJSON(v json.RawMessage) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage(`{}`)
	}
	return v
}

const maxDetailChars = 2000

func truncateText(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
