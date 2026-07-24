-- name: ObtenerDatosFinancierosDashboard :one
SELECT
    c.dia_cre AS dias_credito,
    c.lim_cre AS limite_credito,
    CAST(COALESCE(SUM(f.saldo_fac), 0) AS DECIMAL(15,2)) AS saldo_pendiente_total,
    CAST(COALESCE(SUM(CASE WHEN f.f_pago < CURRENT_TIMESTAMP AND f.saldo_fac > 0 THEN f.saldo_fac ELSE 0 END), 0) AS DECIMAL(15,2)) AS saldo_vencido
FROM clientes_integracion c
LEFT JOIN facturas_integracion f ON c.cve_cte = f.cve_cte
WHERE c.cve_cte = $1
GROUP BY c.dia_cre, c.lim_cre;

-- name: ObtenerPedidosActivosCount :one
SELECT COUNT(*)::bigint AS total
FROM pedidos
WHERE cliente_id = $1 AND deleted_at IS NULL
  AND estado NOT IN ('Entregado', 'Cancelado');

-- name: ObtenerBackordersActivosCount :one
SELECT COUNT(*)::bigint AS total
FROM backorders
WHERE cliente_id = $1 AND deleted_at IS NULL
  AND estado_backorder NOT IN ('Convertido', 'Cancelado');

-- name: ObtenerDevolucionesEnProcesoCount :one
SELECT COUNT(*)::bigint AS total
FROM devoluciones_garantias
WHERE cliente_id = $1 AND deleted_at IS NULL
  AND estatus != 'Cancelada';
