# ==========================================
# ETAPA 1: Construcción (Builder)
# ==========================================
FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# Descargamos dependencias primero para optimizar caché
COPY go.mod go.sum ./
RUN go mod download

# Copiamos el código fuente
COPY . .

# Compilamos el binario estático
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o kubo-api ./cmd/api/main.go 

# ==========================================
# ETAPA 2: Imagen Final de Production
# ==========================================
FROM alpine:3.19

# Instalar dependencias necesarias y limpiar caché de apk
RUN apk --no-cache add ca-certificates tzdata poppler-utils \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# Copiamos el binario desde la etapa de construcción
COPY --from=builder /app/kubo-api .

# Copiamos la configuración
COPY configs/db/database.yaml ./configs/db/

# --- CONFIGURACIÓN DE VOLUMEN Y USUARIO ---
# Creamos el usuario sin privilegios
RUN adduser -D -u 10001 appuser

# 1. 👇 REGLA DE ORO: Asegurar que el usuario pueda escribir archivos temporales grandes
RUN chown -R appuser:appuser /tmp

# 2. Creamos la carpeta de subidas y le asignamos la propiedad al appuser
# Esto garantiza que tu API de Go pueda escribir, leer y borrar archivos ahí dentro.
RUN mkdir -p uploads && chown -R appuser:appuser /app/uploads

# 3. Declaramos formalmente la carpeta como un volumen
VOLUME ["/app/uploads"]

# Cambiamos al usuario seguro para la ejecución
USER appuser

EXPOSE 8080

CMD ["./kubo-api"]