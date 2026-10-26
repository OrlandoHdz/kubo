-- Migración 32: datos de rastreo y entrega estimada para el chat de pedidos.

ALTER TABLE pedidos
    ADD COLUMN IF NOT EXISTS paqueteria TEXT,
    ADD COLUMN IF NOT EXISTS fecha_entrega_estimada TIMESTAMP;

COMMENT ON COLUMN pedidos.paqueteria
    IS 'Paquetería utilizada para el envío; editable por administración.';
COMMENT ON COLUMN pedidos.fecha_entrega_estimada
    IS 'Fecha estimada de entrega mostrada al cliente en el chat.';

-- Índices para las consultas del chat de pedidos.
CREATE INDEX IF NOT EXISTS idx_pedidos_cliente_fecha
    ON pedidos (cliente_id, fecha_pedido DESC);

CREATE INDEX IF NOT EXISTS idx_pedido_detalles_pedido
    ON pedido_detalles (pedido_id);
