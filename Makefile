# Variables
MAIN_PACKAGE=cmd/api/main.go
SEED_PACKAGE=cmd/seed/main.go
DB_TEST_PACKAGE=cmd/setup_db/main.go

# Colores para la terminal (opcional pero profesional)
YELLOW=\033[0;33m
NC=\033[0m

.PHONY: run seed test-db help

## help: Muestra los comandos disponibles
help:
	@echo "Comandos disponibles para Alialloys - Kubo:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## run: Ejecuta la aplicación principal (API)
run:
	@echo "${YELLOW}Iniciando servidor de Kubo...${NC}"
	go run $(MAIN_PACKAGE)

## seed: Poblar la base de datos con datos de prueba (usuarios, clientes)
seed:
	@echo "${YELLOW}Poblando base de datos con datos de prueba...${NC}"
	go run $(SEED_PACKAGE)

## test-db: Probar la conexión a PostgreSQL con la config actual
test-db:
	@echo "${YELLOW}Probando conexión a la base de datos...${NC}"
	go run $(DB_TEST_PACKAGE)

## sqlc: Generar código Go a partir de archivos SQL
sqlc:
	@echo "${YELLOW}Generando código de sqlc...${NC}"
	sqlc generate
	