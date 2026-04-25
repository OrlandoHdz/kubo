-- Encabezado de la Orden [cite: 258]
CREATE TABLE pedidos (
    id SERIAL PRIMARY KEY,
    folio VARCHAR(20) UNIQUE NOT NULL,
    cliente_id INTEGER REFERENCES clientes(id),
    usuario_id INTEGER REFERENCES usuarios(id),
    estado TEXT NOT NULL DEFAULT 'Pendiente', 
    metodo_pago TEXT NOT NULL, -- 'Tarjeta' o 'Crédito' 
    subtotal DECIMAL(12, 2) NOT NULL,
    iva DECIMAL(12, 2) NOT NULL,
    total_orden DECIMAL(12, 2) NOT NULL, 
    es_backorder BOOLEAN DEFAULT FALSE, 
    fecha_pedido TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP, -- NULL si no ha sido borrado
    
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)

);

-- Detalle de partidas [cite: 259, 261]
CREATE TABLE pedido_detalles (
    id SERIAL PRIMARY KEY,
    pedido_id INTEGER REFERENCES pedidos(id) ON DELETE CASCADE,
    variante_id INTEGER REFERENCES productos_variantes(id),
    cantidad INTEGER NOT NULL,
    precio_unitario_aplicado DECIMAL(12, 2) NOT NULL, 

    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP, -- NULL si no ha sido borrado
    
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);