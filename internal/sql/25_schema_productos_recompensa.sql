-- =====================================================
-- Migración 25: Tablas espejo del catálogo de productos de recompensa
-- Estructura idéntica a productos_padre y productos_variantes
-- =====================================================

-- PADRE: espejo de productos_padre
CREATE TABLE IF NOT EXISTS public.productos_recompensa_padre (
    id SERIAL PRIMARY KEY,
    -- Sin FK a productos_integracion (no hay SAI para recompensas)
    cve_prod_integracion VARCHAR(20) NULL,
    titulo VARCHAR(400),
    descripcion VARCHAR(200),
    descripcion_extendida TEXT,
    foto_url TEXT,
    foto_url2 TEXT,
    foto_url_3 TEXT,
    foto_url_4 TEXT,
    foto_url_5 TEXT,
    foto_url_6 TEXT,
    foto_url_7 TEXT,
    foto_url_8 TEXT,
    ficha_tecnica TEXT,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);

-- VARIANTE: espejo de productos_variantes
-- ÚNICA DIFERENCIA: 3 precios → 1 solo precio_puntos
CREATE TABLE IF NOT EXISTS public.productos_recompensa_variantes (
    id SERIAL PRIMARY KEY,
    padre_id INTEGER NOT NULL REFERENCES public.productos_recompensa_padre(id) ON DELETE CASCADE,
    sku VARCHAR(50) UNIQUE NOT NULL,
    medida VARCHAR(50),
    precio_puntos INTEGER NOT NULL DEFAULT 0,
    stock_actual INTEGER NOT NULL DEFAULT 0,
    unidad_medida VARCHAR(50) NOT NULL DEFAULT 'Pza',
    lead_time_dias INTEGER DEFAULT 2,
    especificaciones VARCHAR(50),
    multiplos INTEGER DEFAULT 1,
    categoria VARCHAR(50),
    subgrupo VARCHAR(50),
    modelo VARCHAR(50),
    tipo VARCHAR(50),
    marca VARCHAR(50),
    permitir_backorder BOOLEAN DEFAULT TRUE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    created_by INTEGER REFERENCES usuarios(id),
    updated_by INTEGER REFERENCES usuarios(id),
    deleted_by INTEGER REFERENCES usuarios(id)
);
