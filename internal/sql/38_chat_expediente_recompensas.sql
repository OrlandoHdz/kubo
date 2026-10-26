-- Migración 38: vistas de backorders, devoluciones y recompensas para el chat.
--
-- Todas las vistas de datos de cliente exponen cliente_id para que las
-- consultas sqlc filtren SIEMPRE por la sesión autenticada. No se exponen
-- notas administrativas, evidencias, métodos de pago ni datos personales.

CREATE OR REPLACE VIEW chat_backorders AS
SELECT
    b.id,
    b.folio,
    b.cliente_id,
    b.estado_backorder,
    coalesce(to_char(b.fecha_backorder, 'YYYY-MM-DD"T"HH24:MI:SS'), '')::text AS fecha_backorder,
    coalesce(b.total_orden, 0)::float8 AS total_orden,
    coalesce(b.guia_backorder, '')::text AS guia_backorder,
    coalesce(p.folio, '')::text AS pedido_origen_folio
FROM backorders b
LEFT JOIN pedidos p ON p.id = b.pedido_origen_id
WHERE b.deleted_at IS NULL;

CREATE OR REPLACE VIEW chat_backorder_items AS
SELECT
    d.id,
    d.backorder_id,
    b.cliente_id,
    coalesce(v.sku, '')::text AS sku,
    coalesce(nullif(p.titulo, ''), nullif(p.descripcion, ''), '')::text AS nombre,
    d.cantidad,
    coalesce(d.precio_unitario_aplicado, 0)::float8 AS precio_unitario_aplicado,
    d.disponible
FROM backorder_detalles d
JOIN backorders b ON b.id = d.backorder_id
LEFT JOIN productos_variantes v ON v.id = d.variante_id
LEFT JOIN productos_padre p ON p.id = v.padre_id
WHERE d.deleted_at IS NULL
  AND b.deleted_at IS NULL;

