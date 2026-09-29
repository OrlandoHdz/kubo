-- =====================================================
-- Migración 23: Columnas aditivas de recompensas en clientes
-- Todos los campos son NULL o con DEFAULT para no romper registros existentes.
-- =====================================================

ALTER TABLE public.clientes
  ADD COLUMN IF NOT EXISTS nivel_recompensa_id integer NULL,
  ADD COLUMN IF NOT EXISTS saldo_puntos integer NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS nivel_manual_fijado boolean NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS fecha_actualizacion_nivel timestamp without time zone NULL;
