-- Migración 29: Tabla de ajustes de inventario para productos de recompensa
CREATE TABLE IF NOT EXISTS public.ajustes_inventario_recompensa (
    id serial PRIMARY KEY,
    variante_id integer NOT NULL REFERENCES public.productos_recompensa_variantes(id),
    tipo text NOT NULL CHECK (tipo IN ('SUMAR','RESTAR')),
    cantidad integer NOT NULL CHECK (cantidad > 0),
    comentario text NOT NULL,
    stock_anterior integer NOT NULL,
    stock_nuevo integer NOT NULL,
    usuario_id integer REFERENCES public.usuarios(id),
    usuario_nombre text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ajustes_variante ON public.ajustes_inventario_recompensa(variante_id);
CREATE INDEX IF NOT EXISTS idx_ajustes_fecha ON public.ajustes_inventario_recompensa(created_at);
