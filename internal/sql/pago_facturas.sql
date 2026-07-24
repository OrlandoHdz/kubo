-- name: GetCveCteByClienteId :one
SELECT ci.cve_cte FROM clientes_integracion ci
JOIN clientes c ON c.rfc = ci.rfc_cte
WHERE c.id = $1 LIMIT 1;

-- name: ListFacturasEmitidasPorCveCte :many
SELECT * FROM facturas_integracion
WHERE cve_cte = $1 AND status_fac = 'Emitida'
ORDER BY falta_fac DESC;

-- name: ListFacturasEmitidasPorCveCteRango :many
SELECT * FROM facturas_integracion
WHERE cve_cte = $1 AND status_fac = 'Emitida'
  AND (falta_fac IS NULL OR (falta_fac >= $2 AND falta_fac <= $3))
ORDER BY falta_fac DESC;

-- name: CrearHistorialPagoFactura :one
INSERT INTO historial_facturas_pagadas (
    factura_id,
    cliente_id,
    no_factura,
    monto_pagado,
    metodo_pago,
    tarjeta_terminacion,
    respuesta_simulada,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: ListHistorialFacturasPagadas :many
SELECT * FROM historial_facturas_pagadas
ORDER BY created_at DESC;

-- name: ListHistorialFacturasPagadasRango :many
SELECT * FROM historial_facturas_pagadas
WHERE created_at >= $1
  AND created_at <= $2
ORDER BY created_at DESC;

-- name: GetHistorialFacturaPagada :one
SELECT * FROM historial_facturas_pagadas
WHERE id = $1 LIMIT 1;

-- name: GetFacturaIntegracionPorId :one
SELECT * FROM facturas_integracion
WHERE id = $1 LIMIT 1;

-- name: CrearPagoFactura :one
INSERT INTO pagos_facturas (
    cliente_id,
    metodo_pago,
    tarjeta_terminacion,
    monto_total,
    respuesta_simulada,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: CrearPagoFacturaDetalle :exec
INSERT INTO pagos_facturas_detalles (
    pago_id,
    factura_id,
    no_factura,
    monto_pagado
) VALUES (
    $1, $2, $3, $4
);

-- name: ListPagosFacturas :many
SELECT * FROM pagos_facturas
ORDER BY created_at DESC;

-- name: ListPagosFacturasRango :many
SELECT * FROM pagos_facturas
WHERE created_at >= $1
  AND created_at <= $2
ORDER BY created_at DESC;

-- name: GetPagoFactura :one
SELECT * FROM pagos_facturas
WHERE id = $1 LIMIT 1;

-- name: GetPagoFacturaDetalles :many
SELECT pfd.*, fi.falta_fac, fi.total_fac AS total_factura
FROM pagos_facturas_detalles pfd
LEFT JOIN facturas_integracion fi ON fi.id = pfd.factura_id
WHERE pfd.pago_id = $1
ORDER BY pfd.id;
