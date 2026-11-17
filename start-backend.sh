#!/bin/bash
set -euo pipefail

IMAGE="kubo-api:v1"
SERVER="ecs-user@43.112.73.182"
KEY="/Users/orlando/Proyectos/Orlando/KUBO/Documentos/Key Servidor/orlandokey.pem"
CONTAINER_NAME="kubo-api-container"
ENV_FILE="/home/ecs-user/kubo.env"

echo "=== 1. Verificando conexión con el servidor ==="
ssh -i "$KEY" "$SERVER" "echo OK: \$(hostname)"

echo "=== 2. Verificando que la imagen existe en el servidor ==="
if ! ssh -i "$KEY" "$SERVER" "sudo docker image inspect $IMAGE >/dev/null 2>&1"; then
  echo "ERROR: la imagen $IMAGE no existe en el servidor."
  echo "Ejecuta ./deploy-backend.sh para construirla y subirla."
  exit 1
fi
echo "OK: $IMAGE"

echo "=== 3. Verificando archivo de variables ==="
if ! ssh -i "$KEY" "$SERVER" "test -s $ENV_FILE"; then
  echo "ERROR: no existe o está vacío $ENV_FILE"
  exit 1
fi
echo "OK: $ENV_FILE"

echo "=== 4. Deteniendo y eliminando contenedor anterior (si existe) ==="
ssh -i "$KEY" "$SERVER" "sudo docker stop $CONTAINER_NAME 2>/dev/null || true; sudo docker rm $CONTAINER_NAME 2>/dev/null || true; echo 'OK'"

echo "=== 5. Levantando contenedor ==="
ssh -i "$KEY" "$SERVER" "sudo docker run -d \
  -p 8080:8080 \
  -v /home/ecs-user/uploads:/app/uploads \
  --env-file $ENV_FILE \
  --restart unless-stopped \
  --name $CONTAINER_NAME \
  $IMAGE"

echo "=== 6. Estado del contenedor ==="
ssh -i "$KEY" "$SERVER" "sudo docker ps --filter name=$CONTAINER_NAME --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"

echo "=== Backend levantado ==="
