-- name: CrearTransaccionBanregio :one
INSERT INTO transacciones_banregio (
    bnrg_monto_trans,
    bnrg_id_afiliacion,
    bnrg_fecha_local,
    bnrg_codigo_aut,
    bnrg_folio,
    bnrg_texto,
    bnrg_referencia,
    bnrg_id_mediop,
    bnrg_codigo_proc,
    bnrg_hora_local,
    bnrg_codigo_emisor,
    pedido_id,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetTransaccionBanregio :one
SELECT * FROM transacciones_banregio
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListarTransaccionesBanregio :many
SELECT * FROM transacciones_banregio
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListarTransaccionesPorCliente :many
SELECT t.*, p.folio AS pedido_folio
FROM transacciones_banregio t
JOIN pedidos p ON t.pedido_id = p.id
WHERE p.cliente_id = $1 AND t.deleted_at IS NULL AND p.deleted_at IS NULL
ORDER BY t.created_at DESC;

-- name: SoftDeleteTransaccionBanregio :exec
UPDATE transacciones_banregio
SET
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1 AND deleted_at IS NULL;
