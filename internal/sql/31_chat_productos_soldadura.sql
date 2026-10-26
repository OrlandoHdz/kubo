-- Migración 31: atributos técnicos de soldadura y tabla de búsqueda full-text.
--
-- Decisión de diseño: el tsvector NO se agrega a productos_padre ni a
-- productos_variantes para no romper las consultas existentes que usan
-- SELECT * / RETURNING *. Se mantiene en productos_busqueda.

ALTER TABLE productos_variantes
    ADD COLUMN IF NOT EXISTS proceso TEXT,
    ADD COLUMN IF NOT EXISTS atributos JSONB NOT NULL DEFAULT '{}'::jsonb;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_productos_variantes_proceso'
          AND conrelid = 'productos_variantes'::regclass
    ) THEN
        ALTER TABLE productos_variantes
            ADD CONSTRAINT chk_productos_variantes_proceso
            CHECK (proceso IS NULL OR proceso IN ('MIG', 'TIG', 'MMA', 'FCAW', 'SAW'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_productos_variantes_atributos_gin
    ON productos_variantes USING gin (atributos);

CREATE TABLE IF NOT EXISTS productos_busqueda (
    variante_id INTEGER PRIMARY KEY
        REFERENCES productos_variantes(id) ON DELETE CASCADE,
    nombre TEXT NOT NULL DEFAULT '',
    descripcion TEXT NOT NULL DEFAULT '',
    sku TEXT NOT NULL DEFAULT '',
    atributos_texto TEXT NOT NULL DEFAULT '',
    proceso TEXT,
    documento TSVECTOR GENERATED ALWAYS AS (
        to_tsvector(
            'spanish'::regconfig,
            f_unaccent(
                coalesce(nombre, '') || ' ' ||
                coalesce(descripcion, '') || ' ' ||
                coalesce(sku, '') || ' ' ||
                coalesce(atributos_texto, '') || ' ' ||
                coalesce(proceso, '')
            )
        )
    ) STORED
);

COMMENT ON TABLE productos_busqueda
    IS 'Índice de búsqueda desnormalizado para el chat; se mantiene por triggers.';

-- Recalcula/elimina la fila de búsqueda de una variante.
CREATE OR REPLACE FUNCTION refrescar_producto_busqueda(p_variante_id INTEGER)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_existe BOOLEAN;
BEGIN
    SELECT EXISTS (
        SELECT 1
        FROM productos_variantes v
        JOIN productos_padre p ON p.id = v.padre_id
        WHERE v.id = p_variante_id
          AND v.deleted_at IS NULL
          AND p.deleted_at IS NULL
    ) INTO v_existe;

    IF NOT v_existe THEN
        DELETE FROM productos_busqueda WHERE variante_id = p_variante_id;
        RETURN;
    END IF;

    INSERT INTO productos_busqueda (
        variante_id, nombre, descripcion, sku, atributos_texto, proceso
    )
    SELECT
        v.id,
        coalesce(nullif(p.titulo, ''), nullif(p.descripcion, ''), pi.nom_prod, v.sku, ''),
        coalesce(nullif(p.descripcion_extendida, ''), p.descripcion, ''),
        coalesce(v.sku, ''),
        concat_ws(
            ' ',
            v.atributos::text,
            v.marca,
            v.categoria,
            v.subgrupo,
            v.modelo,
            v.tipo,
            v.medida,
            v.unidad_medida,
            v.especificaciones
        ),
        v.proceso
    FROM productos_variantes v
    JOIN productos_padre p ON p.id = v.padre_id
    LEFT JOIN productos_integracion pi
        ON pi.cve_prod = p.cve_prod_integracion
    WHERE v.id = p_variante_id
    ON CONFLICT (variante_id) DO UPDATE SET
        nombre = EXCLUDED.nombre,
        descripcion = EXCLUDED.descripcion,
        sku = EXCLUDED.sku,
        atributos_texto = EXCLUDED.atributos_texto,
        proceso = EXCLUDED.proceso;
END;
$$;

CREATE OR REPLACE FUNCTION trg_productos_variantes_busqueda()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refrescar_producto_busqueda(OLD.id);
        RETURN OLD;
    END IF;

    PERFORM refrescar_producto_busqueda(NEW.id);

    IF TG_OP = 'UPDATE' AND NEW.padre_id IS DISTINCT FROM OLD.padre_id THEN
        PERFORM refrescar_producto_busqueda(OLD.id);
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_productos_variantes_busqueda ON productos_variantes;
CREATE TRIGGER trg_productos_variantes_busqueda
AFTER INSERT OR UPDATE OF
    sku, padre_id, atributos, proceso, deleted_at,
    marca, categoria, subgrupo, modelo, tipo, medida, unidad_medida, especificaciones
ON productos_variantes
FOR EACH ROW EXECUTE FUNCTION trg_productos_variantes_busqueda();

CREATE OR REPLACE FUNCTION trg_productos_padre_busqueda()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT id FROM productos_variantes WHERE padre_id = NEW.id
    LOOP
        PERFORM refrescar_producto_busqueda(r.id);
    END LOOP;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_productos_padre_busqueda ON productos_padre;
CREATE TRIGGER trg_productos_padre_busqueda
AFTER UPDATE OF titulo, descripcion, descripcion_extendida, deleted_at, cve_prod_integracion
ON productos_padre
FOR EACH ROW EXECUTE FUNCTION trg_productos_padre_busqueda();

CREATE OR REPLACE FUNCTION trg_productos_integracion_busqueda()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT v.id
        FROM productos_variantes v
        JOIN productos_padre p ON p.id = v.padre_id
        WHERE p.cve_prod_integracion = NEW.cve_prod
    LOOP
        PERFORM refrescar_producto_busqueda(r.id);
    END LOOP;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_productos_integracion_busqueda ON productos_integracion;
CREATE TRIGGER trg_productos_integracion_busqueda
AFTER UPDATE OF nom_prod
ON productos_integracion
FOR EACH ROW EXECUTE FUNCTION trg_productos_integracion_busqueda();

-- Backfill para catálogo existente.
INSERT INTO productos_busqueda (
    variante_id, nombre, descripcion, sku, atributos_texto, proceso
)
SELECT
    v.id,
    coalesce(nullif(p.titulo, ''), nullif(p.descripcion, ''), pi.nom_prod, v.sku, ''),
    coalesce(nullif(p.descripcion_extendida, ''), p.descripcion, ''),
    coalesce(v.sku, ''),
    concat_ws(
        ' ',
        v.atributos::text,
        v.marca,
        v.categoria,
        v.subgrupo,
        v.modelo,
        v.tipo,
        v.medida,
        v.unidad_medida,
        v.especificaciones
    ),
    v.proceso
FROM productos_variantes v
JOIN productos_padre p ON p.id = v.padre_id
LEFT JOIN productos_integracion pi
    ON pi.cve_prod = p.cve_prod_integracion
WHERE v.deleted_at IS NULL
  AND p.deleted_at IS NULL
ON CONFLICT (variante_id) DO UPDATE SET
    nombre = EXCLUDED.nombre,
    descripcion = EXCLUDED.descripcion,
    sku = EXCLUDED.sku,
    atributos_texto = EXCLUDED.atributos_texto,
    proceso = EXCLUDED.proceso;

DELETE FROM productos_busqueda pb
WHERE NOT EXISTS (
    SELECT 1
    FROM productos_variantes v
    JOIN productos_padre p ON p.id = v.padre_id
    WHERE v.id = pb.variante_id
      AND v.deleted_at IS NULL
      AND p.deleted_at IS NULL
);

CREATE INDEX IF NOT EXISTS idx_productos_busqueda_documento_gin
    ON productos_busqueda USING gin (documento);

CREATE INDEX IF NOT EXISTS idx_productos_busqueda_nombre_trgm
    ON productos_busqueda USING gin (f_unaccent(lower(nombre)) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_productos_busqueda_sku_trgm
    ON productos_busqueda USING gin (sku gin_trgm_ops);

-- La vista chat_productos expone v.sku; este índice permite búsquedas
-- parciales de SKU con ILIKE.
CREATE INDEX IF NOT EXISTS idx_productos_variantes_sku_trgm
    ON productos_variantes USING gin (sku gin_trgm_ops);

-- Búsqueda exacta de SKU sin distinguir mayúsculas/minúsculas.
CREATE INDEX IF NOT EXISTS idx_productos_variantes_sku_upper
    ON productos_variantes (upper(sku));
