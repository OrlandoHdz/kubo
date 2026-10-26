-- Migración 37 (OPCIONAL): Row Level Security como defensa adicional.
--
-- Aplicar solo si las vistas quedaron propiedad de chat_view_owner (ver
-- migración 34). El dueño de las tablas (rol de la app principal) sigue
-- saltándose RLS, por lo que la aplicación existente no se ve afectada.
--
-- IMPORTANTE: si el rol de la app principal NO es dueño de pedidos y
-- pedido_detalles, RLS también le aplicará y podría dejar de ver pedidos.
-- En ese caso agrega una política para ese rol o no apliques esta migración.
--
-- El chat siempre filtra por cliente_id en SQL; esta migración es una segunda
-- barrera si en el futuro se agrega otra consulta.

ALTER TABLE pedidos ENABLE ROW LEVEL SECURITY;
ALTER TABLE pedido_detalles ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS chat_pedidos_view_owner ON pedidos;
CREATE POLICY chat_pedidos_view_owner ON pedidos
    FOR SELECT TO chat_view_owner
    USING (
        cliente_id = NULLIF(current_setting('app.customer_id', true), '')::int
    );

DROP POLICY IF EXISTS chat_pedido_detalles_view_owner ON pedido_detalles;
CREATE POLICY chat_pedido_detalles_view_owner ON pedido_detalles
    FOR SELECT TO chat_view_owner
    USING (
        EXISTS (
            SELECT 1
            FROM pedidos p
            WHERE p.id = pedido_detalles.pedido_id
              AND p.cliente_id = NULLIF(current_setting('app.customer_id', true), '')::int
        )
    );
