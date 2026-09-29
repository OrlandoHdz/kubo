-- internal/sql/productos_recompensa.sql
-- Espejo exacto de productos.sql para el catálogo de recompensas

-- ==========================================
-- CRUD PRODUCTOS RECOMPENSA PADRE
-- ==========================================

-- name: ExisteProductoRecompensaPadreEliminadoPorCve :one
SELECT EXISTS(
    SELECT 1 FROM productos_recompensa_padre
    WHERE cve_prod_integracion = $1 AND deleted_at IS NOT NULL
) AS existe;

-- name: CrearProductoRecompensaPadre :one
INSERT INTO productos_recompensa_padre (
    cve_prod_integracion,
    descripcion,
    titulo,
    descripcion_extendida,
    foto_url,
    foto_url2,
    foto_url_3,
    foto_url_4,
    foto_url_5,
    foto_url_6,
    foto_url_7,
    foto_url_8,
    ficha_tecnica,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: GetProductoRecompensaPadre :one
SELECT p.id, p.cve_prod_integracion, p.titulo, p.descripcion, p.descripcion_extendida, p.foto_url, p.foto_url2, p.foto_url_3, p.foto_url_4, p.foto_url_5, p.foto_url_6, p.foto_url_7, p.foto_url_8, p.ficha_tecnica, p.created_at, p.updated_at, p.deleted_at, p.created_by, p.updated_by, p.deleted_by,
       COALESCE(pi.nom_prod, '') AS nombre_tecnico
FROM productos_recompensa_padre p
LEFT JOIN productos_integracion pi ON p.cve_prod_integracion = pi.cve_prod
WHERE p.id = $1 AND p.deleted_at IS NULL LIMIT 1;

-- name: GetProductoRecompensaPadreByDescripcion :one
SELECT p.id, p.cve_prod_integracion, p.titulo, p.descripcion, p.descripcion_extendida, p.foto_url, p.foto_url2, p.foto_url_3, p.foto_url_4, p.foto_url_5, p.foto_url_6, p.foto_url_7, p.foto_url_8, p.ficha_tecnica, p.created_at, p.updated_at, p.deleted_at, p.created_by, p.updated_by, p.deleted_by,
       COALESCE(pi.nom_prod, '') AS nombre_tecnico
FROM productos_recompensa_padre p
LEFT JOIN productos_integracion pi ON p.cve_prod_integracion = pi.cve_prod
WHERE p.descripcion = $1 AND p.deleted_at IS NULL LIMIT 1;

-- name: ListarProductosRecompensaPadre :many
SELECT p.id, p.cve_prod_integracion, p.titulo, p.descripcion, p.descripcion_extendida, p.foto_url, p.foto_url2, p.foto_url_3, p.foto_url_4, p.foto_url_5, p.foto_url_6, p.foto_url_7, p.foto_url_8, p.ficha_tecnica, p.created_at, p.updated_at, p.deleted_at, p.created_by, p.updated_by, p.deleted_by,
       COALESCE(pi.nom_prod, '') AS nombre_tecnico
FROM productos_recompensa_padre p
LEFT JOIN productos_integracion pi ON p.cve_prod_integracion = pi.cve_prod
WHERE p.deleted_at IS NULL;

-- name: ActualizarProductoRecompensaPadre :one
UPDATE productos_recompensa_padre
SET
    cve_prod_integracion = $2,
    descripcion = $3,
    titulo = $4,
    descripcion_extendida = $5,
    foto_url = $6,
    foto_url2 = $7,
    foto_url_3 = $8,
    foto_url_4 = $9,
    foto_url_5 = $10,
    foto_url_6 = $11,
    foto_url_7 = $12,
    foto_url_8 = $13,
    ficha_tecnica = $14,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $15
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteProductoRecompensaPadre :exec
UPDATE productos_recompensa_padre
SET
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1;

-- ==========================================
-- CRUD VARIANTES DE RECOMPENSA
-- ==========================================

-- name: CrearVarianteRecompensa :one
INSERT INTO productos_recompensa_variantes (
    padre_id, sku, medida, precio_puntos,
    stock_actual, unidad_medida, lead_time_dias, especificaciones,
    categoria, subgrupo, modelo, tipo, marca, multiplos, permitir_backorder, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) RETURNING *;

-- name: GetVarianteRecompensaBySKU :one
SELECT * FROM productos_recompensa_variantes
WHERE sku = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetVarianteRecompensa :one
SELECT * FROM productos_recompensa_variantes
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetVarianteRecompensaConExistencia :one
SELECT v.id, v.padre_id, v.sku, v.multiplos, v.permitir_backorder,
       CAST(COALESCE(ei.existencia_total, 0::numeric) AS numeric(20,8)) AS existencia_total
FROM productos_recompensa_variantes v
LEFT JOIN productos_recompensa_padre p ON v.padre_id = p.id
LEFT JOIN LATERAL (
    SELECT SUM(existencia) AS existencia_total
    FROM existencias_integracion
    WHERE cve_prod = p.cve_prod_integracion
) ei ON true
WHERE v.id = $1 AND v.deleted_at IS NULL LIMIT 1;

-- name: ListarVariantesRecompensaPorPadre :many
SELECT v.id, v.padre_id, v.sku, v.medida, v.precio_puntos, v.stock_actual, v.unidad_medida, v.lead_time_dias, v.especificaciones, v.categoria, v.subgrupo, v.modelo, v.tipo, v.marca, v.multiplos, v.permitir_backorder, v.created_at, v.updated_at, v.deleted_at, v.created_by, v.updated_by, v.deleted_by,
       p.descripcion AS padre_descripcion,
       p.titulo AS padre_titulo,
       p.descripcion_extendida AS padre_descripcion_extendida,
       CAST(COALESCE(ei.existencia_total, 0::numeric) AS numeric(20,8)) AS existencia_total
FROM productos_recompensa_variantes v
LEFT JOIN productos_recompensa_padre p ON v.padre_id = p.id
LEFT JOIN LATERAL (
    SELECT SUM(existencia) AS existencia_total
    FROM existencias_integracion
    WHERE cve_prod = p.cve_prod_integracion
) ei ON true
WHERE v.padre_id = $1 AND v.deleted_at IS NULL;

-- name: GetStockRealVarianteRecompensa :one
SELECT COALESCE(SUM(ei.existencia), 0)::numeric(20,8) AS stock_real
FROM productos_recompensa_variantes v
JOIN productos_recompensa_padre p ON v.padre_id = p.id
LEFT JOIN existencias_integracion ei ON p.cve_prod_integracion = ei.cve_prod
WHERE v.id = $1 AND v.deleted_at IS NULL;

-- name: ActualizarStockVarianteRecompensa :exec
UPDATE productos_recompensa_variantes
SET
    stock_actual = $2,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $1;

-- name: ActualizarVarianteRecompensa :one
UPDATE productos_recompensa_variantes
SET
    padre_id = $2,
    sku = $3,
    medida = $4,
    precio_puntos = $5,
    stock_actual = $6,
    unidad_medida = $7,
    lead_time_dias = $8,
    especificaciones = $9,
    categoria = $10,
    subgrupo = $11,
    modelo = $12,
    tipo = $13,
    marca = $14,
    multiplos = $15,
    permitir_backorder = $16,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $17
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteVarianteRecompensa :exec
UPDATE productos_recompensa_variantes
SET
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1;
