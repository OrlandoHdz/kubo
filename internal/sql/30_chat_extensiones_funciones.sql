-- Migración 30: extensiones y funciones para búsqueda tolerante a acentos/errores.
-- No ejecutar automáticamente: aplicar en desarrollo después de revisión.

CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- unaccent() es STABLE, no IMMUTABLE. Este wrapper permite usarlo en
-- columnas generadas e índices de expresión.
CREATE OR REPLACE FUNCTION f_unaccent(text)
RETURNS text
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
STRICT
AS $$
    SELECT public.unaccent('public.unaccent'::regdictionary, $1)
$$;

COMMENT ON FUNCTION f_unaccent(text)
    IS 'Wrapper IMMUTABLE de unaccent para índices y columnas generadas.';
