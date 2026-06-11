-- internal/sql/productos.sql

-- ==========================================
-- CRUD PRODUCTOS PADRE (Contenedores)
-- ==========================================

-- name: CrearProductoPadre :one
INSERT INTO productos_padre (
    cve_prod_integracion, 
    descripcion, 
    foto_url, 
    ficha_tecnica, 
    created_by
) VALUES (
    $1, 
    $2, 
    $3, 
    $4, 
    $5
) RETURNING *;

-- name: GetProductoPadre :one
SELECT p.id, p.cve_prod_integracion, p.descripcion, p.foto_url, p.ficha_tecnica, p.created_at, p.updated_at, p.deleted_at, p.created_by, p.updated_by, p.deleted_by, 
       COALESCE(pi.nom_prod, '') AS nombre_tecnico, 
       COALESCE(pi.cse_prod, '') AS categoria
FROM productos_padre p
LEFT JOIN productos_integracion pi ON p.cve_prod_integracion = pi.cve_prod
WHERE p.id = $1 AND p.deleted_at IS NULL LIMIT 1;

-- name: GetProductoPadreByDescripcion :one
SELECT p.id, p.cve_prod_integracion, p.descripcion, p.foto_url, p.ficha_tecnica, p.created_at, p.updated_at, p.deleted_at, p.created_by, p.updated_by, p.deleted_by, 
       COALESCE(pi.nom_prod, '') AS nombre_tecnico, 
       COALESCE(pi.cse_prod, '') AS categoria
FROM productos_padre p
LEFT JOIN productos_integracion pi ON p.cve_prod_integracion = pi.cve_prod
WHERE p.descripcion = $1 AND p.deleted_at IS NULL LIMIT 1;

-- name: ListarProductosPadre :many
SELECT p.id, p.cve_prod_integracion, p.descripcion, p.foto_url, p.ficha_tecnica, p.created_at, p.updated_at, p.deleted_at, p.created_by, p.updated_by, p.deleted_by, 
       COALESCE(pi.nom_prod, '') AS nombre_tecnico, 
       COALESCE(pi.cse_prod, '') AS categoria
FROM productos_padre p
LEFT JOIN productos_integracion pi ON p.cve_prod_integracion = pi.cve_prod
WHERE p.deleted_at IS NULL;

-- name: ActualizarProductoPadre :one
UPDATE productos_padre
SET 
    cve_prod_integracion = $2,
    descripcion = $3,
    foto_url = $4,
    ficha_tecnica = $5,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $6
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteProductoPadre :exec
-- Borrado lógico del contenedor principal (afecta la visibilidad del catálogo)
UPDATE productos_padre
SET 
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1;

-- ==========================================
-- CRUD VARIANTES (SKUs Específicos)
-- ==========================================

-- name: CrearVariante :one
-- Registra una variante (ej. Tornillo de 1/2") ligada a un Padre (cite: 187, 191)
INSERT INTO productos_variantes (
padre_id, sku, medida, precio_distribuidor, precio_lista, precio_publico, 
    stock_actual, unidad_medida, lead_time_dias, especificaciones, 
    categoria, subgrupo, modelo, tipo, marca, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) RETURNING *;

-- name: GetVarianteBySKU :one
SELECT * FROM productos_variantes 
WHERE sku = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetVariante :one
SELECT * FROM productos_variantes 
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListarVariantesPorPadre :many
-- Para mostrar todas las medidas de un mismo producto (cite: 184)
SELECT * FROM productos_variantes 
WHERE padre_id = $1 AND deleted_at IS NULL;

-- name: ActualizarStock :exec
-- Sincronización de inventario en tiempo real (cite: 43)
UPDATE productos_variantes
SET 
    stock_actual = $2,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $1;

-- name: ActualizarVariante :one
UPDATE productos_variantes
SET 
    padre_id = $2,
    sku = $3,
    medida = $4,
    precio_distribuidor = $5,
    precio_lista = $6,
    precio_publico = $7,
    stock_actual = $8,
    unidad_medida = $9,
    lead_time_dias = $10,
    especificaciones = $11,
    categoria = $12,
    subgrupo = $13,
    modelo = $14,
    tipo = $15,
    marca = $16,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $17
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteVariante :exec
UPDATE productos_variantes
SET 
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1;