#!/bin/bash
set -euo pipefail

OSSUTIL="$HOME/.local/bin/ossutil"
BUCKET="oss://kubo-frontend"
SOURCE="/Users/orlando/Proyectos/Orlando/KUBO/app/kubo-web/dist/spa"
REGION="us-east-1"

echo "=== Subiendo frontend a OSS (bucket: kubo-frontend) ==="
$OSSUTIL sync "$SOURCE/" "$BUCKET/" --region "$REGION" --delete --force

echo "=== Subida completada ==="