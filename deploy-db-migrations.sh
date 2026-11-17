#!/usr/bin/env bash
#
# Aplica migraciones SQL a la base de datos de producción con seguimiento en
# public.schema_migrations.
#
# Uso habitual para la base existente:
#   1) ./deploy-db-migrations.sh status
#   2) ./deploy-db-migrations.sh baseline --through 29
#   3) ./deploy-db-migrations.sh up --dry-run
#   4) ./deploy-db-migrations.sh up
#
# La variable MIGRATION_DATABASE_URL debe usar un rol con permisos de DDL,
# creación de roles y, si las migraciones lo requieren, creación de extensiones.
# No uses chat_ro ni chat_audit (son roles del chat) ni un rol de solo lectura.
#
# La URL y el respaldo se pueden definir en .env.migrations:
#   MIGRATION_DATABASE_URL=postgres://...
#   MIGRATION_BACKUP_DIR=backups

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="$SCRIPT_DIR/internal/sql"
ENV_FILE="$SCRIPT_DIR/.env.migrations"

if [[ -f "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
fi

DB_URL="${MIGRATION_DATABASE_URL:-}"
BACKUP_DIR="${MIGRATION_BACKUP_DIR:-$SCRIPT_DIR/backups}"
case "$BACKUP_DIR" in
  /*) ;;
  *) BACKUP_DIR="$SCRIPT_DIR/$BACKUP_DIR" ;;
esac

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/kubo-migrations.XXXXXX")"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

log() {
  printf '%s\n' "$*"
}

die() {
  printf 'ERROR: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Uso:
  ./deploy-db-migrations.sh status
  ./deploy-db-migrations.sh baseline --through 29 [--include-optional] [--yes]
  ./deploy-db-migrations.sh up [--dry-run] [--include-optional] [--no-backup] [--yes]

Comandos:
  status     Muestra migraciones aplicadas, pendientes y diferencias de checksum.
  baseline   Marca como aplicadas las migraciones <= --through sin ejecutarlas.
             Úsalo una sola vez en la base existente antes del primer "up".
  up         Aplica las migraciones pendientes en orden.

Opciones de up:
  --dry-run           Solo muestra lo que se aplicaría.
  --include-optional  Incluye migraciones con "_opcional" en el nombre (por ejemplo RLS).
  --no-backup         No genera respaldo pg_dump antes de aplicar.
  --yes               No pide confirmación interactiva.

Variables:
  MIGRATION_DATABASE_URL  URL de producción con permisos de migración (obligatoria).
  MIGRATION_BACKUP_DIR    Directorio de respaldos (default: backups).

Ejemplo de .env.migrations:
  MIGRATION_DATABASE_URL=postgres://admin:password@host:5432/kubo_db?sslmode=prefer
  MIGRATION_BACKUP_DIR=backups

Notas:
  - La base existente debe tener el esquema base aplicado (usuarios, clientes,
    pedidos, productos_variantes). El script no está pensado para crear una
    base desde cero: las migraciones 01-29 no están ordenadas para eso.
  - Las migraciones 30-39 requieren un rol con DDL. Las migraciones 34 y 36
    crean roles y requieren permisos de CREATEROLE (o superusuario).
  - La migración 37 es opcional y no se aplica salvo --include-optional.
  - Cada archivo se aplica en una transacción junto con su registro en
    schema_migrations. Si falla, no queda registrado.
EOF
}

require_db() {
  if [[ -z "$DB_URL" ]]; then
    die "Falta MIGRATION_DATABASE_URL. Defínela en el entorno o en $ENV_FILE."
  fi
  case "$DB_URL" in
    postgres://*|postgresql://*) ;;
    *) die "MIGRATION_DATABASE_URL debe empezar con postgres:// o postgresql://" ;;
  esac
  command -v psql >/dev/null 2>&1 || die "psql no está instalado o no está en PATH."
}

safe_url() {
  printf '%s\n' "$DB_URL" | sed -E 's#(postgres(ql)?://[^:/@]+:)[^@]*@#\1***@#'
}

psql_q() {
  psql "$DB_URL" -X -q -A -t -v ON_ERROR_STOP=1 -c "$1"
}

base_schema_count() {
  psql_q "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('usuarios','clientes','pedidos','productos_variantes')"
}

migration_files() {
  find "$MIGRATIONS_DIR" -maxdepth 1 -type f -name '[0-9][0-9]_*.sql' | sort
}

checksum_for() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

is_optional() {
  case "$(basename "$1")" in
    *_opcional*) return 0 ;;
    *) return 1 ;;
  esac
}

table_exists() {
  [[ "$(psql_q "SELECT to_regclass('public.schema_migrations') IS NOT NULL")" == "t" ]]
}

load_applied() {
  psql_q "SELECT filename FROM public.schema_migrations ORDER BY filename" > "$TMP_DIR/applied_names"
  psql_q "SELECT filename || E'\t' || checksum FROM public.schema_migrations ORDER BY filename" > "$TMP_DIR/applied_checksums"
}

is_applied() {
  grep -Fxq "$(basename "$1")" "$TMP_DIR/applied_names"
}

check_checksums() {
  local mismatch=0
  : > "$TMP_DIR/checksum_errors"

  while IFS=$'\t' read -r name expected; do
    [[ -n "$name" ]] || continue
    local file="$MIGRATIONS_DIR/$name"
    if [[ ! -f "$file" ]]; then
      log "AVISO: migración registrada pero no encontrada en el repo: $name"
      continue
    fi
    local actual
    actual="$(checksum_for "$file")"
    if [[ "$actual" != "$expected" ]]; then
      printf '%s\n' "$name" >> "$TMP_DIR/checksum_errors"
      mismatch=1
    fi
  done < "$TMP_DIR/applied_checksums"

  if [[ "$mismatch" -ne 0 ]]; then
    log "ERROR: checksum distinto en migraciones ya aplicadas:"
    sed 's/^/  - /' "$TMP_DIR/checksum_errors"
    log "No se aplicará nada. Revisa si editaste una migración después de aplicarla."
  fi
  return "$mismatch"
}

# ---------------------------------------------------------------------------
# status
# ---------------------------------------------------------------------------
cmd_status() {
  require_db

  if ! table_exists; then
    log "No existe public.schema_migrations."
    log "La base es existente. Ejecuta primero:"
    log "  ./deploy-db-migrations.sh baseline --through <última versión aplicada>"
    exit 0
  fi

  load_applied
  local applied
  applied="$(wc -l < "$TMP_DIR/applied_names" | tr -d ' ')"
  log "Base: $(safe_url)"
  log "Migraciones registradas: $applied"
  log ""

  log "Pendientes:"
  local pending=0
  while IFS= read -r file; do
    local name
    name="$(basename "$file")"
    is_applied "$file" && continue
    if is_optional "$file"; then
      log "  $name (opcional)"
    else
      log "  $name"
    fi
    pending=$((pending + 1))
  done < <(migration_files)

  if [[ "$pending" -eq 0 ]]; then
    log "  (ninguna)"
  fi

  check_checksums || true
}

# ---------------------------------------------------------------------------
# baseline
# ---------------------------------------------------------------------------
cmd_baseline() {
  local through=""
  local include_optional="0"
  local yes="0"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --through)
        if [[ $# -lt 2 ]]; then
          die "--through requiere un valor."
        fi
        through="$2"
        shift 2
        ;;
      --include-optional)
        include_optional="1"
        shift
        ;;
      --yes)
        yes="1"
        shift
        ;;
      *)
        die "Opción desconocida para baseline: $1"
        ;;
    esac
  done

  require_db

  [[ -n "$through" ]] || die "baseline requiere --through <última versión aplicada>."
  [[ "$through" =~ ^[0-9]+$ ]] || die "--through debe ser un número."
  if (( 10#$through < 1 )); then
    die "--through debe ser mayor o igual a 1."
  fi

  local core
  core="$(base_schema_count)"
  if [[ "$core" != "4" ]]; then
    die "La base no parece tener el esquema base (se encontraron $core de 4 tablas). Abortando baseline."
  fi

  local chat_objects
  chat_objects="$(psql_q "SELECT (to_regclass('public.productos_busqueda') IS NOT NULL OR to_regclass('public.chat_productos') IS NOT NULL OR to_regclass('public.chat_escalations') IS NOT NULL)::text")"
  if [[ "$chat_objects" == "true" ]] && (( 10#$through < 35 )); then
    log "AVISO: ya existen objetos de las migraciones de chat, pero el baseline es <= $through."
    log "Producción podría tener migraciones de chat aplicadas. Verifica el estado real antes de continuar."
  fi

  local baseline_sql="$TMP_DIR/baseline.sql"
  local count=0

  {
    printf '%s\n' "BEGIN;"
    printf '%s\n' "CREATE TABLE IF NOT EXISTS public.schema_migrations ("
    printf '%s\n' "    filename TEXT PRIMARY KEY,"
    printf '%s\n' "    checksum TEXT NOT NULL,"
    printf '%s\n' "    baseline BOOLEAN NOT NULL DEFAULT FALSE,"
    printf '%s\n' "    applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP"
    printf '%s\n' ");"

    while IFS= read -r file; do
      local name version sum
      name="$(basename "$file")"
      version="${name%%_*}"

      if (( 10#$version > 10#$through )); then
        continue
      fi

      if is_optional "$file" && [[ "$include_optional" != "1" ]]; then
        printf 'Omitiendo opcional en baseline: %s\n' "$name" >&2
        continue
      fi

      sum="$(checksum_for "$file")"
      printf "INSERT INTO public.schema_migrations (filename, checksum, baseline) VALUES ('%s', '%s', TRUE) ON CONFLICT (filename) DO NOTHING;\n" "$name" "$sum"
      count=$((count + 1))
    done < <(migration_files)

    printf '%s\n' "COMMIT;"
  } > "$baseline_sql"

  if [[ "$count" -eq 0 ]]; then
    die "No hay migraciones <= $through para marcar."
  fi

  log "Base: $(safe_url)"
  log "Se marcarán $count migraciones (<= $through) como aplicadas, sin ejecutarlas."
  if [[ "$yes" != "1" ]]; then
    printf 'Escribe BASELINE para continuar: '
    read -r answer
    [[ "$answer" == "BASELINE" ]] || die "Cancelado."
  fi

  psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -f "$baseline_sql"
  log "Baseline completado. Ahora ejecuta: ./deploy-db-migrations.sh status"
}

# ---------------------------------------------------------------------------
# up
# ---------------------------------------------------------------------------
cmd_up() {
  local dry_run="0"
  local include_optional="0"
  local no_backup="0"
  local yes="0"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dry-run)
        dry_run="1"
        shift
        ;;
      --include-optional)
        include_optional="1"
        shift
        ;;
      --no-backup)
        no_backup="1"
        shift
        ;;
      --yes)
        yes="1"
        shift
        ;;
      *)
        die "Opción desconocida para up: $1"
        ;;
    esac
  done

  require_db

  if ! table_exists; then
    die "No existe public.schema_migrations. Ejecuta baseline --through <última versión aplicada> primero."
  fi

  load_applied
  if ! check_checksums; then
    exit 1
  fi

  local core
  core="$(base_schema_count)"
  if [[ "$core" != "4" ]]; then
    die "La base no tiene el esquema base (se encontraron $core de 4 tablas). Abortando."
  fi

  local applied_count
  applied_count="$(wc -l < "$TMP_DIR/applied_names" | tr -d ' ')"
  if [[ "$applied_count" -eq 0 ]]; then
    die "public.schema_migrations está vacío. Ejecuta baseline --through <última versión aplicada> primero."
  fi

  : > "$TMP_DIR/pending"
  local pending=0
  local omitted_optional=0

  while IFS= read -r file; do
    local name
    name="$(basename "$file")"
    is_applied "$file" && continue

    if is_optional "$file" && [[ "$include_optional" != "1" ]]; then
      omitted_optional=1
      continue
    fi

    printf '%s\n' "$file" >> "$TMP_DIR/pending"
    pending=$((pending + 1))
  done < <(migration_files)

  log "Base: $(safe_url)"
  if [[ "$pending" -eq 0 ]]; then
    log "No hay migraciones pendientes."
    if [[ "$omitted_optional" -eq 1 ]]; then
      log "Hay migraciones opcionales omitidas. Usa --include-optional para incluirlas."
    fi
    exit 0
  fi

  log "Migraciones pendientes:"
  while IFS= read -r file; do
    log "  $(basename "$file")"
  done < "$TMP_DIR/pending"
  if [[ "$omitted_optional" -eq 1 ]]; then
    log "  (migraciones opcionales omitidas; usa --include-optional)"
  fi

  if [[ "$dry_run" == "1" ]]; then
    log "Dry-run: no se aplicó nada."
    exit 0
  fi

  if [[ "$yes" != "1" ]]; then
    log ""
    log "Se aplicarán $pending migraciones en la base indicada."
    if [[ "$omitted_optional" -eq 1 ]]; then
      log "Las migraciones opcionales no se aplicarán."
    fi
    printf 'Escribe MIGRATE para continuar: '
    read -r answer
    [[ "$answer" == "MIGRATE" ]] || die "Cancelado."
  fi

  if [[ "$no_backup" != "1" ]]; then
    command -v pg_dump >/dev/null 2>&1 || die "pg_dump no está instalado o no está en PATH. Usa --no-backup solo si tienes otro respaldo."
    mkdir -p "$BACKUP_DIR"
    local backup="$BACKUP_DIR/kubo_$(date +%Y%m%d_%H%M%S).dump"
    log "Creando respaldo antes de migrar: $backup"
    pg_dump "$DB_URL" --format=custom --file="$backup"
    if [[ ! -s "$backup" ]]; then
      die "El respaldo quedó vacío o no se pudo crear: $backup. No se aplicó ninguna migración."
    fi
    chmod 600 "$backup" 2>/dev/null || true
    log "Respaldo completado."
  fi

  while IFS= read -r file; do
    local name sum
    name="$(basename "$file")"
    sum="$(checksum_for "$file")"

    if grep -Eiq 'create[[:space:]]+(unique[[:space:]]+)?index[[:space:]]+concurrently' "$file"; then
      die "La migración $name usa CREATE INDEX CONCURRENTLY y no puede ejecutarse en la transacción única del script. Aplícala manualmente."
    fi
    if grep -Eiq '^[[:space:]]*(begin|start[[:space:]]+transaction|commit|rollback)[[:space:]]*;' "$file"; then
      die "La migración $name contiene BEGIN/COMMIT/ROLLBACK explícito y no puede combinarse con el tracking automático."
    fi

    log "Aplicando $name..."
    {
      printf '%s\n' "SET search_path TO public;"
      cat "$file"
      printf "\nINSERT INTO public.schema_migrations (filename, checksum, baseline) VALUES ('%s', '%s', FALSE);\n" "$name" "$sum"
    } > "$TMP_DIR/migration.sql"

    psql "$DB_URL" -X -q -v ON_ERROR_STOP=1 -1 -f "$TMP_DIR/migration.sql"
    log "OK: $name"
  done < "$TMP_DIR/pending"

  log ""
  log "Migraciones aplicadas correctamente."
  log "Si es la primera vez, asigna contraseña a chat_ro y chat_audit y configura en el backend:"
  log "  ANTHROPIC_API_KEY, CHAT_DB_URL, CHAT_AUDIT_DB_URL"
  log "  (opcional: CHAT_ESCALATION_EMAILS, CHAT_ADMIN_PANEL_URL)"
}

# ---------------------------------------------------------------------------
# main
# ---------------------------------------------------------------------------
COMMAND="${1:-}"
if [[ $# -gt 0 ]]; then
  shift
fi

case "$COMMAND" in
  status)
    cmd_status "$@"
    ;;
  baseline)
    cmd_baseline "$@"
    ;;
  up)
    cmd_up "$@"
    ;;
  help|-h|--help|"")
    usage
    ;;
  *)
    die "Comando desconocido: $COMMAND. Usa --help."
    ;;
esac
