# Variables
MAIN_PACKAGE=cmd/api/main.go
SEED_PACKAGE=cmd/seed/main.go
DB_TEST_PACKAGE=cmd/setup_db/main.go
WORKER_PACKAGE=cmd/worker/main.go

# DBF paths (ajusta si es necesario)
CLIENTES_DBF=$(HOME)/Proyectos/Orlando/KUBO/BaseDatos_SAI/CLIENTES/CLIENTES.DBF
EXISTENCIAS_DBF=$(HOME)/Proyectos/Orlando/KUBO/BaseDatos_SAI/EXISTENCIAS/EXISTE.DBF
FACTURAS_DBF=$(HOME)/Proyectos/Orlando/KUBO/BaseDatos_SAI/FACTURAS/facturac.DBF
CREDITOS_DBF=$(HOME)/Proyectos/Orlando/KUBO/BaseDatos_SAI/CREDITOS/creditos.DBF
PRODUCTOS_DBF=$(HOME)/Proyectos/Orlando/KUBO/BaseDatos_SAI/PRODUCTO/PRODUCTO.DBF

# Colores para la terminal (opcional pero profesional)
YELLOW=\033[0;33m
NC=\033[0m

.PHONY: build run seed test-db help sync-clients sync-existencias sync-facturas sync-creditos sync-productos sqlc

## help: Muestra los comandos disponibles
help:
	@echo "Comandos disponibles para Alialloys - Kubo:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## build: Compila todos los binarios y los guarda en la carpeta bin/
build:
	@echo "${YELLOW}Compilando todos los binarios...${NC}"
	@mkdir -p bin
	go build -o bin/api ./cmd/api
	go build -o bin/seed ./cmd/seed
	go build -o bin/setup_db ./cmd/setup_db
	go build -o bin/worker ./cmd/worker

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
	@echo "${YELLOW}Probando conexión a PostgreSQL...${NC}"
	go run $(DB_TEST_PACKAGE)

## sqlc: Generar código Go a partir de archivos SQL
sqlc:
	@echo "${YELLOW}Generando código de sqlc...${NC}"
	sqlc generate

## sync-clients: Sincronizar clientes desde el ERP SAI (DBF)
sync-clients:
	@echo "${YELLOW}Sincronizando clientes desde SAI ERP...${NC}"
	go run $(WORKER_PACKAGE) -clientes $(CLIENTES_DBF)

## sync-existencias: Sincronizar existencias desde el ERP SAI (DBF)
sync-existencias:
	@echo "${YELLOW}Sincronizando existencias desde SAI ERP...${NC}"
	go run $(WORKER_PACKAGE) -existencias $(EXISTENCIAS_DBF)

## sync-facturas: Sincronizar facturas desde el ERP SAI (DBF)
sync-facturas:
	@echo "${YELLOW}Sincronizando facturas desde SAI ERP...${NC}"
	go run $(WORKER_PACKAGE) -facturas $(FACTURAS_DBF)

## sync-creditos: Sincronizar creditos desde el ERP SAI (DBF)
sync-creditos:
	@echo "${YELLOW}Sincronizando creditos desde SAI ERP...${NC}"
	go run $(WORKER_PACKAGE) -creditos $(CREDITOS_DBF)

## sync-productos: Sincronizar productos desde el ERP SAI (DBF)
sync-productos:
	@echo "${YELLOW}Sincronizando productos desde SAI ERP...${NC}"
	go run $(WORKER_PACKAGE) -productos $(PRODUCTOS_DBF)