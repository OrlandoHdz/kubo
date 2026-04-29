CREATE TABLE solicitud_registro_nuevo_cliente (
    id SERIAL PRIMARY KEY,
    nombre_comercial TEXT NOT NULL,
    razon_social TEXT NOT NULL,
    rfc VARCHAR(13) UNIQUE NOT NULL,
    tipo_contribuyente VARCHAR(20) NOT NULL CHECK (tipo_contribuyente IN ('Persona Moral', 'Persona Fisica')),
    solicitud_estado TEXT NOT NULL DEFAULT 'Pendiente',
    observacion TEXT,
    comentarios TEXT,
    constancia_sat_url TEXT, -- Guardará la ruta o URL del PDF en el servidor/S3

    -- Dirección Fiscal
    calle TEXT NOT NULL,
    numero TEXT NOT NULL,
    colonia TEXT NOT NULL,
    ciudad TEXT NOT NULL,
    estado TEXT NOT NULL,
    cp VARCHAR(5) NOT NULL,
    
    -- Información de contacto
    nombre_contacto TEXT NOT NULL,
    puesto_contacto TEXT NOT NULL,
    correo_contacto TEXT NOT NULL,
    telefono_contacto TEXT NOT NULL,

    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP, -- NULL si no ha sido borrado
    
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);