CREATE OR REPLACE VIEW chat_devoluciones AS
SELECT
    d.id,
    d.folio,
    d.cliente_id,
    d.pedido_folio,
    d.tipo,
    coalesce(d.numeros_parte, '')::text AS numeros_parte,
    coalesce(d.cantidades, '')::text AS cantidades,
    d.estatus,
    coalesce(to_char(d.created_at, 'YYYY-MM-DD"T"HH24:MI:SS'), '')::text AS fecha_creacion,
    coalesce(to_char(d.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS'), '')::text AS fecha_actualizacion
FROM devoluciones_garantias d
WHERE d.deleted_at IS NULL;

CREATE OR REPLACE VIEW chat_recompensas_cliente AS
SELECT
    c.id AS cliente_id,
    c.saldo_puntos,
    c.nivel_manual_fijado,
    coalesce(to_char(c.fecha_actualizacion_nivel, 'YYYY-MM-DD"T"HH24:MI:SS'), '')::text AS fecha_actualizacion_nivel,
    nr.id AS nivel_id,
    coalesce(nr.nombre, '')::text AS nivel_nombre,
    coalesce(nr.numero_nivel, 0)::int AS nivel_numero,
    coalesce(nr.promedio_compra_trimestral, 0)::float8 AS nivel_promedio_trimestral,
    coalesce(nr.porcentaje_descuento, 0)::float8 AS nivel_descuento,
    coalesce(compras.total_90dias, 0)::float8 AS compra_90dias,
    (coalesce(compras.total_90dias, 0) / 3.0)::float8 AS promedio_trimestral,
    coalesce(param.monto_compra_por_punto, 0)::float8 AS monto_compra_por_punto,
    coalesce(param.valor_punto_pesos, 0)::float8 AS valor_punto_pesos
FROM clientes c
LEFT JOIN niveles_recompensa nr
    ON nr.id = c.nivel_recompensa_id
   AND nr.deleted_at IS NULL
LEFT JOIN LATERAL (
    SELECT sum(p.total_orden) AS total_90dias
    FROM pedidos p
    WHERE p.cliente_id = c.id
      AND p.deleted_at IS NULL
      AND p.estado NOT IN ('Cancelado')
      AND p.created_at >= CURRENT_TIMESTAMP - INTERVAL '90 days'
) compras ON true
LEFT JOIN LATERAL (
    SELECT monto_compra_por_punto, valor_punto_pesos
    FROM parametros_recompensa
    ORDER BY id DESC
    LIMIT 1
) param ON true
WHERE c.deleted_at IS NULL;

CREATE OR REPLACE VIEW chat_niveles_recompensa AS
SELECT
    nr.id,
    nr.nombre,
    nr.numero_nivel,
    coalesce(nr.promedio_compra_trimestral, 0)::float8 AS promedio_compra_trimestral,
    coalesce(nr.porcentaje_descuento, 0)::float8 AS porcentaje_descuento
FROM niveles_recompensa nr
WHERE nr.deleted_at IS NULL;

CREATE OR REPLACE VIEW chat_parametros_recompensa AS
SELECT
    p.monto_compra_por_punto::float8 AS monto_compra_por_punto,
    p.valor_punto_pesos::float8 AS valor_punto_pesos
FROM parametros_recompensa p
ORDER BY p.id DESC
LIMIT 1;

CREATE OR REPLACE VIEW chat_transacciones_puntos AS
SELECT
    t.id,
    t.cliente_id,
    t.tipo,
    t.puntos,
    coalesce(to_char(t.created_at, 'YYYY-MM-DD"T"HH24:MI:SS'), '')::text AS fecha,
    coalesce(p.folio, '')::text AS pedido_folio
FROM transacciones_puntos t
LEFT JOIN pedidos p ON p.id = t.pedido_id;

-- Índices para las consultas del chat.
CREATE INDEX IF NOT EXISTS idx_backorders_cliente_fecha
    ON backorders (cliente_id, fecha_backorder DESC);

CREATE INDEX IF NOT EXISTS idx_backorder_detalles_backorder
    ON backorder_detalles (backorder_id);

CREATE INDEX IF NOT EXISTS idx_devoluciones_cliente_fecha
    ON devoluciones_garantias (cliente_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_transacciones_puntos_cliente_fecha
    ON transacciones_puntos (cliente_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_pedidos_cliente_created_at
    ON pedidos (cliente_id, created_at DESC);

-- Columnas necesarias para el dueño de las vistas. Varias ya se otorgaron en
-- la migración 34; repetirlas es idempotente.
GRANT SELECT (
    id, folio, cliente_id, estado_backorder, fecha_backorder, total_orden,
    guia_backorder, pedido_origen_id, deleted_at
) ON backorders TO chat_view_owner;

GRANT SELECT (
    id, backorder_id, variante_id, cantidad, precio_unitario_aplicado,
    disponible, deleted_at
) ON backorder_detalles TO chat_view_owner;

GRANT SELECT (
    id, folio, cliente_id, pedido_folio, tipo, numeros_parte, cantidades,
    estatus, created_at, updated_at, deleted_at
) ON devoluciones_garantias TO chat_view_owner;

GRANT SELECT (
    id, nivel_recompensa_id, saldo_puntos, nivel_manual_fijado,
    fecha_actualizacion_nivel, deleted_at
) ON clientes TO chat_view_owner;

GRANT SELECT (
    id, nombre, numero_nivel, promedio_compra_trimestral,
    porcentaje_descuento, deleted_at
) ON niveles_recompensa TO chat_view_owner;

GRANT SELECT (
    id, monto_compra_por_punto, valor_punto_pesos
) ON parametros_recompensa TO chat_view_owner;

GRANT SELECT (
    id, cliente_id, tipo, puntos, pedido_id, created_at
) ON transacciones_puntos TO chat_view_owner;

-- El promedio de 90 días de chat_recompensas_cliente usa pedidos.created_at.
GRANT SELECT (created_at) ON pedidos TO chat_view_owner;

DO $$
BEGIN
    GRANT SELECT ON
        chat_backorders,
        chat_backorder_items,
        chat_devoluciones,
        chat_recompensas_cliente,
        chat_niveles_recompensa,
        chat_parametros_recompensa,
        chat_transacciones_puntos
    TO chat_ro;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'No se pudo re-otorgar SELECT a chat_ro; verifica que ya tenga acceso a las nuevas vistas.';
END $$;

DO $$
BEGIN
    ALTER VIEW chat_backorders OWNER TO chat_view_owner;
    ALTER VIEW chat_backorder_items OWNER TO chat_view_owner;
    ALTER VIEW chat_devoluciones OWNER TO chat_view_owner;
    ALTER VIEW chat_recompensas_cliente OWNER TO chat_view_owner;
    ALTER VIEW chat_niveles_recompensa OWNER TO chat_view_owner;
    ALTER VIEW chat_parametros_recompensa OWNER TO chat_view_owner;
    ALTER VIEW chat_transacciones_puntos OWNER TO chat_view_owner;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'No se pudo transferir la propiedad de las nuevas vistas a chat_view_owner. El filtro explícito por cliente_id sigue protegiendo las consultas.';
END $$;
