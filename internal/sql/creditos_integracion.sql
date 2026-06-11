-- name: CreateCredito :one
INSERT INTO creditos_integracion (
    cve_dda, no_nota, tip_not, fecha, no_fac, no_cliente, no_agente, 
    no_estado, tot_imp, subtotal, desc_not, tot_nota, num, saldo, 
    tot_des, lugar, cve_factu, cve_mon, tip_cam, cve_suc, mes, aqo, usuario, trans
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
)
RETURNING *;

-- name: ListAllCreditos :many
SELECT * FROM creditos_integracion
ORDER BY fecha DESC;

-- name: GetCreditoByID :one
SELECT * FROM creditos_integracion
WHERE id = $1 LIMIT 1;

-- name: GetCreditoByFactura :one
-- Aprovecha la restricción UNIQUE (no_fac, cve_factu)
SELECT * FROM creditos_integracion
WHERE no_fac = $1 AND cve_factu = $2 LIMIT 1;

-- name: ListCreditosByCliente :many
SELECT * FROM creditos_integracion
WHERE no_cliente = $1
ORDER BY fecha DESC;

-- name: UpdateCreditoSaldo :one
UPDATE creditos_integracion
SET 
    saldo = $2,
    usuario = $3
WHERE id = $1
RETURNING *;

-- name: UpsertCredito :one
INSERT INTO creditos_integracion (
    cve_dda, no_nota, tip_not, fecha, no_fac, no_cliente, no_agente, 
    no_estado, tot_imp, subtotal, desc_not, tot_nota, num, saldo, 
    tot_des, lugar, cve_factu, cve_mon, tip_cam, cve_suc, mes, aqo, usuario, trans
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
)
ON CONFLICT (no_fac, cve_factu) DO UPDATE SET
    cve_dda = EXCLUDED.cve_dda,
    no_nota = EXCLUDED.no_nota,
    tip_not = EXCLUDED.tip_not,
    fecha = EXCLUDED.fecha,
    no_cliente = EXCLUDED.no_cliente,
    no_agente = EXCLUDED.no_agente,
    no_estado = EXCLUDED.no_estado,
    tot_imp = EXCLUDED.tot_imp,
    subtotal = EXCLUDED.subtotal,
    desc_not = EXCLUDED.desc_not,
    tot_nota = EXCLUDED.tot_nota,
    num = EXCLUDED.num,
    saldo = EXCLUDED.saldo,
    tot_des = EXCLUDED.tot_des,
    lugar = EXCLUDED.lugar,
    cve_mon = EXCLUDED.cve_mon,
    tip_cam = EXCLUDED.tip_cam,
    cve_suc = EXCLUDED.cve_suc,
    mes = EXCLUDED.mes,
    aqo = EXCLUDED.aqo,
    usuario = EXCLUDED.usuario,
    trans = EXCLUDED.trans
RETURNING *;


