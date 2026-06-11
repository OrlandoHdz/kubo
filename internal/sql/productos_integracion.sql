-- name: CreateProducto :one
INSERT INTO productos_integracion (
    cse_prod, cve_prod, nom_prod, desc_prod, uni_med, uni_med_p, 
    costo_prod, f_act_cto, f_act_pre, cto_ent, fec_ent, fec_ant
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetProductoByID :one
SELECT * FROM productos_integracion
WHERE id = $1 LIMIT 1;

-- name: GetProductoByClave :one
-- Aprovecha el índice UNIQUE que definiste en cve_prod
SELECT * FROM productos_integracion
WHERE cve_prod = $1 LIMIT 1;

-- name: ListProductos :many
SELECT * FROM productos_integracion
ORDER BY nom_prod ASC;

-- name: UpdateProducto :one
UPDATE productos_integracion
SET 
    cse_prod = $2,
    nom_prod = $3,
    desc_prod = $4,
    uni_med = $5,
    uni_med_p = $6
WHERE id = $1
RETURNING *;

-- name: UpdateCostos :one
-- Consulta específica para cuando se actualizan precios/costos y sus fechas
UPDATE productos_integracion
SET 
    costo_prod = $2,
    f_act_cto = $3,
    cto_ent = $4,
    fec_ent = $5,
    fec_ant = $6
WHERE id = $1
RETURNING *;

-- name: UpsertProducto :one
INSERT INTO productos_integracion (
    cse_prod, cve_prod, nom_prod, desc_prod, uni_med, uni_med_p, 
    costo_prod, f_act_cto, f_act_pre, cto_ent, fec_ent, fec_ant
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (cve_prod) DO UPDATE SET
    cse_prod = EXCLUDED.cse_prod,
    nom_prod = EXCLUDED.nom_prod,
    desc_prod = EXCLUDED.desc_prod,
    uni_med = EXCLUDED.uni_med,
    uni_med_p = EXCLUDED.uni_med_p,
    costo_prod = EXCLUDED.costo_prod,
    f_act_cto = EXCLUDED.f_act_cto,
    f_act_pre = EXCLUDED.f_act_pre,
    cto_ent = EXCLUDED.cto_ent,
    fec_ent = EXCLUDED.fec_ent,
    fec_ant = EXCLUDED.fec_ant
RETURNING *;


