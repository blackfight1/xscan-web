#!/usr/bin/env bash
# XScan Web deployment helper
# Default mode fits this project now:
#   frontend in Docker + backend/xscan on host

set -euo pipefail

MODE="hybrid"

usage() {
  cat <<'EOF'
Usage:
  ./deploy.sh [--mode hybrid|host|docker]

Modes:
  hybrid (default): frontend in Docker, backend/xscan on host
  host:             frontend static files + backend all on host
  docker:           both frontend/backend via docker compose
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      MODE="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "[!] Unknown argument: $1"
      usage
      exit 1
      ;;
  esac
done

if [[ "$MODE" != "hybrid" && "$MODE" != "host" && "$MODE" != "docker" ]]; then
  echo "[!] Invalid mode: $MODE"
  usage
  exit 1
fi

echo "=========================================="
echo "  XScan Web Deploy Script"
echo "  Mode: $MODE"
echo "=========================================="

prepare_dirs() {
  echo "[1/5] Preparing directories..."
  mkdir -p data results tools xscan
}

prepare_xscan() {
  echo "[2/5] Checking xscan..."
  if [[ ! -f "./xscan/xscan" ]]; then
    if [[ -d "./xscan_3.6.5_linux_amd64" ]]; then
      echo "[*] Found ./xscan_3.6.5_linux_amd64, copying into ./xscan ..."
      cp -r ./xscan_3.6.5_linux_amd64/* ./xscan/
    else
      echo "[!] Missing ./xscan/xscan"
      echo "    Put xscan binary/config.yaml/dict in ./xscan/"
      exit 1
    fi
  fi

  chmod +x ./xscan/xscan

  if [[ ! -f "./xscan/config.yaml" ]]; then
    echo "[!] Warning: ./xscan/config.yaml not found"
  fi
  if [[ ! -e "./xscan/dict" ]]; then
    echo "[!] Warning: ./xscan/dict not found"
  fi
}

prepare_tools() {
  echo "[3/5] Checking tools..."
  if command -v subfinder >/dev/null 2>&1; then
    cp "$(command -v subfinder)" ./tools/ 2>/dev/null || true
    echo "[✓] subfinder found"
  else
    echo "[!] subfinder not found (pipeline will fallback to root domain)"
  fi

  if command -v httpx >/dev/null 2>&1; then
    cp "$(command -v httpx)" ./tools/ 2>/dev/null || true
    echo "[✓] httpx found"
  else
    echo "[!] httpx not found (pipeline will fallback to https://domain)"
  fi
}

ensure_config() {
  echo "[4/5] Checking config..."
  if [[ ! -f "./config.json" ]]; then
    cp ./backend/config.json ./config.json
    echo "[!] Generated ./config.json from template, please update auth_token"
  fi
}

build_backend_host() {
  echo "[5/5] Building backend binary..."
  if ! command -v go >/dev/null 2>&1; then
    echo "[!] Go is required for host/hybrid mode build"
    exit 1
  fi
  (
    cd backend
    go mod tidy
    CGO_ENABLED=1 go build -o ../xscan-web-server .
  )
  echo "[✓] Built ./xscan-web-server"
}

build_frontend_static() {
  echo "[*] Building frontend static files (host mode)..."
  if ! command -v npm >/dev/null 2>&1; then
    echo "[!] npm is required for host mode"
    exit 1
  fi
  (
    cd frontend
    npm install
    npm run build
  )
  rm -rf static
  cp -r frontend/dist static
  echo "[✓] Frontend static files synced to ./static"
}

print_next_steps_hybrid() {
  cat <<'EOF'

==========================================
Deploy completed (hybrid mode)
==========================================
Next:
  1) Start backend on host:
       ./xscan-web-server
  2) Start frontend container:
       docker compose up -d frontend

Notes:
  - frontend will proxy /api to host.docker.internal:8080
  - ensure backend is listening on 0.0.0.0:8080
  - update auth_token in ./config.json
==========================================
EOF
}

print_next_steps_host() {
  cat <<'EOF'

==========================================
Deploy completed (host mode)
==========================================
Start:
  ./xscan-web-server
==========================================
EOF
}

print_next_steps_docker() {
  cat <<'EOF'

==========================================
Deploy completed (docker mode prep)
==========================================
Run:
  docker compose up -d --build
==========================================
EOF
}

prepare_dirs
prepare_xscan
prepare_tools
ensure_config

case "$MODE" in
  hybrid)
    build_backend_host
    print_next_steps_hybrid
    ;;
  host)
    build_backend_host
    build_frontend_static
    print_next_steps_host
    ;;
  docker)
    print_next_steps_docker
    ;;
esac

