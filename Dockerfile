# ==========================================
# ETAPA 1: Construcción (Builder)
# ==========================================
# Usamos la imagen oficial de Go (ligera) para compilar
FROM golang:1.26.2-alpine AS builder

# Directorio de trabajo dentro del contenedor
WORKDIR /app

# Descargamos las dependencias primero (esto optimiza el caché de Docker)
COPY go.mod go.sum ./
RUN go mod download

# Copiamos todo el código fuente
COPY . .

# Compilamos el binario. 
# CGO_ENABLED=0 es CRUCIAL para que el binario funcione en contenedores vacíos sin requerir librerías de C.
# GOOS=linux asegura que se compile para el sistema operativo del contenedor.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kubo-api ./cmd/api/main.go 

# ==========================================
# ETAPA 2: Imagen Final de Producción
# ==========================================
# Usamos 'alpine' que pesa solo ~5MB. (También podrías usar 'scratch' que pesa 0MB)
FROM alpine:latest

# Añadimos certificados raíz, zonas horarias y poppler-utils para el parseo de PDF
RUN apk --no-cache add ca-certificates tzdata poppler-utils

WORKDIR /app

# 1. Copiamos el binario compilado desde la ETAPA 1
COPY --from=builder /app/kubo-api .

# 2. Copiamos tu archivo de configuración manteniendo la estructura de carpetas
# Asegúrate de crear la carpeta de destino en el contenedor
RUN mkdir -p configs/db
COPY configs/db/database.yaml ./configs/db/

# Exponemos el puerto que usa tu API (ejemplo: 8080)
EXPOSE 8080

# Comando para arrancar la aplicación
CMD ["./kubo-api"]
