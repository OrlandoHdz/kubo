create table productos_integracion (
    id serial primary key,
    cse_prod varchar(10),
    cve_prod varchar(20),
    nom_prod varchar(40),
    desc_prod varchar(40),
    uni_med varchar(5),
    uni_med_p varchar(5),
    costo_prod numeric(20,8),
    f_act_cto TIMESTAMP,
    f_act_pre TIMESTAMP,
    cto_ent numeric(20,8),
    fec_ent TIMESTAMP,
    fec_ant TIMESTAMP,
    CONSTRAINT uinique_prod_cve_prod unique(cve_prod)
);