#!/bin/bash
set -euo pipefail

IMAGE="kubo-api:v1"
TAR="kubo-api.tar"
SERVER="ecs-user@43.112.73.182"
KEY="/Users/orlando/Proyectos/Orlando/KUBO/Documentos/Key Servidor/orlandokey.pem"
REMOTE_PATH="/home/ecs-user/"
CONTAINER_NAME="kubo-api-container"

echo "=== 1. Creando imagen Docker para linux/amd64 ==="
docker build --platform linux/amd64 -t "$IMAGE" .

echo "=== 2. Empaquetando imagen ==="
docker save -o "$TAR" "$IMAGE"

echo "=== 3. Subiendo a Alibaba Cloud ==="
scp -i "$KEY" "$TAR" "$SERVER:$REMOTE_PATH"

echo "=== 4. Cargando imagen en el servidor ==="
ssh -i "$KEY" "$SERVER" "sudo docker load -i $REMOTE_PATH$TAR"

echo "=== 5. Deteniendo y eliminando contenedor anterior ==="
ssh -i "$KEY" "$SERVER" "sudo docker stop $CONTAINER_NAME 2>/dev/null; sudo docker rm $CONTAINER_NAME 2>/dev/null; echo 'OK'"

echo "=== 6. Levantando nuevo contenedor ==="
ssh -i "$KEY" "$SERVER" "sudo docker run -d \
  -p 8080:8080 \
  -v /home/ecs-user/uploads:/app/uploads \
  --name $CONTAINER_NAME \
  $IMAGE"

echo "=== Deploy completado ==="