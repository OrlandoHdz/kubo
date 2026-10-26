-- Migración 39: contacto y seguimiento de escalamientos del chat.
--
-- chat_audit sigue teniendo solo INSERT. La bandeja de administración lee y
-- actualiza estas filas con el rol principal de la aplicación.

ALTER TABLE chat_escalations
    ADD COLUMN IF NOT EXISTS contacto_nombre TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS contacto_email TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS contacto_telefono TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS nota_admin TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS atendida_por INTEGER,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

COMMENT ON COLUMN chat_escalations.contacto_nombre
    IS 'Nombre proporcionado por el usuario en el formulario de asesor.';
COMMENT ON COLUMN chat_escalations.contacto_email
    IS 'Correo de contacto para seguimiento; puede ser el de la sesión o el del formulario.';
COMMENT ON COLUMN chat_escalations.contacto_telefono
    IS 'Teléfono de contacto opcional.';
COMMENT ON COLUMN chat_escalations.nota_admin
    IS 'Nota interna de seguimiento; no se muestra al cliente.';
COMMENT ON COLUMN chat_escalations.atendida_por
    IS 'Usuario interno que atendió o cerró la solicitud.';

CREATE INDEX IF NOT EXISTS idx_chat_escalations_updated
    ON chat_escalations (updated_at DESC);

-- Permiso para la pestaña del Panel de Control.
INSERT INTO permisos (clave, nombre, grupo, orden) VALUES
    ('chat-escalamientos', 'Chat / Asesor Humano', 'General', 18)
ON CONFLICT (clave) DO NOTHING;
