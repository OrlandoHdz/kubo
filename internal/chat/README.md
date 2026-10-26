# Chat inteligente de Kubo

Asesor virtual basado en Claude con tool use. El navegador habla por SSE con
Go; Go consulta PostgreSQL con sqlc usando un rol de solo lectura.

## Arquitectura

```text
Quasar
  └─ POST /api/chat (SSE)
       └─ Go internal/chat
            ├─ Claude /v1/messages (stream + tools)
            ├─ chat_ro  ──> vistas SQL (catálogo y pedidos)
            └─ chat_audit ─> INSERT en chat_conversations/chat_messages/chat_escalations
```

- `CHAT_DB_URL` usa el rol `chat_ro`: solo `SELECT` sobre las vistas de
  catálogo, pedidos, backorders, devoluciones, recompensas y niveles;
  `statement_timeout=3s`, `default_transaction_read_only=on`.
- `CHAT_AUDIT_DB_URL` usa el rol `chat_audit`: solo `INSERT` en las tablas de
  auditoría.
- Las herramientas de pedidos reciben el `cliente_id` desde la sesión JWT y la
  vista `chat_usuarios_clientes`; el modelo nunca puede enviarlo.
- Las consultas de pedidos se ejecutan en una transacción read-only con
  `set_config('app.customer_id', ..., true)`. La migración 37 agrega RLS
  opcional como defensa adicional.

## Herramientas

| Herramienta | Sesión | Contenido |
|---|---|---|
| `buscar_productos` | No | Catálogo, máx. 8 resultados |
| `obtener_producto` | No | Detalle por SKU |
| `consultar_pedido` | Sí | Pedido por folio del cliente |
| `listar_pedidos_cliente` | Sí | Últimos pedidos, máx. 10 |
| `consultar_backorder` | Sí | Backorder por folio + items |
| `listar_backorders_cliente` | Sí | Últimos backorders, máx. 10 |
| `consultar_devolucion` | Sí | Devolución/garantía por folio |
| `listar_devoluciones_cliente` | Sí | Últimas devoluciones, máx. 10 |
| `consultar_recompensas_cliente` | Sí | Puntos, nivel actual/siguiente, promedio 90 días y movimientos |
| `listar_niveles_recompensa` | No | Catálogo público de niveles y parámetros |
| `escalar_a_humano` | No | Registra atención humana con contacto |

Las herramientas marcadas con “Sí” obtienen el `cliente_id` de la sesión JWT,
nunca de los argumentos del modelo. Las vistas de backorders y devoluciones no
exponen notas administrativas, evidencias ni datos de pago.

## Escalamiento a humano

Cuando el usuario pide un asesor:

1. El frontend abre el formulario “Hablar con un asesor”.
2. Se capturan nombre y al menos un correo o teléfono; el correo de la sesión
   se usa como respaldo si falta.
3. El backend guarda el contacto en `chat_escalations` (nunca en el historial
   del modelo) y envía un correo al equipo con los datos y el motivo.
4. La solicitud aparece en el Panel de Control, pestaña
   **Chat / Asesor Humano**, con la conversación completa.
5. El administrador puede marcarla como atendida, cerrarla, reabrirla y dejar
   una nota interna.

Si un invitado pide un asesor escribiendo en el chat sin usar el formulario, la
herramienta responde `requiere_contacto` y el asistente le pide usar el botón
para dejar sus datos. `chat_audit` sigue teniendo solo INSERT; la bandeja usa el
pool principal de la aplicación para leer y actualizar.

El permiso de la pestaña es `chat-escalamientos`. Los administradores sin
permisos configurados lo ven automáticamente; si tienen permisos explícitos,
deben activarlo en la gestión de usuarios.

## Variables de entorno

Copiar `.env.example` a `.env`. El backend carga `.env` sin sobrescribir
variables ya exportadas.

| Variable | Obligatoria | Default | Descripción |
|---|---|---|---|
| `ANTHROPIC_API_KEY` | Sí | — | Solo backend |
| `CLAUDE_MODEL` | No | `claude-sonnet-5-5` | Alternativa: `claude-haiku-5-5` |
| `CHAT_DB_URL` | Para herramientas | — | Rol `chat_ro` |
| `CHAT_AUDIT_DB_URL` | Para auditoría | — | Rol `chat_audit` |
| `CHAT_ESCALATION_EMAILS` | No | `default_recipients` de `email.yaml` | Correos separados por coma |
| `CHAT_ADMIN_PANEL_URL` | No | `http://localhost:9000/app/dashboard` | Enlace en el correo |
| `CHAT_MAX_TOKENS` | No | `2048` | Tokens máximos por respuesta |
| `CHAT_RATE_LIMIT` | No | `20` | Solicitudes/minuto por usuario o IP |
| `CHAT_MAX_MESSAGE_CHARS` | No | `2000` | Límite del mensaje actual |
| `CHAT_MAX_HISTORY_MESSAGES` | No | `12` | Mensajes de historial aceptados |
| `CHAT_MAX_HISTORY_CHARS` | No | `12000` | Caracteres de historial |
| `CHAT_LLM_TIMEOUT` | No | `60s` | Timeout por llamada a Claude |
| `ANTHROPIC_BASE_URL` | No | `https://api.anthropic.com` | Útil para mocks |

Sin `CHAT_DB_URL` el chat funciona en modo texto, pero sin herramientas de
catálogo, pedidos, backorders, devoluciones ni recompensas. Sin
`CHAT_AUDIT_DB_URL` no se guarda auditoría ni se registra el escalamiento.

## Migraciones

Aplicar en desarrollo en este orden, con un rol que pueda crear roles y ceder
propiedad de vistas (idealmente un superusuario o el dueño de la base):

