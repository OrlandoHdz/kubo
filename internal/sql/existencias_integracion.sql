-- name: CrearExistenciaIntegracion :one
INSERT INTO existencias_integracion (
    cse_prod, cve_prod, lugar, existencia, med_prod, fech_umod, 
    inv_ini, lote, fech_lote, ref_lote, costo_prom, new_med, 
    costuepeps, costoprom2
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: ObtenerExistenciaPorProductoYLugar :one
SELECT * FROM existencias_integracion
WHERE cve_prod = $1 AND lugar = $2 LIMIT 1;

-- name: ObtenerTodasLasExistencias :many
SELECT * FROM existencias_integracion;

-- name: ObtenerExistenciasPorProducto :many
SELECT * FROM existencias_integracion
WHERE cve_prod = $1;

-- name: UpsertExistenciaIntegracion :one
INSERT INTO existencias_integracion (
    cse_prod, cve_prod, lugar, existencia, med_prod, fech_umod, 
    inv_ini, lote, fech_lote, ref_lote, costo_prom, new_med, 
    costuepeps, costoprom2
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
ON CONFLICT (cve_prod, lugar) DO UPDATE SET
    cse_prod = EXCLUDED.cse_prod,
    existencia = EXCLUDED.existencia,
    med_prod = EXCLUDED.med_prod,
    fech_umod = EXCLUDED.fech_umod,
    inv_ini = EXCLUDED.inv_ini,
    lote = EXCLUDED.lote,
    fech_lote = EXCLUDED.fech_lote,
    ref_lote = EXCLUDED.ref_lote,
    costo_prom = EXCLUDED.costo_prom,
    new_med = EXCLUDED.new_med,
    costuepeps = EXCLUDED.costuepeps,
    costoprom2 = EXCLUDED.costoprom2
RETURNING *;