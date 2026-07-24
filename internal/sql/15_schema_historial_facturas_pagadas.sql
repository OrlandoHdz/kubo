CREATE TABLE historial_facturas_pagadas (
    id SERIAL PRIMARY KEY,
    factura_id INTEGER NOT NULL REFERENCES facturas_integracion(id),
    cliente_id INTEGER NOT NULL REFERENCES clientes(id),
    no_factura VARCHAR(20) NOT NULL,
    monto_pagado DECIMAL(12,2) NOT NULL,
    metodo_pago VARCHAR(30) NOT NULL DEFAULT 'Tarjeta de Credito',
    tarjeta_terminacion VARCHAR(4) NOT NULL DEFAULT '',
    respuesta_simulada TEXT NOT NULL DEFAULT '',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by INTEGER REFERENCES usuarios(id)
);
