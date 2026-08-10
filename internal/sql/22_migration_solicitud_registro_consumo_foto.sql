-- Migration: Consumo o Distribuidor + Foto del Negocio en Solicitudes de Registro

-- 1. Agregar campo de tipo de consumo a las solicitudes de registro
ALTER TABLE solicitud_registro_nuevo_cliente
    ADD COLUMN IF NOT EXISTS consumo_o_distribuidor TEXT NOT NULL DEFAULT 'Consumo propio'
        CHECK (consumo_o_distribuidor IN ('Consumo propio', 'Distribuidor', 'Ambos')),
    ADD COLUMN IF NOT EXISTS foto_negocio_url TEXT;

-- 2. Agregar bandera de precio distribuidor a clientes.
--    Solo la opción 'Distribuidor' activa el beneficio en solicitudes nuevas;
--    los clientes existentes conservan el beneficio que ya tenían.
ALTER TABLE clientes
    ADD COLUMN IF NOT EXISTS tiene_precio_distribuidor BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE clientes SET tiene_precio_distribuidor = TRUE WHERE tiene_precio_distribuidor = FALSE;