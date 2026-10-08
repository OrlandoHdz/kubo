-- Agregar columnas de descuento por nivel de recompensa a pedidos y backorders

ALTER TABLE pedidos
    ADD COLUMN descuento_aplicado DECIMAL(12, 2) NOT NULL DEFAULT 0,
    ADD COLUMN descuento_porcentaje DECIMAL(5, 2) NOT NULL DEFAULT 0,
    ADD COLUMN puntos_otorgados INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN nivel_recompensa_id INTEGER REFERENCES niveles_recompensa(id);

ALTER TABLE backorders
    ADD COLUMN descuento_aplicado DECIMAL(12, 2) NOT NULL DEFAULT 0,
    ADD COLUMN descuento_porcentaje DECIMAL(5, 2) NOT NULL DEFAULT 0,
    ADD COLUMN puntos_otorgados INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN nivel_recompensa_id INTEGER REFERENCES niveles_recompensa(id);

-- Agregar comentario a las columnas
COMMENT ON COLUMN pedidos.descuento_aplicado IS 'Monto del descuento aplicado por nivel de recompensa';
COMMENT ON COLUMN pedidos.descuento_porcentaje IS 'Porcentaje de descuento aplicado por nivel de recompensa';
COMMENT ON COLUMN pedidos.puntos_otorgados IS 'Puntos otorgados al cliente por esta compra';
COMMENT ON COLUMN pedidos.nivel_recompensa_id IS 'Nivel de recompensa del cliente al momento de la compra';

COMMENT ON COLUMN backorders.descuento_aplicado IS 'Monto del descuento aplicado por nivel de recompensa';
COMMENT ON COLUMN backorders.descuento_porcentaje IS 'Porcentaje de descuento aplicado por nivel de recompensa';
COMMENT ON COLUMN backorders.puntos_otorgados IS 'Puntos otorgados al cliente por este backorder';
COMMENT ON COLUMN backorders.nivel_recompensa_id IS 'Nivel de recompensa del cliente al momento del backorder';
