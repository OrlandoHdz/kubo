CREATE TABLE clientes_integracion (
    id SERIAL PRIMARY KEY,               -- Identificador único autoincremental
    cve_cte INTEGER,                     -- Clave del cliente del sistema original
    nom_cte VARCHAR(255),
    dir_cte VARCHAR(255),
    col_cte VARCHAR(255),
    cd_cte VARCHAR(100),
    edo_cte VARCHAR(100),
    rfc_cte VARCHAR(20),
    cp_cte VARCHAR(10),
    contacto VARCHAR(255),
    tel1_cte VARCHAR(50),
    lim_cre NUMERIC(15, 2),
    dia_cre INTEGER,
    cve_age INTEGER,
    lada_cte VARCHAR(10),
    cve_zon INTEGER,
    cve_sub INTEGER,
    cve_can INTEGER,
    cuentacon VARCHAR(50),
    contado INTEGER,
    pais_cte VARCHAR(100),
    email_cte VARCHAR(255),
    CONSTRAINT unique_cve_cte UNIQUE (cve_cte)
);
