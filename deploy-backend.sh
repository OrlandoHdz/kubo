#!/bin/bash
set -euo pipefail

IMAGE="kubo-api:v1"
TAR="kubo-api.tar"
SERVER="ecs-user@43.112.73.182"
KEY="/Users/orlando/Proyectos/Orlando/KUBO/Documentos/Key Servidor/orlandokey.pem"
REMOTE_PATH="/home/ecs-user/"
CONTAINER_NAME="kubo-api-container"
ENV_FILE="/home/ecs-user/kubo.env"

echo "=== 1. Verificando archivo de variables en el servidor ==="
if ! ssh -i "$KEY" "$SERVER" "
  test -s $ENV_FILE || { echo 'MISSING'; exit 2; }
  grep -q '^ANTHROPIC_API_KEY=.' $ENV_FILE || { echo 'MISSING_KEY:ANTHROPIC_API_KEY'; exit 2; }
  grep -q '^CHAT_DB_URL=.' $ENV_FILE || { echo 'MISSING_KEY:CHAT_DB_URL'; exit 2; }
  grep -q '^CHAT_AUDIT_DB_URL=.' $ENV_FILE || { echo 'MISSING_KEY:CHAT_AUDIT_DB_URL'; exit 2; }
  if grep -Eq '^[[:space:]]*[^#[:space:]][^=]*=.*(CAMBIAR_PASSWORD|reemplaza-esta-clave|@HOST:|tudominio\\.com)' $ENV_FILE; then
    echo 'PLACEHOLDERS'
    grep -nE '^[[:space:]]*[^#[:space:]][^=]*=.*(CAMBIAR_PASSWORD|reemplaza-esta-clave|@HOST:|tudominio\\.com)' $ENV_FILE | cut -d= -f1
    exit 3
  fi
"; then
  echo "ERROR: $ENV_FILE existe, pero no pasó la validación."
  echo "Revisa las variables de ejemplo que se muestran arriba (solo nombres, sin valores)."
  echo "El archivo debe contener ANTHROPIC_API_KEY, CHAT_DB_URL y CHAT_AUDIT_DB_URL."
  echo ""
  echo "Si el archivo no existe, créalo a partir de kubo.env.example:"
  echo "  cp kubo.env.example kubo.env"
  echo "  # edita kubo.env con los valores reales"
  echo "  scp -i \"$KEY\" kubo.env $SERVER:$ENV_FILE"
  echo "  ssh -i \"$KEY\" $SERVER \"chmod 600 $ENV_FILE\""
  exit 1
fi
echo "OK: $ENV_FILE"

echo "=== 2. Creando imagen Docker para linux/amd64 ==="
docker build --platform linux/amd64 -t "$IMAGE" .

echo "=== 3. Empaquetando imagen ==="
docker save -o "$TAR" "$IMAGE"

echo "=== 4. Subiendo a Alibaba Cloud ==="
scp -i "$KEY" "$TAR" "$SERVER:$REMOTE_PATH"

echo "=== 5. Cargando imagen en el servidor ==="
ssh -i "$KEY" "$SERVER" "sudo docker load -i $REMOTE_PATH$TAR"

echo "=== 6. Deteniendo y eliminando contenedor anterior ==="
ssh -i "$KEY" "$SERVER" "sudo docker stop $CONTAINER_NAME 2>/dev/null; sudo docker rm $CONTAINER_NAME 2>/dev/null; echo 'OK'"

echo "=== 7. Levantando nuevo contenedor ==="
ssh -i "$KEY" "$SERVER" "sudo docker run -d \
  -p 8080:8080 \
  -v /home/ecs-user/uploads:/app/uploads \
  --env-file $ENV_FILE \
  --restart unless-stopped \
  --name $CONTAINER_NAME \
  $IMAGE"

echo "=== Deploy completado ==="
