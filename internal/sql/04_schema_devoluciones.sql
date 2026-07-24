CREATE TABLE devoluciones_garantias (
    id SERIAL PRIMARY KEY,
    folio VARCHAR(20) UNIQUE NOT NULL,
    cliente_id INTEGER REFERENCES clientes(id),
    pedido_folio VARCHAR(20) NOT NULL,
    tipo VARCHAR(20) NOT NULL CHECK (tipo IN ('Devolucion', 'Garantia')),
    numeros_parte TEXT NOT NULL DEFAULT '',
    nota_cliente TEXT NOT NULL DEFAULT '',
    evidencias TEXT NOT NULL DEFAULT '',
    estatus VARCHAR(20) NOT NULL DEFAULT 'Pendiente' CHECK (estatus IN ('Pendiente', 'Aprobada', 'Rechazada', 'Cancelada')),
    nota_administrador TEXT NOT NULL DEFAULT '',

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);
