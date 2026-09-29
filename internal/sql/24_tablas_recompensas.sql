-- =====================================================
-- Migración 24: Tablas del módulo de recompensas (niveles, parámetros, transacciones)
-- Todas las tablas son nuevas (no modifican existentes).
-- =====================================================

-- Niveles de recompensa (Bronce, Plata, Oro, etc.)
CREATE TABLE IF NOT EXISTS public.niveles_recompensa (
    id serial PRIMARY KEY,
    nombre text NOT NULL,
    numero_nivel integer NOT NULL DEFAULT 0,
    promedio_compra_trimestral numeric(12,2) NOT NULL DEFAULT 0.00,
    porcentaje_descuento numeric(5,2) NOT NULL DEFAULT 0.00,
    imagen_insignia_url text NULL,

    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    deleted_at timestamp without time zone,
    created_by integer REFERENCES public.usuarios(id),
    updated_by integer REFERENCES public.usuarios(id),
    deleted_by integer REFERENCES public.usuarios(id)
);

-- Parámetros globales del sistema de recompensas
CREATE TABLE IF NOT EXISTS public.parametros_recompensa (
    id serial PRIMARY KEY,
    monto_compra_por_punto numeric(12,2) NOT NULL DEFAULT 25.00,
    valor_punto_pesos numeric(12,2) NOT NULL DEFAULT 0.40,

    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_by integer REFERENCES public.usuarios(id)
);

-- Historial de transacciones de puntos por cliente
CREATE TABLE IF NOT EXISTS public.transacciones_puntos (
    id serial PRIMARY KEY,
    cliente_id integer NOT NULL REFERENCES public.clientes(id),
    tipo text NOT NULL, -- 'ACUMULADO', 'CANJEADO', 'REINICIO_ANUAL'
    puntos integer NOT NULL,
    pedido_id integer NULL,

    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    created_by integer REFERENCES public.usuarios(id)
);

-- Insertar parámetros por defecto si no existen
INSERT INTO public.parametros_recompensa (monto_compra_por_punto, valor_punto_pesos)
SELECT 25.00, 0.40
WHERE NOT EXISTS (SELECT 1 FROM public.parametros_recompensa);

-- Insertar nivel base (Nivel 0) si no existe
INSERT INTO public.niveles_recompensa (nombre, numero_nivel, promedio_compra_trimestral, porcentaje_descuento)
SELECT 'Nivel 0', 0, 0.00, 0.00
WHERE NOT EXISTS (SELECT 1 FROM public.niveles_recompensa WHERE numero_nivel = 0);
