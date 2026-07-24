-- name: ObtenerEstadoCuentaPorCveCte :many
SELECT
    c.cve_cte,
    c.nom_cte,
    c.dia_cre AS dias_credito,
    c.lim_cre AS limite_credito,
    f.id AS factura_id,
    f.no_fac AS folio,
    f.falta_fac AS fecha_emision,
    f.f_pago AS fecha_vencimiento,
    f.total_fac AS monto_total,
    f.saldo_fac AS saldo_pendiente,
    f.status_fac AS estatus
FROM clientes_integracion c
INNER JOIN facturas_integracion f ON c.cve_cte = f.cve_cte
WHERE c.cve_cte = $1
ORDER BY f.falta_fac DESC;
