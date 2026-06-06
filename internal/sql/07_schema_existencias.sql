CREATE TABLE existencias_integracion (
    id SERIAL PRIMARY KEY,               -- Identificador único autoincremental para Postgres
    cse_prod VARCHAR(10),                -- Clase/Categoría del producto
    cve_prod VARCHAR(20) NOT NULL,       -- Clave del producto
    lugar VARCHAR(10) NOT NULL,          -- Almacén / Ubicación / Lugar
    existencia NUMERIC(20, 8),           -- Cantidad en existencia (Alta precisión)
    med_prod NUMERIC(5, 2),              -- Medida del producto
    fech_umod TIMESTAMP,                 -- Fecha de última modificación
    inv_ini BIGINT,                      -- Inventario inicial (N:8.0, sin decimales)
    lote VARCHAR(30),                    -- Número de lote
    fech_lote TIMESTAMP,                 -- Fecha del lote
    ref_lote VARCHAR(255),               -- Referencia del lote (Aumentado a 255 por si las dudas)
    costo_prom NUMERIC(20, 8),           -- Costo promedio
    new_med VARCHAR(6),                  -- Nueva medida
    costuepeps NUMERIC(20, 8),           -- Costo UEPS / PEPS
    costoprom2 NUMERIC(20, 8),            -- Costo promedio 2
    CONSTRAINT unique_producto_lugar UNIQUE (cve_prod, lugar)
);