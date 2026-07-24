-- name: ObtenerStatsInventario :one
SELECT
    COUNT(*)::bigint AS total_variantes,
    COUNT(*) FILTER (WHERE stock_actual = 0)::bigint AS sin_stock,
    COUNT(*) FILTER (WHERE stock_actual > 0 AND stock_actual < 10)::bigint AS stock_bajo
FROM productos_variantes
WHERE deleted_at IS NULL;

-- name: ObtenerStatsPedidos :one
SELECT
    COUNT(*)::bigint AS total_pedidos,
    COALESCE(SUM(total_orden), 0)::DECIMAL(15,2) AS monto_total_ordenes
FROM pedidos
WHERE deleted_at IS NULL;

-- name: ObtenerStatsBackorders :one
SELECT
    COUNT(*)::bigint AS total_backorders,
    COALESCE(SUM(total_orden), 0)::DECIMAL(15,2) AS monto_retenido
FROM backorders
WHERE deleted_at IS NULL
  AND estado_backorder IN ('Pendiente', 'Stock Disponible');

-- name: ObtenerStatsDevoluciones :one
SELECT
    COUNT(*) FILTER (WHERE estatus = 'Pendiente')::bigint AS pendientes,
    COUNT(*) FILTER (WHERE estatus = 'Aprobada')::bigint AS aprobadas,
    COUNT(*) FILTER (WHERE estatus = 'Rechazada')::bigint AS rechazadas,
    COUNT(*) FILTER (WHERE estatus = 'Cancelada')::bigint AS canceladas
FROM devoluciones_garantias
WHERE deleted_at IS NULL;

-- name: ObtenerStatsFacturas :one
SELECT
    CAST(COALESCE(SUM(CASE WHEN f_pago < CURRENT_TIMESTAMP AND saldo_fac > 0 THEN saldo_fac ELSE 0 END), 0) AS DECIMAL(15,2)) AS cartera_vencida,
    CAST(COALESCE(SUM(CASE WHEN saldo_fac > 0 THEN saldo_fac ELSE 0 END), 0) AS DECIMAL(15,2)) AS saldo_pendiente_total
FROM facturas_integracion;

-- name: ObtenerIngresosRecientes :one
SELECT
    CAST(COALESCE(SUM(monto_total), 0) AS DECIMAL(15,2)) AS total_ingresos,
    COUNT(*)::bigint AS total_pagos
FROM pagos_facturas
WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '30 days';

-- name: ObtenerIngresosMensuales :many
SELECT
    EXTRACT(YEAR FROM created_at)::int AS anio,
    EXTRACT(MONTH FROM created_at)::int AS mes,
    CAST(COALESCE(SUM(monto_total), 0) AS DECIMAL(15,2)) AS total
FROM pagos_facturas
WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '6 months'
GROUP BY anio, mes
ORDER BY anio, mes;
