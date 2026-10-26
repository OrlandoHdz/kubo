-- Consultas de recompensas y niveles para el chat. Solo lectura.

-- name: ObtenerRecompensasCliente :one
SELECT
    cliente_id,
    saldo_puntos,
    nivel_manual_fijado,
    fecha_actualizacion_nivel,
    nivel_id,
    nivel_nombre,
    nivel_numero,
    nivel_promedio_trimestral,
    nivel_descuento,
    compra_90dias,
    promedio_trimestral,
    monto_compra_por_punto,
    valor_punto_pesos
FROM chat_recompensas_cliente
WHERE cliente_id = sqlc.arg(cliente_id)::int
LIMIT 1;

-- name: ListarNivelesRecompensa :many
SELECT
    id,
    nombre,
    numero_nivel,
    promedio_compra_trimestral,
    porcentaje_descuento
FROM chat_niveles_recompensa
ORDER BY numero_nivel ASC;

-- name: ObtenerParametrosRecompensa :one
SELECT
    monto_compra_por_punto,
    valor_punto_pesos
FROM chat_parametros_recompensa
LIMIT 1;

-- name: ListarTransaccionesPuntos :many
SELECT
    tipo,
    puntos,
    fecha,
    pedido_folio
FROM chat_transacciones_puntos
WHERE cliente_id = sqlc.arg(cliente_id)::int
ORDER BY fecha DESC
LIMIT LEAST(GREATEST(sqlc.arg(limite)::int, 1), 20);
