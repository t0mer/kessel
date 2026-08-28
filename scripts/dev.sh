#!/usr/bin/env bash
# Runs the backend and the Vite dev server together with hot reload.
# Backend on :8080, frontend on :5173 (proxying /api, /healthz, /metrics).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

DATA_DIR="${DATA_DIR:-./data}"
mkdir -p "$DATA_DIR"

echo ">> starting backend on :8080 (data: $DATA_DIR)"
go run ./cmd/kessel serve --data-dir "$DATA_DIR" --log-format text &
BACK=$!
trap 'kill $BACK 2>/dev/null || true' EXIT

echo ">> starting frontend dev server on :5173"
(cd web && npm install && npm run dev)
