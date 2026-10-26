-- Migración 33: vistas de solo lectura para el rol chat_ro.
-- Exponen únicamente las columnas necesarias y ocultan datos personales,
-- pagos, costos internos y notas administrativas.

CREATE OR REPLACE VIEW chat_productos AS
SELECT
    v.id,
    p.id AS padre_id,
    v.sku,
    coalesce(nullif(p.titulo, ''), nullif(p.descripcion, ''), pi.nom_prod, v.sku, '')::text AS nombre,
    coalesce(nullif(p.descripcion_extendida, ''), p.descripcion, '')::text AS descripcion,
    coalesce(p.descripcion, '')::text AS descripcion_corta,
    coalesce(p.ficha_tecnica, '')::text AS ficha_tecnica,
    coalesce(p.foto_url, '')::text AS imagen_url,
    coalesce(v.precio_lista, 0)::float8 AS precio_lista,
    coalesce(v.precio_distribuidor, 0)::float8 AS precio_distribuidor,
    coalesce(v.precio_publico, 0)::float8 AS precio_publico,
    coalesce(v.proceso, '')::text AS proceso,
    v.atributos AS atributos,
    coalesce(v.especificaciones, '')::text AS especificaciones,
    coalesce(v.medida, '')::text AS medida,
    coalesce(v.unidad_medida, '')::text AS unidad_medida,
    coalesce(v.marca, '')::text AS marca,
    coalesce(v.categoria, '')::text AS categoria,
    coalesce(v.subgrupo, '')::text AS subgrupo,
    coalesce(v.modelo, '')::text AS modelo,
    coalesce(v.tipo, '')::text AS tipo,
    coalesce(v.multiplos, 1)::int AS multiplos,
    coalesce(v.lead_time_dias, 0)::int AS lead_time_dias,
    coalesce(v.permitir_backorder, true) AS permitir_backorder,
    coalesce(stock.existencia_total, v.stock_actual, 0)::float8 AS stock,
    (coalesce(stock.existencia_total, v.stock_actual, 0) > 0) AS en_stock,
    pb.documento AS documento
FROM productos_variantes v
JOIN productos_padre p ON p.id = v.padre_id
LEFT JOIN productos_integracion pi
    ON pi.cve_prod = p.cve_prod_integracion
LEFT JOIN productos_busqueda pb
    ON pb.variante_id = v.id
LEFT JOIN LATERAL (
    SELECT sum(e.existencia) AS existencia_total
    FROM existencias_integracion e
    WHERE e.cve_prod = p.cve_prod_integracion
) stock ON true
WHERE v.deleted_at IS NULL
  AND p.deleted_at IS NULL;

COMMENT ON VIEW chat_productos
    IS 'Catálogo de solo lectura para el chat; no expone costos ni datos internos.';

CREATE OR REPLACE VIEW chat_pedidos AS
SELECT
    p.id,
    p.folio,
    p.cliente_id,
    p.estado,
    coalesce(to_char(p.fecha_pedido, 'YYYY-MM-DD"T"HH24:MI:SS'), '')::text AS fecha_pedido,
    coalesce(p.total_orden, 0)::float8 AS total_orden,
    coalesce(p.guia, '')::text AS guia,
    coalesce(p.paqueteria, '')::text AS paqueteria,
    coalesce(to_char(p.fecha_entrega_estimada, 'YYYY-MM-DD'), '')::text AS fecha_entrega_estimada,
    coalesce(p.has_backorder, false) AS has_backorder
FROM pedidos p
WHERE p.deleted_at IS NULL;

COMMENT ON VIEW chat_pedidos
    IS 'Pedidos visibles para el chat; el filtro por cliente se aplica en cada consulta.';

CREATE OR REPLACE VIEW chat_pedido_items AS
SELECT
    d.id,
    d.pedido_id,
    p.folio AS pedido_folio,
    p.cliente_id,
    coalesce(v.sku, vr.sku, '')::text AS sku,
    coalesce(
        nullif(pp.titulo, ''),
        nullif(pp.descripcion, ''),
        nullif(pr.titulo, ''),
        nullif(pr.descripcion, ''),
        vr.sku,
        v.sku,
        ''
    )::text AS nombre,
    d.cantidad,
    coalesce(d.precio_unitario_aplicado, 0)::float8 AS precio_unitario_aplicado,
    coalesce(d.shipped_quantity, 0)::int AS shipped_quantity,
    coalesce(d.backorder_quantity, 0)::int AS backorder_quantity
FROM pedido_detalles d
JOIN pedidos p ON p.id = d.pedido_id
LEFT JOIN productos_variantes v ON v.id = d.variante_id
LEFT JOIN productos_padre pp ON pp.id = v.padre_id
LEFT JOIN productos_recompensa_variantes vr ON vr.id = d.variante_id
LEFT JOIN productos_recompensa_padre pr ON pr.id = vr.padre_id
WHERE d.deleted_at IS NULL
  AND p.deleted_at IS NULL;

COMMENT ON VIEW chat_pedido_items
    IS 'Partidas de pedido para el chat; no expone costos ni auditoría interna.';

CREATE OR REPLACE VIEW chat_usuarios_clientes AS
SELECT
    u.id AS usuario_id,
    u.cliente_id,
    coalesce(c.tiene_precio_distribuidor, false) AS tiene_precio_distribuidor
FROM usuarios u
LEFT JOIN clientes c ON c.id = u.cliente_id
WHERE u.deleted_at IS NULL
  AND coalesce(u.is_active, true) = true
  AND (c.deleted_at IS NULL OR u.cliente_id IS NULL);

COMMENT ON VIEW chat_usuarios_clientes
    IS 'Mapeo mínimo usuario -> cliente para resolver la sesión sin exponer PII.';
