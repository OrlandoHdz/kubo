CREATE INDEX IF NOT EXISTS idx_pedidos_created_at ON pedidos(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backorders_created_at ON backorders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_devoluciones_garantias_created_at ON devoluciones_garantias(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_historial_facturas_pagadas_created_at ON historial_facturas_pagadas(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_facturas_integracion_falta_fac ON facturas_integracion(falta_fac DESC);
