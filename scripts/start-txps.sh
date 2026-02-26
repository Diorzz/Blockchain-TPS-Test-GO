#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Building txps UI..."
cd "$ROOT_DIR/txps"
if [ ! -d node_modules ]; then
  echo "Installing dependencies..."
  npm install
fi
npm run build

cd "$ROOT_DIR"
mkdir -p logs
echo "Starting TPS server (metrics + static) on port 8081..."
# Run server in background
nohup go run tps_server.go > logs/tps_server.log 2>&1 &
SERVER_PID=$!
sleep 1
if ps -p $SERVER_PID > /dev/null; then
  echo "tps_server running (pid=$SERVER_PID)"
else
  echo "Failed to start tps_server, check logs/tps_server.log"
  exit 1
fi

echo "txps built. Serve static files from txps/dist at http://localhost:8081/ and metrics at /metrics"
