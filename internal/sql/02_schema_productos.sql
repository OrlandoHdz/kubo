CREATE TABLE productos_padre (
    id SERIAL PRIMARY KEY,
    cve_prod_integracion VARCHAR(20) REFERENCES productos_integracion(cve_prod),
    titulo VARCHAR(400),
    descripcion VARCHAR(200),
    foto_url TEXT,
    ficha_tecnica TEXT, -- Para PDFs y Hojas de Seguridad [cite: 149]
    descripcion_extendida TEXT,
    foto_url2 TEXT,
    foto_url_3 TEXT,
    foto_url_4 TEXT,
    foto_url_5 TEXT,
    foto_url_6 TEXT,
    foto_url_7 TEXT,
    foto_url_8 TEXT,

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
    medida VARCHAR(50), -- Ej. 1/4" de diámetro x 1" de largo [cite: 191]
    precio_distribuidor DECIMAL(12, 2) NOT NULL,
    precio_lista DECIMAL(12, 2) NOT NULL, -- [cite: 153]
    precio_publico DECIMAL(12, 2) NOT NULL,
    stock_actual INTEGER NOT NULL DEFAULT 0, -- [cite: 159]
    unidad_medida VARCHAR(50) NOT NULL DEFAULT 'Pza', -- [cite: 261]
    lead_time_dias INTEGER DEFAULT 2, -- [cite: 162]
    especificaciones VARCHAR(50),
    multiplos INTEGER DEFAULT 1,
    categoria VARCHAR(50),
    subgrupo VARCHAR(50),
    modelo VARCHAR(50),
    tipo VARCHAR(50),
    marca VARCHAR(50),
    permitir_backorder BOOLEAN DEFAULT TRUE,
    
    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP, -- NULL si no ha sido borrado
    
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);



