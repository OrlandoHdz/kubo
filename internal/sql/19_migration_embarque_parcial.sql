-- Migration: Embarque Parcial, Backorders y Auditoría de Cambios en Pedidos

-- 1. Agregar columnas a pedido_detalles para control de envíos
ALTER TABLE pedido_detalles
    ADD COLUMN IF NOT EXISTS shipped_quantity INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS backorder_quantity INTEGER NOT NULL DEFAULT 0;

-- 2. Agregar bandera de backorder a pedidos
ALTER TABLE pedidos
    ADD COLUMN IF NOT EXISTS has_backorder BOOLEAN NOT NULL DEFAULT FALSE;

-- 3. Crear tabla de auditoría de modificaciones
CREATE TABLE IF NOT EXISTS order_modifications (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES pedidos(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES usuarios(id),
    item_id INTEGER REFERENCES pedido_detalles(id) ON DELETE CASCADE,
    original_quantity INTEGER NOT NULL,
    shipped_quantity INTEGER NOT NULL,
    backorder_quantity INTEGER NOT NULL DEFAULT 0,
    notes TEXT DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
