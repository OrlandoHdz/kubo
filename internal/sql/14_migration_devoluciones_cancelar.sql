ALTER TABLE devoluciones_garantias DROP CONSTRAINT IF EXISTS devoluciones_garantias_estatus_check;
ALTER TABLE devoluciones_garantias ADD CONSTRAINT devoluciones_garantias_estatus_check CHECK (estatus IN ('Pendiente', 'Aprobada', 'Rechazada', 'Cancelada'));
