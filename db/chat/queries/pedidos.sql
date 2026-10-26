-- Consultas de pedidos para el chat.
-- REGLA CRÍTICA: cliente_id proviene del contexto de sesión resuelto por el
-- servidor, nunca de argumentos del modelo. Todas las consultas filtran por él.

-- name: ObtenerContextoCliente :one
SELECT
    usuario_id,
    cliente_id,
    tiene_precio_distribuidor
FROM chat_usuarios_clientes
WHERE usuario_id = sqlc.arg(usuario_id)::int
LIMIT 1;

-- name: SetCustomerContext :exec
SELECT set_config(
    'app.customer_id',
    sqlc.arg(customer_id)::text,
    true
);

-- name: ObtenerPedidoCliente :one
SELECT
    id,
    folio,
    estado,
    fecha_pedido,
    total_orden,
    guia,
    paqueteria,
    fecha_entrega_estimada,
    has_backorder
FROM chat_pedidos
WHERE folio = sqlc.arg(folio)::text
  AND cliente_id = sqlc.arg(cliente_id)::int
LIMIT 1;

-- name: ListarPedidoItems :many
SELECT
    sku,
    nombre,
    cantidad,
    precio_unitario_aplicado,
    shipped_quantity,
    backorder_quantity
FROM chat_pedido_items
WHERE pedido_id = sqlc.arg(pedido_id)::int
  AND cliente_id = sqlc.arg(cliente_id)::int
ORDER BY id
LIMIT 50;

-- name: ListarPedidosCliente :many
SELECT
    folio,
    estado,
    fecha_pedido,
    total_orden,
    guia,
    paqueteria,
    fecha_entrega_estimada,
    has_backorder
FROM chat_pedidos
WHERE cliente_id = sqlc.arg(cliente_id)::int
ORDER BY fecha_pedido DESC
LIMIT LEAST(GREATEST(sqlc.arg(limite)::int, 1), 10);
