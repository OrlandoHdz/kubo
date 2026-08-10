-- Tabla de permisos/módulos del menú del Panel de Control
CREATE TABLE IF NOT EXISTS permisos (
    id SERIAL PRIMARY KEY,
    clave TEXT UNIQUE NOT NULL,
    nombre TEXT NOT NULL,
    grupo TEXT NOT NULL DEFAULT 'General',
    orden INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Relación usuario <-> permisos
CREATE TABLE IF NOT EXISTS usuario_permisos (
    id SERIAL PRIMARY KEY,
    usuario_id INTEGER NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    permiso_id INTEGER NOT NULL REFERENCES permisos(id) ON DELETE CASCADE,
    activo BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by INTEGER REFERENCES usuarios(id),
    UNIQUE (usuario_id, permiso_id)
);

-- Insertar los módulos del menú (idempotente)
INSERT INTO permisos (clave, nombre, grupo, orden) VALUES
    ('dashboard', 'Dashboard', 'General', 1),
    ('solicitudes', 'Solicitudes Registro', 'General', 2),
    ('clientes', 'Clientes SAI', 'Integración SAI', 3),
    ('facturas', 'Facturas SAI', 'Integración SAI', 4),
    ('existencias', 'Existencias SAI', 'Integración SAI', 5),
    ('creditos', 'Créditos SAI', 'Integración SAI', 6),
    ('productos-sai', 'Productos SAI', 'Integración SAI', 7),
    ('usuarios', 'Usuarios', 'General', 8),
    ('productos', 'Catálogo de Productos', 'General', 9),
    ('pedidos', 'Pedidos', 'General', 10),
    ('backorders', 'Backorders', 'General', 11),
    ('auditoria-pedidos', 'Auditoría de Pedidos', 'General', 12),
    ('devoluciones', 'Devoluciones y Garantías', 'General', 13),
    ('pago-facturas', 'Pago de Facturas', 'General', 14),
    ('transacciones-banregio', 'Transacciones Banregio', 'General', 15)
ON CONFLICT (clave) DO NOTHING;
