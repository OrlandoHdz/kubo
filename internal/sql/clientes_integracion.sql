-- name: CrearClienteIntegracion :one
INSERT INTO clientes_integracion (
    cve_cte, 
    nom_cte, 
    dir_cte, 
    col_cte, 
    cd_cte, 
    edo_cte, 
    rfc_cte, 
    cp_cte, 
    contacto, 
    tel1_cte, 
    lim_cre, 
    dia_cre, 
    cve_age, 
    lada_cte, 
    cve_zon, 
    cve_sub, 
    cve_can, 
    cuentacon, 
    contado, 
    pais_cte, 
    email_cte
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
) RETURNING *;

-- name: ObtenerClienteIntegracion :one
SELECT * FROM clientes_integracion
WHERE id = $1 LIMIT 1;

-- name: ObtenerClientesIntegracion :many
SELECT * FROM clientes_integracion;

-- name: ObtenerClientesIntegracionPorCveCte :one
SELECT * FROM clientes_integracion
WHERE cve_cte = $1 LIMIT 1;

-- name: UpsertClienteIntegracion :one
INSERT INTO clientes_integracion (
    cve_cte, nom_cte, dir_cte, col_cte, cd_cte, edo_cte, rfc_cte, cp_cte, 
    contacto, tel1_cte, lim_cre, dia_cre, cve_age, lada_cte, cve_zon, 
    cve_sub, cve_can, cuentacon, contado, pais_cte, email_cte
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
)
ON CONFLICT (cve_cte) DO UPDATE SET
    nom_cte = EXCLUDED.nom_cte,
    dir_cte = EXCLUDED.dir_cte,
    col_cte = EXCLUDED.col_cte,
    cd_cte = EXCLUDED.cd_cte,
    edo_cte = EXCLUDED.edo_cte,
    rfc_cte = EXCLUDED.rfc_cte,
    cp_cte = EXCLUDED.cp_cte,
    contacto = EXCLUDED.contacto,
    tel1_cte = EXCLUDED.tel1_cte,
    lim_cre = EXCLUDED.lim_cre,
    dia_cre = EXCLUDED.dia_cre,
    cve_age = EXCLUDED.cve_age,
    lada_cte = EXCLUDED.lada_cte,
    cve_zon = EXCLUDED.cve_zon,
    cve_sub = EXCLUDED.cve_sub,
    cve_can = EXCLUDED.cve_can,
    cuentacon = EXCLUDED.cuentacon,
    contado = EXCLUDED.contado,
    pais_cte = EXCLUDED.pais_cte,
    email_cte = EXCLUDED.email_cte
RETURNING *;
