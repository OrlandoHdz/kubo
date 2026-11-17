#!/bin/bash
set -euo pipefail

OSSUTIL="$HOME/.local/bin/ossutil"
BUCKET="oss://kubo-frontend"
FRONTEND_DIR="/Users/orlando/Proyectos/Orlando/KUBO/app/kubo-web"
SOURCE="$FRONTEND_DIR/dist/spa"
REGION="us-east-1"
NODE_BIN="$HOME/.nvm/versions/node/v22.23.1/bin"

if [ ! -x "$NODE_BIN/npm" ]; then
  echo "ERROR: No encontré npm en $NODE_BIN"
  exit 1
fi

echo "=== 1. Construyendo frontend con Node 22 ==="
(cd "$FRONTEND_DIR" && PATH="$NODE_BIN:$PATH" npm run build)

echo "=== 2. Subiendo frontend a OSS (bucket: kubo-frontend) ==="
$OSSUTIL sync "$SOURCE/" "$BUCKET/" --region "$REGION" --delete --force

echo "=== Subida completada ==="