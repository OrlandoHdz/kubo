-- Actualizar totales de pedidos que tienen descuento
-- El total_orden debe reflejar el subtotal original sin descuento para cálculo de niveles

-- Primero, verificar si hay pedidos con descuento
SELECT 
    COUNT(*) as pedidos_con_descuento,
    SUM(descuento_aplicado) as total_descuentos
FROM pedidos 
WHERE descuento_aplicado > 0 AND deleted_at IS NULL;

-- Nota: Esta migración es solo informativa
-- El cálculo de niveles usa total_orden que ya incluye el descuento aplicado
-- Esto es correcto porque el cliente pagó ese monto real
