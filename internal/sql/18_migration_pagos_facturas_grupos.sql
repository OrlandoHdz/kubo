CREATE TABLE IF NOT EXISTS pagos_facturas (
    id SERIAL PRIMARY KEY,
    cliente_id INTEGER NOT NULL REFERENCES clientes(id),
    metodo_pago VARCHAR(30) NOT NULL DEFAULT 'Tarjeta de Credito',
    tarjeta_terminacion VARCHAR(4) NOT NULL DEFAULT '',
    monto_total DECIMAL(12,2) NOT NULL,
    respuesta_simulada TEXT NOT NULL DEFAULT '',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER REFERENCES usuarios(id)
);

CREATE TABLE IF NOT EXISTS pagos_facturas_detalles (
    id SERIAL PRIMARY KEY,
    pago_id INTEGER NOT NULL REFERENCES pagos_facturas(id),
    factura_id INTEGER NOT NULL REFERENCES facturas_integracion(id),
    no_factura VARCHAR(50) NOT NULL,
    monto_pagado DECIMAL(12,2) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pagos_facturas_created_at ON pagos_facturas(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pagos_facturas_detalles_pago_id ON pagos_facturas_detalles(pago_id);
