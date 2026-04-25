CREATE TABLE productos_padre (
    id SERIAL PRIMARY KEY,
    nombre_tecnico TEXT NOT NULL,
    descripcion TEXT,
    categoria TEXT NOT NULL,
    marca TEXT, -- [cite: 147]
    documentacion_url TEXT, -- Para PDFs y Hojas de Seguridad [cite: 149]

    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP, -- NULL si no ha sido borrado
    
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);

CREATE TABLE productos_variantes (
    id SERIAL PRIMARY KEY,
    padre_id INTEGER REFERENCES productos_padre(id) ON DELETE CASCADE,
    sku VARCHAR(50) UNIQUE NOT NULL,
    medida TEXT, -- Ej. 1/4" de diámetro x 1" de largo [cite: 191]
    precio_lista DECIMAL(12, 2) NOT NULL, -- [cite: 153]
    stock_actual INTEGER NOT NULL DEFAULT 0, -- [cite: 159]
    unidad_medida TEXT NOT NULL DEFAULT 'Pza', -- [cite: 261]
    lead_time_dias INTEGER DEFAULT 2, -- [cite: 162]

    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP, -- NULL si no ha sido borrado
    
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);
