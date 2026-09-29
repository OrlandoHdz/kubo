-- =====================================================
-- Consultas sqlc para el módulo de recompensas (niveles, parámetros, transacciones)
-- =====================================================

-- ===== CLIENTES / NIVELES =====

-- name: UpdateClienteNivelManual :one
UPDATE public.clientes
SET
    nivel_recompensa_id = $2,
    nivel_manual_fijado = true,
    fecha_actualizacion_nivel = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateClienteSaldoPuntos :exec
UPDATE public.clientes
SET
    saldo_puntos = saldo_puntos + $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetClienteConNivel :one
SELECT
    c.*,
    nr.nombre AS nivel_nombre,
    nr.porcentaje_descuento AS nivel_descuento,
    nr.imagen_insignia_url AS nivel_insignia_url,
    nr.numero_nivel AS nivel_numero
FROM public.clientes c
LEFT JOIN public.niveles_recompensa nr ON c.nivel_recompensa_id = nr.id
WHERE c.id = $1 AND c.deleted_at IS NULL
LIMIT 1;

-- name: GetClienteSaldoPuntos :one
SELECT id, saldo_puntos FROM public.clientes
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListarClientesConNivel :many
SELECT
    c.id,
    c.nombre_comercial,
    c.rfc,
    c.saldo_puntos,
    c.nivel_manual_fijado,
    c.fecha_actualizacion_nivel,
    nr.nombre AS nivel_nombre,
    nr.porcentaje_descuento AS nivel_descuento,
    nr.numero_nivel AS nivel_numero
FROM public.clientes c
LEFT JOIN public.niveles_recompensa nr ON c.nivel_recompensa_id = nr.id
WHERE c.deleted_at IS NULL
ORDER BY c.nombre_comercial;

-- name: GetClientesConSaldoMayorCero :many
SELECT id, saldo_puntos FROM public.clientes
WHERE saldo_puntos > 0 AND deleted_at IS NULL;

-- name: UpdateClienteNivelAutomatico :one
UPDATE public.clientes
SET
    nivel_recompensa_id = $2,
    fecha_actualizacion_nivel = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL AND nivel_manual_fijado = false
RETURNING id;

-- ===== PARÁMETROS =====

-- name: GetParametrosRecompensa :one
SELECT * FROM public.parametros_recompensa
ORDER BY id DESC LIMIT 1;

-- name: UpdateParametrosRecompensa :one
UPDATE public.parametros_recompensa
SET
    monto_compra_por_punto = $1,
    valor_punto_pesos = $2,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $4
RETURNING *;

-- ===== NIVELES DE RECOMPENSA =====

-- name: ListNivelesRecompensa :many
SELECT * FROM public.niveles_recompensa
WHERE deleted_at IS NULL
ORDER BY numero_nivel ASC;

-- name: CreateNivelRecompensa :one
INSERT INTO public.niveles_recompensa (
    nombre,
    numero_nivel,
    promedio_compra_trimestral,
    porcentaje_descuento,
    imagen_insignia_url,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateNivelRecompensa :one
UPDATE public.niveles_recompensa
SET
    nombre = $2,
    numero_nivel = $3,
    promedio_compra_trimestral = $4,
    porcentaje_descuento = $5,
    imagen_insignia_url = $6,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $7
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteNivelRecompensa :exec
UPDATE public.niveles_recompensa
SET
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1;

-- name: GetNivelRecompensa :one
SELECT * FROM public.niveles_recompensa
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetNivelRecompensaByNumero :one
SELECT * FROM public.niveles_recompensa
WHERE numero_nivel = $1 AND deleted_at IS NULL LIMIT 1;

-- ===== TRANSACCIONES DE PUNTOS =====

-- name: CreateTransaccionPuntos :one
INSERT INTO public.transacciones_puntos (
    cliente_id,
    tipo,
    puntos,
    pedido_id,
    created_by
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: ListTransaccionesByCliente :many
SELECT * FROM public.transacciones_puntos
WHERE cliente_id = $1
ORDER BY created_at DESC;

-- name: GetTotalCompras90Dias :one
SELECT
    COALESCE(SUM(p.total_orden), 0)::numeric(12,2) as total_90dias
FROM public.pedidos p
WHERE p.cliente_id = $1
  AND p.deleted_at IS NULL
  AND p.estado NOT IN ('Cancelado')
  AND p.created_at >= CURRENT_TIMESTAMP - INTERVAL '90 days';
