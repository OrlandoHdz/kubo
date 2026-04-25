CREATE TABLE clientes (
    id SERIAL PRIMARY KEY,
    nombre_comercial TEXT NOT NULL,
    razon_social TEXT NOT NULL,
    rfc VARCHAR(13) UNIQUE NOT NULL,
    estado TEXT NOT NULL DEFAULT 'Pendiente', 

    -- Configuración Financiera
    monto_minimo_compra DECIMAL(12, 2) NOT NULL DEFAULT 0.00, 
    linea_credito_total DECIMAL(12, 2) NOT NULL DEFAULT 0.00, 
    linea_credito_utilizada DECIMAL(12, 2) NOT NULL DEFAULT 0.00, 
    dias_credito INTEGER NOT NULL DEFAULT 30, 
    permitir_pago_credito BOOLEAN NOT NULL DEFAULT FALSE, 
    metodo_pago_preferente TEXT NOT NULL DEFAULT 'Tarjeta', 
    
    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP, -- NULL si no ha sido borrado
    
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);

CREATE TABLE usuarios (
    id SERIAL PRIMARY KEY,
    cliente_id INTEGER REFERENCES clientes(id) ON DELETE CASCADE,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,

    -- Roles posibles: 
    -- Internos: 'Admin', 'Vendedor', 'Comprador_Interno'
    -- Externos: 'Cliente_Admin', 'Cliente_Comprador'
    rol TEXT NOT NULL,
    is_active BOOLEAN DEFAULT FALSE,

    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP, -- NULL si no ha sido borrado   
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);

-- Ahora vinculamos usuarios con clientes (llave foránea circular)
--ALTER TABLE usuarios ADD CONSTRAINT fk_usuario_cliente 
--FOREIGN KEY (cliente_id) REFERENCES clientes(id) ON DELETE CASCADE;
