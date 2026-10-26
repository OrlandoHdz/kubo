-- Migración 34: roles de solo lectura para el chat.
--
-- Los roles se crean SIN contraseña. El administrador debe asignarla fuera del
-- repositorio, por ejemplo:
--   ALTER ROLE chat_ro WITH PASSWORD '...';
--
-- Requiere ejecutarse con un rol que pueda crear roles y ceder propiedad de
-- vistas (normalmente un superusuario o el dueño de la base).

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'chat_view_owner') THEN
        CREATE ROLE chat_view_owner NOLOGIN;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'chat_ro') THEN
        CREATE ROLE chat_ro LOGIN;
    END IF;
END $$;

GRANT USAGE ON SCHEMA public TO chat_view_owner, chat_ro;

-- Necesario para poder transferir la propiedad de las vistas a un rol
-- distinto. chat_view_owner es NOLOGIN y solo se usa como dueño de vistas.
DO $$
BEGIN
    GRANT CREATE ON SCHEMA public TO chat_view_owner;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'No se pudo otorgar CREATE en public a chat_view_owner; la transferencia de vistas puede fallar.';
END $$;

DO $$
BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO chat_ro', current_database());
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'No se pudo otorgar CONNECT a chat_ro; el administrador debe hacerlo manualmente.';
END $$;

-- El dueño de las vistas necesita leer las tablas base. Es NOLOGIN y solo se
-- usa como propietario de las vistas (security definer), por lo que chat_ro
-- nunca obtiene permisos directos sobre las tablas. Se conceden únicamente
-- las columnas que usan las vistas: nunca password_hash, pagos ni notas.
GRANT SELECT (
    id, titulo, descripcion, descripcion_extendida, ficha_tecnica, foto_url,
    cve_prod_integracion, deleted_at
) ON productos_padre TO chat_view_owner;

GRANT SELECT (
    id, padre_id, sku, medida, precio_distribuidor, precio_lista, precio_publico,
    stock_actual, unidad_medida, lead_time_dias, especificaciones, multiplos,
    categoria, subgrupo, modelo, tipo, marca, permitir_backorder, deleted_at,
    proceso, atributos
) ON productos_variantes TO chat_view_owner;

GRANT SELECT (cve_prod, nom_prod) ON productos_integracion TO chat_view_owner;

GRANT SELECT (cve_prod, existencia) ON existencias_integracion TO chat_view_owner;

GRANT SELECT (id, sku, padre_id) ON productos_recompensa_variantes TO chat_view_owner;

GRANT SELECT (id, titulo, descripcion) ON productos_recompensa_padre TO chat_view_owner;

GRANT SELECT (
    id, folio, cliente_id, estado, fecha_pedido, total_orden, guia,
    paqueteria, fecha_entrega_estimada, has_backorder, deleted_at
) ON pedidos TO chat_view_owner;

GRANT SELECT (
    id, pedido_id, variante_id, cantidad, precio_unitario_aplicado,
    shipped_quantity, backorder_quantity, deleted_at
) ON pedido_detalles TO chat_view_owner;

GRANT SELECT (variante_id, documento) ON productos_busqueda TO chat_view_owner;

GRANT SELECT (id, cliente_id, deleted_at, is_active) ON usuarios TO chat_view_owner;

GRANT SELECT (id, tiene_precio_distribuidor, deleted_at) ON clientes TO chat_view_owner;

-- Se concede antes de transferir propiedad: el rol que ejecuta la migración es
-- dueño actual de las vistas y puede otorgar SELECT. El bloque tolera una
-- re-ejecución cuando la propiedad ya fue transferida.
DO $$
BEGIN
    GRANT SELECT ON
        chat_productos,
        chat_pedidos,
        chat_pedido_items,
        chat_usuarios_clientes
    TO chat_ro;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'No se pudo re-otorgar SELECT a chat_ro; verifica que ya tenga acceso a las vistas.';
END $$;

-- Transferencia de propiedad (necesaria para que RLS opcional aplique a las
-- vistas). Si el rol actual no puede transferir, se deja un aviso y las vistas
-- siguen siendo utilizables gracias al filtro explícito por cliente_id.
DO $$
BEGIN
    ALTER VIEW chat_productos OWNER TO chat_view_owner;
    ALTER VIEW chat_pedidos OWNER TO chat_view_owner;
    ALTER VIEW chat_pedido_items OWNER TO chat_view_owner;
    ALTER VIEW chat_usuarios_clientes OWNER TO chat_view_owner;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'No se pudo transferir la propiedad de las vistas a chat_view_owner. RLS opcional no aplicará; el filtro explícito por cliente_id sigue protegiendo las consultas.';
END $$;

ALTER ROLE chat_ro SET statement_timeout = '3s';
ALTER ROLE chat_ro SET default_transaction_read_only = on;
ALTER ROLE chat_ro SET idle_in_transaction_session_timeout = '5s';
