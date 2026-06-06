-- name: CrearFacturaIntegracion :one
INSERT INTO facturas_integracion (
    no_fac, no_ped, cve_cte, cve_age, falta_fac, status_fac, subt_fac, descuento,
    descue, total_fac, saldo_fac, f_pago, contrarec, lugar, cve_factu, cve_mon,
    tip_cam, saldo_fac2, staley, cve_suc, mes, anio, usuario, trans, staley2,
    ped_int, com_cob, cierre, u_tip_cam, cve_age2, com_cob2, din_com, din_com2,
    fech_sal, fech_emb, cve_flet, tot_flet, tot_env, prec_flet, prv_real, timp_car,
    numpol, numpolcan, imp_carac, pesotot, retiva_fac, deposito, suc_depo, status_2,
    cvedirent, reimpre, descue2, descue3, descue4, no_peda, cve_suca, cve_factua,
    cve_entre, ieps
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44, $45, $46, $47, $48, $49, $50, $51, $52, $53, $54, $55, $56, $57, $58, $59
) RETURNING *;

-- name: ObtenerFacturaIntegracion :one
SELECT * FROM facturas_integracion
WHERE id = $1 LIMIT 1;

-- name: ObtenerFacturasIntegracion :many
SELECT * FROM facturas_integracion order by falta_fac desc, no_fac desc;

-- name: ObtenerFacturaIntegracionPorNoFac :one
SELECT * FROM facturas_integracion
WHERE no_fac = $1 LIMIT 1;

-- name: UpsertFacturaIntegracion :one
INSERT INTO facturas_integracion (
    no_fac, no_ped, cve_cte, cve_age, falta_fac, status_fac, subt_fac, descuento,
    descue, total_fac, saldo_fac, f_pago, contrarec, lugar, cve_factu, cve_mon,
    tip_cam, saldo_fac2, staley, cve_suc, mes, anio, usuario, trans, staley2,
    ped_int, com_cob, cierre, u_tip_cam, cve_age2, com_cob2, din_com, din_com2,
    fech_sal, fech_emb, cve_flet, tot_flet, tot_env, prec_flet, prv_real, timp_car,
    numpol, numpolcan, imp_carac, pesotot, retiva_fac, deposito, suc_depo, status_2,
    cvedirent, reimpre, descue2, descue3, descue4, no_peda, cve_suca, cve_factua,
    cve_entre, ieps
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44, $45, $46, $47, $48, $49, $50, $51, $52, $53, $54, $55, $56, $57, $58, $59
)
ON CONFLICT (no_fac, cve_factu) DO UPDATE SET
    no_ped = EXCLUDED.no_ped,
    cve_cte = EXCLUDED.cve_cte,
    cve_age = EXCLUDED.cve_age,
    falta_fac = EXCLUDED.falta_fac,
    status_fac = EXCLUDED.status_fac,
    subt_fac = EXCLUDED.subt_fac,
    descuento = EXCLUDED.descuento,
    descue = EXCLUDED.descue,
    total_fac = EXCLUDED.total_fac,
    saldo_fac = EXCLUDED.saldo_fac,
    f_pago = EXCLUDED.f_pago,
    contrarec = EXCLUDED.contrarec,
    lugar = EXCLUDED.lugar,
    cve_mon = EXCLUDED.cve_mon,
    tip_cam = EXCLUDED.tip_cam,
    saldo_fac2 = EXCLUDED.saldo_fac2,
    staley = EXCLUDED.staley,
    cve_suc = EXCLUDED.cve_suc,
    mes = EXCLUDED.mes,
    anio = EXCLUDED.anio,
    usuario = EXCLUDED.usuario,
    trans = EXCLUDED.trans,
    staley2 = EXCLUDED.staley2,
    ped_int = EXCLUDED.ped_int,
    com_cob = EXCLUDED.com_cob,
    cierre = EXCLUDED.cierre,
    u_tip_cam = EXCLUDED.u_tip_cam,
    cve_age2 = EXCLUDED.cve_age2,
    com_cob2 = EXCLUDED.com_cob2,
    din_com = EXCLUDED.din_com,
    din_com2 = EXCLUDED.din_com2,
    fech_sal = EXCLUDED.fech_sal,
    fech_emb = EXCLUDED.fech_emb,
    cve_flet = EXCLUDED.cve_flet,
    tot_flet = EXCLUDED.tot_flet,
    tot_env = EXCLUDED.tot_env,
    prec_flet = EXCLUDED.prec_flet,
    prv_real = EXCLUDED.prv_real,
    timp_car = EXCLUDED.timp_car,
    numpol = EXCLUDED.numpol,
    numpolcan = EXCLUDED.numpolcan,
    imp_carac = EXCLUDED.imp_carac,
    pesotot = EXCLUDED.pesotot,
    retiva_fac = EXCLUDED.retiva_fac,
    deposito = EXCLUDED.deposito,
    suc_depo = EXCLUDED.suc_depo,
    status_2 = EXCLUDED.status_2,
    cvedirent = EXCLUDED.cvedirent,
    reimpre = EXCLUDED.reimpre,
    descue2 = EXCLUDED.descue2,
    descue3 = EXCLUDED.descue3,
    descue4 = EXCLUDED.descue4,
    no_peda = EXCLUDED.no_peda,
    cve_suca = EXCLUDED.cve_suca,
    cve_factua = EXCLUDED.cve_factua,
    cve_entre = EXCLUDED.cve_entre,
    ieps = EXCLUDED.ieps
RETURNING *;
