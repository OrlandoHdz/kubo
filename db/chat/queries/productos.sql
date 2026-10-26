-- Consultas de catálogo para el chat. Solo lectura.
-- El precio se elige en el servidor según el contexto del cliente autenticado.
-- LIMIT 8 es fijo; el modelo nunca decide el límite de resultados.

-- name: BuscarProductos :many
WITH busqueda AS (
    SELECT websearch_to_tsquery(
        'spanish',
        f_unaccent(sqlc.arg(consulta)::text)
    ) AS tsq
)
SELECT
    cp.id,
    cp.padre_id,
    cp.sku,
    cp.nombre,
    CASE
        WHEN sqlc.arg(es_distribuidor)::boolean THEN cp.precio_distribuidor
        ELSE cp.precio_lista
    END::float8 AS precio,
    'MXN'::text AS moneda,
    cp.proceso,
    cp.atributos,
    cp.stock,
    cp.en_stock,
    cp.imagen_url,
    cp.marca,
    cp.categoria,
    cp.descripcion,
    cp.especificaciones
FROM chat_productos cp
CROSS JOIN busqueda b
WHERE cp.precio_lista > 0
  AND (
      cp.documento @@ b.tsq
      OR cp.sku ILIKE '%' || sqlc.arg(consulta)::text || '%'
      OR f_unaccent(lower(cp.nombre)) LIKE '%' || f_unaccent(lower(sqlc.arg(consulta)::text)) || '%'
      OR similarity(
          f_unaccent(lower(cp.nombre)),
          f_unaccent(lower(sqlc.arg(consulta)::text))
      ) > 0.2
  )
  AND (sqlc.narg(proceso)::text IS NULL OR cp.proceso = sqlc.narg(proceso)::text)
  AND (sqlc.narg(filtros)::jsonb IS NULL OR cp.atributos @> sqlc.narg(filtros)::jsonb)
  AND (
      sqlc.narg(amperaje_min)::float8 IS NULL
      OR NULLIF(cp.atributos ->> 'amperaje_min', '')::float8 >= sqlc.narg(amperaje_min)::float8
  )
  AND (
      sqlc.narg(amperaje_max)::float8 IS NULL
      OR NULLIF(cp.atributos ->> 'amperaje_max', '')::float8 <= sqlc.narg(amperaje_max)::float8
  )
  AND (sqlc.narg(en_stock)::boolean IS NULL OR cp.en_stock = sqlc.narg(en_stock)::boolean)
ORDER BY
    ts_rank(cp.documento, b.tsq) DESC NULLS LAST,
    similarity(
        f_unaccent(lower(cp.nombre)),
        f_unaccent(lower(sqlc.arg(consulta)::text))
    ) DESC,
    cp.nombre ASC
LIMIT 8;

-- name: ObtenerProductoPorSKU :one
SELECT
    cp.id,
    cp.padre_id,
    cp.sku,
    cp.nombre,
    cp.descripcion,
    cp.descripcion_corta,
    cp.ficha_tecnica,
    cp.imagen_url,
    CASE
        WHEN sqlc.arg(es_distribuidor)::boolean THEN cp.precio_distribuidor
        ELSE cp.precio_lista
    END::float8 AS precio,
    'MXN'::text AS moneda,
    cp.proceso,
    cp.atributos,
    cp.especificaciones,
    cp.medida,
    cp.unidad_medida,
    cp.marca,
    cp.categoria,
    cp.subgrupo,
    cp.modelo,
    cp.tipo,
    cp.multiplos,
    cp.lead_time_dias,
    cp.permitir_backorder,
    cp.stock,
    cp.en_stock
FROM chat_productos cp
WHERE upper(cp.sku) = upper(sqlc.arg(sku)::text)
  AND cp.precio_lista > 0
LIMIT 1;
