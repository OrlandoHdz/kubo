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
    guia VARCHAR(100) DEFAULT '',
    notas_admin VARCHAR(250) DEFAULT '',
    has_backorder BOOLEAN NOT NULL DEFAULT FALSE,
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
    shipped_quantity INTEGER NOT NULL DEFAULT 0,
    backorder_quantity INTEGER NOT NULL DEFAULT 0,

    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP, -- NULL si no ha sido borrado
    
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);

-- Historial de cambios/modificaciones en pedidos (embarques parciales, backorders, etc.)
CREATE TABLE order_modifications (
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

-- Backorders independientes [cite: 123]
CREATE TABLE backorders (
    id SERIAL PRIMARY KEY,
    folio VARCHAR(20) UNIQUE NOT NULL,
    cliente_id INTEGER REFERENCES clientes(id),
    usuario_id INTEGER REFERENCES usuarios(id),
    estado_backorder VARCHAR(20) NOT NULL DEFAULT 'Pendiente', -- Pendiente, Stock Disponible, Convertido, Cancelado
    metodo_pago TEXT NOT NULL, -- 'Tarjeta' o 'Crédito'
    subtotal DECIMAL(12, 2) NOT NULL,
    iva DECIMAL(12, 2) NOT NULL,
    total_orden DECIMAL(12, 2) NOT NULL,
    guia_backorder VARCHAR(100) DEFAULT '',
    notas_admin_backorder VARCHAR(250) DEFAULT '',
    pedido_origen_id INTEGER REFERENCES pedidos(id), -- Audit: qué pedido nació de este backorder (al convertir)
    fecha_backorder TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);

CREATE TABLE backorder_detalles (
    id SERIAL PRIMARY KEY,
    backorder_id INTEGER REFERENCES backorders(id) ON DELETE CASCADE,
    variante_id INTEGER REFERENCES productos_variantes(id),
    cantidad INTEGER NOT NULL,
    precio_unitario_aplicado DECIMAL(12, 2) NOT NULL,
    disponible BOOLEAN NOT NULL DEFAULT FALSE,

    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);