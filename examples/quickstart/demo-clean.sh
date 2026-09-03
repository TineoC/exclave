#!/usr/bin/env bash
set -euo pipefail
CLUSTER=${CLUSTER:-exclave-demo}
REG_NAME=${REG_NAME:-exclave-registry}
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

kind delete cluster --name "$CLUSTER" 2>/dev/null || true
docker rm -f "$REG_NAME" 2>/dev/null || true
rm -rf "$HERE/.demo" "$HERE/charts/product/charts" "$HERE/charts/svc/charts" \
       "$HERE/charts/product/Chart.lock" "$HERE/charts/svc/Chart.lock"
docker rmi -f "$(docker images -q 'localhost:5001/acme/*' 2>/dev/null)" 2>/dev/null || true
echo "cleaned up"
