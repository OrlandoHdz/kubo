-- Migración 26: Permisos para módulo de Niveles y Catálogo de Recompensa
INSERT INTO permisos (clave, nombre, grupo, orden) VALUES
    ('parametros-niveles', 'Parámetros Niveles', 'General', 16),
    ('productos-recompensa', 'Productos de Recompensa', 'General', 17)
ON CONFLICT (clave) DO NOTHING;
