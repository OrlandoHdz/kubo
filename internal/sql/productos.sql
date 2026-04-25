-- internal/sql/productos.sql

-- ==========================================
-- CRUD PRODUCTOS PADRE (Contenedores)
-- ==========================================

-- name: CrearProductoPadre :one
INSERT INTO productos_padre (
    nombre_tecnico, 
    descripcion, 
    categoria, 
    marca, 
    documentacion_url, 
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetProductoPadre :one
SELECT * FROM productos_padre 
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListarProductosPadre :many
SELECT * FROM productos_padre 
WHERE deleted_at IS NULL;

-- name: ActualizarProductoPadre :one
UPDATE productos_padre
SET 
    nombre_tecnico = $2,
    descripcion = $3,
    categoria = $4,
    marca = $5,
    documentacion_url = $6,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $7
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
    padre_id, 
    sku, 
    medida, 
    precio_lista, 
    stock_actual, 
    unidad_medida, 
    lead_time_dias, 
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetVarianteBySKU :one
SELECT * FROM productos_variantes 
WHERE sku = $1 AND deleted_at IS NULL LIMIT 1;

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

-- name: SoftDeleteVariante :exec
UPDATE productos_variantes
SET 
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1;