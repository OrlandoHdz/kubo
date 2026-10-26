-- Consultas de devoluciones y garantías para el chat. Solo lectura.

-- name: ObtenerDevolucionCliente :one
SELECT
    folio,
    pedido_folio,
    tipo,
    numeros_parte,
    cantidades,
    estatus,
    fecha_creacion,
    fecha_actualizacion
FROM chat_devoluciones
WHERE folio = sqlc.arg(folio)::text
  AND cliente_id = sqlc.arg(cliente_id)::int
LIMIT 1;

-- name: ListarDevolucionesCliente :many
SELECT
    folio,
    pedido_folio,
    tipo,
    numeros_parte,
    cantidades,
    estatus,
    fecha_creacion,
    fecha_actualizacion
FROM chat_devoluciones
WHERE cliente_id = sqlc.arg(cliente_id)::int
ORDER BY fecha_creacion DESC
LIMIT LEAST(GREATEST(sqlc.arg(limite)::int, 1), 10);
