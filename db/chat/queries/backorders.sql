-- Consultas de backorders para el chat. Solo lectura, filtradas por cliente.

-- name: ObtenerBackorderCliente :one
SELECT
    id,
    folio,
    estado_backorder,
    fecha_backorder,
    total_orden,
    guia_backorder,
    pedido_origen_folio
FROM chat_backorders
WHERE folio = sqlc.arg(folio)::text
  AND cliente_id = sqlc.arg(cliente_id)::int
LIMIT 1;

-- name: ListarBackorderItems :many
SELECT
    sku,
    nombre,
    cantidad,
    precio_unitario_aplicado,
    disponible
FROM chat_backorder_items
WHERE backorder_id = sqlc.arg(backorder_id)::int
  AND cliente_id = sqlc.arg(cliente_id)::int
ORDER BY id
LIMIT 50;

-- name: ListarBackordersCliente :many
SELECT
    folio,
    estado_backorder,
    fecha_backorder,
    total_orden,
    guia_backorder,
    pedido_origen_folio
FROM chat_backorders
WHERE cliente_id = sqlc.arg(cliente_id)::int
ORDER BY fecha_backorder DESC
LIMIT LEAST(GREATEST(sqlc.arg(limite)::int, 1), 10);