1. `30_chat_extensiones_funciones.sql`
2. `31_chat_productos_soldadura.sql`
3. `32_chat_pedidos_tracking.sql`
4. `33_chat_vistas.sql`
5. `34_chat_roles_read.sql`
6. `35_chat_auditoria.sql`
7. `36_chat_roles_audit.sql`
8. `37_chat_rls_opcional.sql` (opcional)
9. `38_chat_expediente_recompensas.sql` (backorders, devoluciones, recompensas y niveles)
10. `39_chat_escalamientos_contacto.sql` (contacto, bandeja admin y permiso)

Los roles se crean sin contraseña. Asignarlas fuera del repositorio:

```sql
ALTER ROLE chat_ro WITH PASSWORD '...';
ALTER ROLE chat_audit WITH PASSWORD '...';
```

Verificar que `chat_ro` solo vea las vistas:

```sql
SET ROLE chat_ro;
SELECT * FROM chat_productos LIMIT 1;
SELECT * FROM usuarios LIMIT 1; -- debe fallar
RESET ROLE;
```

Si la migración 34 no puede transferir la propiedad de las vistas, emite un
aviso. Las consultas siguen seguras por el filtro explícito de `cliente_id`;
solo la RLS opcional de la migración 37 no aplicará.

La migración 37 es opcional. Si el rol de la app principal no es dueño de
`pedidos` y `pedido_detalles`, RLS también le aplicará; en ese caso agrega una
política para ese rol o no apliques la migración.

## Ejecutar

```bash
cd /Users/orlando/Proyectos/Orlando/KUBO/app/kubo
cp .env.example .env
# editar .env
go run ./cmd/api
```

El chat se registra en:

- `POST /api/chat`
- `POST /api/v1/chat` (alias)

La autenticación es opcional a nivel de ruta: invitados pueden buscar
productos; las herramientas de pedidos responden que se requiere iniciar
sesión. Si se envía un `Authorization` inválido, la petición devuelve 401.

## Probar con curl

Invitado:

```bash
curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d '{"message":"Busco electrodos 6013 de 1/8"}'
```

Con sesión:

```bash
TOKEN="<jwt>"
curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message":"¿Dónde está mi pedido KUBO-DEMO-001?"}'
```

Continuar una conversación (el id llega en el evento `meta`):

```bash
curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d '{
    "message":"¿Y cuál es el total?",
    "conversation_id":"<32 hex>",
    "history":[
      {"role":"user","content":"¿Dónde está mi pedido KUBO-DEMO-001?"},
      {"role":"assistant","content":"..."}
    ]
  }'
```

## Eventos SSE

```text
event: meta       {"conversation_id":"...","model":"..."}
event: status     {"stage":"tool_running","tool":"buscar_productos"}
event: text       {"delta":"..."}
event: products   {"items":[{...}]}
event: escalation {"motivo":"...","estado":"pendiente"}
event: done       {"conversation_id":"...","stop_reason":"end_turn","usage":{...}}
event: error      {"code":"...","message":"...","retryable":true}
```

## Pruebas

```bash
cd /Users/orlando/Proyectos/Orlando/KUBO/app/kubo
sqlc generate
sqlc vet
go build ./...
go test ./...
```

Prueba de integración (dos clientes y pedidos cruzados):

```bash
CHAT_TEST_DB_URL='postgres://chat_ro:...@localhost:5432/kubo_test?sslmode=disable' \
CHAT_TEST_ADMIN_DB_URL='postgres://admin:...@localhost:5432/kubo_test?sslmode=disable' \
go test -tags=integration ./internal/chat/...
```

La base de pruebas debe tener las migraciones 30–34 y 38 aplicadas. El test se
salta si faltan las variables.

## Frontend

```bash
cd /Users/orlando/Proyectos/Orlando/KUBO/app/kubo-web
npm install
npm run lint
npm run build
```

El componente está en `src/components/StoreChatBot.vue`, el cliente SSE en
`src/services/chat.js` y el estado en `src/composables/useChat.js`. El markdown
se renderiza con `markdown-it` y se sanitiza con `DOMPurify`.

Nota: `@quasar/app-vite` requiere Node >= 22.22. El entorno actual tiene Node
20; usar el Node 22 instalado en `~/.nvm/versions/node/v22.23.1/bin` para
compilar.

## Seguridad

- La API key de Anthropic nunca llega al navegador.
- Rate limit por usuario autenticado o IP. El límite por IP usa `RemoteAddr`; detrás de un proxy inverso todos los invitados pueden compartir la IP del proxy, así que ajusta `CHAT_RATE_LIMIT` o exige sesión si eso es un problema.
- Límites de mensaje, historial, tokens e iteraciones.
- Herramientas de solo lectura; el chat no modifica carrito, pedidos,
  backorders, devoluciones, recompensas ni pagos.
- “Agregar al carrito” es una acción explícita del usuario en la tarjeta, y
  revalida el precio contra `/productos/:padre_id/variantes`.
- La auditoría no guarda JWT, contraseñas ni datos de pago.
- RLS es opcional; el filtro por `cliente_id` es obligatorio en todas las
  consultas de pedidos, backorders y devoluciones. Las recompensas también se
  consultan por el `cliente_id` de la sesión.

## Fuera de alcance

- Búsqueda semántica con pgvector / `buscar_conocimiento()`.
- Modificar el checkout existente para que el servidor recalcule todos los
  precios al crear el pedido. El chat revalida el precio al agregar al carrito,
  pero el flujo de creación de pedidos actual sigue recibiendo precios del
  cliente.
- Endurecer el secreto JWT hardcodeado en `internal/auth/jwt.go`.
