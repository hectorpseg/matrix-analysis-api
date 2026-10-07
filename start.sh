#!/bin/sh
set -eu

# Render injects PORT; default to 8080 for local runs.
PORT=${PORT:-8080}
export PORT

# Go always reaches Node on localhost inside the combined container.
export NODE_API_URL=http://127.0.0.1:3001

# Start Node stats API in the background, always on port 3001.
PORT=3001 node /app/src/server.js &
NODE_PID=$!

# Simple startup ordering: wait until Node's health endpoint responds.
I=0
while [ "$I" -lt 30 ]; do
  if wget -qO- http://127.0.0.1:3001/health >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$NODE_PID" 2>/dev/null; then
    echo "node-api exited before becoming healthy" >&2
    exit 1
  fi
  I=$((I + 1))
  sleep 0.5
done

# Start Go API in the background so signals can be forwarded to both processes.
/bin/app &
GO_PID=$!

cleanup() {
  echo "received signal, shutting down..." >&2
  kill -TERM "$NODE_PID" "$GO_PID" 2>/dev/null || true
  wait "$NODE_PID" 2>/dev/null || true
  wait "$GO_PID" 2>/dev/null || true
  exit 0
}
trap cleanup TERM INT

# The container stays alive as long as the Go API is running.
wait "$GO_PID"
