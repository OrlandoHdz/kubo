-- name: CrearCliente :one
-- Registra un nuevo cliente e identifica al responsable (Pág. 10)
INSERT INTO clientes (
    nombre_comercial, 
    razon_social, 
    rfc, 
    estado, 
    monto_minimo_compra, 
    linea_credito_total, 
    linea_credito_utilizada,
    dias_credito, 
    permitir_pago_credito, 
    metodo_pago_preferente,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: ActualizarCliente :one
-- Modifica datos generales y financieros registrando el auditor (Pág. 16)
UPDATE clientes
SET 
    nombre_comercial = $2,
    razon_social = $3,
    estado = $4,
    monto_minimo_compra = $5,
    linea_credito_total = $6,
    linea_credito_utilizada = $7,
    dias_credito = $8,  
    permitir_pago_credito = $9,
    metodo_pago_preferente = $10,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $11
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ActualizarSaldoCredito :exec
-- Solo modifica el saldo utilizado (usado tras un pedido) (Pág. 20)
UPDATE clientes
SET 
    linea_credito_utilizada = linea_credito_utilizada + $2,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $1;

-- name: SoftDeleteCliente :exec
-- Borrado lógico para mantener integridad histórica (Pág. 21)
UPDATE clientes
SET 
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2,
    estado = 'Bloqueado'
WHERE id = $1;

-- name: GetCliente :one
-- Obtiene un cliente por su ID para validar solvencia [cite: 131]
SELECT * FROM clientes
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListarClientesActivos :many
-- Lista clientes para el panel de administración [cite: 139]
SELECT * FROM clientes
WHERE deleted_at IS NULL;