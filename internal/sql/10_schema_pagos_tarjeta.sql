CREATE TABLE transacciones_banregio (
    id SERIAL PRIMARY KEY,
    bnrg_monto_trans NUMERIC(12, 2),   -- Soporta decimales para el monto (ej. 100.00)
    bnrg_id_afiliacion VARCHAR(20),    -- Estructura de texto por si varía el largo o incluye letras
    bnrg_fecha_local DATE,             -- Tipo fecha (YYYY-MM-DD)
    bnrg_codigo_aut VARCHAR(10),       -- Código de autorización (se deja texto por los ceros a la izquierda)
    bnrg_folio VARCHAR(30),            -- Folio de la transacción
    bnrg_texto VARCHAR(50),            -- Descripción del estatus (ej. "Aprobado")
    bnrg_referencia VARCHAR(30),       -- Número de referencia
    bnrg_id_mediop VARCHAR(20),        -- ID Medio de pago (ej. "P1927H19")
    bnrg_codigo_proc VARCHAR(10),      -- Código de proceso
    bnrg_hora_local TIME,              -- Tipo hora (HH:MM:SS)
    bnrg_codigo_emisor VARCHAR(10),    -- Código de emisor (mantiene el "00")
    pedido_id INTEGER REFERENCES pedidos(id),


    -- Campos de Auditoría
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP, -- NULL si no ha sido borrado
    
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);