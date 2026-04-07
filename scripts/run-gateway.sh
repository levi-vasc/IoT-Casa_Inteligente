#!/usr/bin/env bash
# run-gateway.sh — compila e executa o gateway IoT nativamente no host Linux.
#
# Uso:
#   chmod +x scripts/run-gateway.sh
#   ./scripts/run-gateway.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GATEWAY_DIR="$REPO_ROOT/gateway"
BINARY="$GATEWAY_DIR/gateway"

# ── Verifica dependência ────────────────────────────────────────────────────────
if ! command -v go &>/dev/null; then
  echo "[ERROR] Go não encontrado. Instale em https://go.dev/dl/" >&2
  exit 1
fi

# ── Compila ────────────────────────────────────────────────────────────────────
echo "[INFO] Compilando gateway..."
(cd "$GATEWAY_DIR" && go build -o gateway .)

echo "[INFO] Gateway compilado em: $BINARY"

# ── Executa ────────────────────────────────────────────────────────────────────
echo "[INFO] Iniciando gateway (Ctrl+C para encerrar)..."
exec "$BINARY"
