CREATE TABLE facturas_integracion (
    id SERIAL PRIMARY KEY,               -- Identificador único autoincremental
    no_fac VARCHAR(50) NOT NULL,         -- Número de factura
    no_ped VARCHAR(50),                  -- Número de pedido
    cve_cte INTEGER,                     -- Clave del cliente
    cve_age INTEGER,                     -- Clave del agente
    falta_fac TIMESTAMP,                 -- Fecha de alta de la factura
    status_fac VARCHAR(10),              -- Estatus de la factura
    subt_fac NUMERIC(15, 2),             -- Subtotal de la factura
    descuento NUMERIC(15, 2),            -- Descuento
    descue NUMERIC(15, 2),               -- Descuento adicional / porcentaje
    total_fac NUMERIC(15, 2),            -- Total de la factura
    saldo_fac NUMERIC(15, 2),            -- Saldo de la factura
    f_pago TIMESTAMP,                    -- Fecha de pago
    contrarec VARCHAR(50),               -- Contrarecibo
    lugar VARCHAR(255),                  -- Lugar
    cve_factu VARCHAR(50),               -- Clave de facturación
    cve_mon VARCHAR(10),                 -- Clave de moneda (ej. pesos, dólares)
    tip_cam NUMERIC(15, 4),              -- Tipo de cambio
    saldo_fac2 NUMERIC(15, 2),           -- Saldo de la factura 2
    staley VARCHAR(50),                  -- Campo Staley
    cve_suc INTEGER,                     -- Clave de sucursal
    mes INTEGER,                         -- Mes
    anio INTEGER,                        -- Año (representa A—O / AÑO)
    usuario VARCHAR(100),                -- Usuario
    trans VARCHAR(50),                   -- Transacción / Transporte
    staley2 VARCHAR(50),                 -- Campo Staley 2
    ped_int VARCHAR(50),                 -- Pedido interno
    com_cob NUMERIC(15, 2),              -- Comisión cobrada
    cierre TIMESTAMP,                    -- Fecha/estado de cierre
    u_tip_cam NUMERIC(15, 4),            -- Último tipo de cambio
    cve_age2 INTEGER,                    -- Clave del agente 2
    com_cob2 NUMERIC(15, 2),             -- Comisión cobrada 2
    din_com NUMERIC(15, 2),              -- Dinero comisión
    din_com2 NUMERIC(15, 2),             -- Dinero comisión 2
    fech_sal TIMESTAMP,                  -- Fecha de salida
    fech_emb TIMESTAMP,                  -- Fecha de embarque
    cve_flet VARCHAR(50),                -- Clave de flete
    tot_flet NUMERIC(15, 2),             -- Total flete
    tot_env NUMERIC(15, 2),              -- Total envío
    prec_flet NUMERIC(15, 2),            -- Precio flete
    prv_real VARCHAR(100),               -- Proveedor real
    timp_car NUMERIC(15, 2),             -- Tiempo de carga / importe cargo
    numpol VARCHAR(50),                  -- Número de póliza
    numpolcan VARCHAR(50),               -- Número de póliza cancelada
    imp_carac NUMERIC(15, 2),            -- Importe características
    pesotot NUMERIC(15, 2),              -- Peso total
    retiva_fac NUMERIC(15, 2),           -- Retención de IVA factura
    deposito NUMERIC(15, 2),             -- Depósito
    suc_depo VARCHAR(50),                -- Sucursal depósito
    status_2 VARCHAR(10),                -- Estatus 2
    cvedirent VARCHAR(50),               -- Clave dirección de entrega
    reimpre INTEGER,                     -- Reimpresiones
    descue2 NUMERIC(15, 2),              -- Descuento 2
    descue3 NUMERIC(15, 2),              -- Descuento 3
    descue4 NUMERIC(15, 2),              -- Descuento 4
    no_peda VARCHAR(50),                 -- Número de pedimento
    cve_suca VARCHAR(50),                -- Clave de sucursal A
    cve_factua VARCHAR(50),              -- Clave de facturación A
    cve_entre VARCHAR(50),               -- Clave de entrega
    ieps NUMERIC(15, 2),                 -- IEPS
    CONSTRAINT unique_no_fac_cve_factu UNIQUE (no_fac,cve_factu)
);
