-- name: CrearAjusteInventario :one
INSERT INTO public.ajustes_inventario_recompensa (
    variante_id,
    tipo,
    cantidad,
    comentario,
    stock_anterior,
    stock_nuevo,
    usuario_id,
    usuario_nombre
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: ListarAjustesInventario :many
SELECT 
    a.id, a.variante_id, a.tipo, a.cantidad, a.comentario, a.stock_anterior, a.stock_nuevo, a.usuario_id, a.usuario_nombre, a.created_at,
    v.sku as variante_sku,
    COALESCE(p.descripcion, '')::text as producto_descripcion,
    COALESCE(p.titulo, '')::text as producto_titulo,
    COALESCE(p.cve_prod_integracion, '')::text as cve_prod,
    COALESCE(u.email, '')::text as usuario_email
FROM public.ajustes_inventario_recompensa a
LEFT JOIN public.productos_recompensa_variantes v ON a.variante_id = v.id
LEFT JOIN public.productos_recompensa_padre p ON v.padre_id = p.id
LEFT JOIN public.usuarios u ON a.usuario_id = u.id
WHERE (sqlc.narg('variante_id')::int IS NULL OR a.variante_id = sqlc.narg('variante_id'))
  AND (sqlc.narg('fecha_inicio')::timestamp IS NULL OR a.created_at >= sqlc.narg('fecha_inicio'))
  AND (sqlc.narg('fecha_fin')::timestamp IS NULL OR a.created_at <= sqlc.narg('fecha_fin'))
ORDER BY a.created_at DESC;

-- name: ListarVariantesRecompensaConProducto :many
SELECT 
    v.id, v.padre_id, v.sku, v.medida, v.precio_puntos, v.stock_actual, v.unidad_medida, v.lead_time_dias, v.especificaciones, v.categoria, v.subgrupo, v.modelo, v.tipo, v.marca, v.multiplos, v.permitir_backorder, v.created_at, v.updated_at, v.deleted_at, v.created_by, v.updated_by, v.deleted_by,
    COALESCE(p.descripcion, '')::text as producto_descripcion,
    COALESCE(p.titulo, '')::text as producto_titulo,
    COALESCE(p.cve_prod_integracion, '')::text as cve_prod
FROM public.productos_recompensa_variantes v
JOIN public.productos_recompensa_padre p ON v.padre_id = p.id
WHERE v.deleted_at IS NULL AND p.deleted_at IS NULL
ORDER BY p.descripcion, v.sku;
