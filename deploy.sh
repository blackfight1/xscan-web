#!/usr/bin/env bash

# XScan Web deployment helper
# Default mode for this project:
#   frontend in Docker + backend/xscan on host

set -euo pipefail

MODE="hybrid"
SERVICE_NAME="xscan-web"
INSTALL_SERVICE="false"

usage() {
  cat <<'EOF'
Usage:
  ./deploy.sh [--mode hybrid|host|docker] [--install-service] [--service-name NAME]

Modes:
  hybrid (default): frontend in Docker, backend/xscan on host
  host:             frontend static files + backend all on host
  docker:           both frontend/backend via docker compose

Options:
  --install-service    Create or update systemd service for backend and restart it
  --service-name NAME  systemd service name (default: xscan-web)
  -h, --help           Show this help
EOF
}

log() {
  printf '[*] %s\n' "$1"
}

warn() {
  printf '[!] %s\n' "$1"
}

die() {
  printf '[x] %s\n' "$1" >&2
  exit 1
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

require_cmd() {
  has_cmd "$1" || die "Missing required command: $1"
}

ensure_project_root() {
  [[ -d ./backend ]] || die "backend directory not found. Run from project root."
  [[ -d ./frontend ]] || die "frontend directory not found. Run from project root."
}

resolve_go_bin() {
  if has_cmd go; then
    GO_BIN="$(command -v go)"
    return 0
  fi
  if [[ -x /usr/local/go/bin/go ]]; then
    GO_BIN="/usr/local/go/bin/go"
    return 0
  fi
  die "Go not found. Install Go or add it to PATH."
}

compose_run() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
    return 0
  fi
  if has_cmd docker-compose; then
    docker-compose "$@"
    return 0
  fi
  die "docker compose not available."
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      MODE="${2:-}"
      shift 2
      ;;
    --install-service)
      INSTALL_SERVICE="true"
      shift
      ;;
    --service-name)
      SERVICE_NAME="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "Unknown argument: $1"
      ;;
  esac
done

if [[ "$MODE" != "hybrid" && "$MODE" != "host" && "$MODE" != "docker" ]]; then
  die "Invalid mode: $MODE"
fi

if [[ -z "$SERVICE_NAME" ]]; then
  die "Service name cannot be empty."
fi

PROJECT_DIR="$(pwd)"
GO_BIN=""

prepare_dirs() {
  log "Preparing directories"
  mkdir -p data results tools xscan static
}

prepare_xscan() {
  log "Checking xscan binary and assets"

  if [[ ! -f ./xscan/xscan ]]; then
    if [[ -d ./xscan_3.6.5_linux_amd64 ]]; then
      log "Found ./xscan_3.6.5_linux_amd64, copying into ./xscan"
      cp -r ./xscan_3.6.5_linux_amd64/* ./xscan/
    else
      die "Missing ./xscan/xscan. Put xscan binary/config.yaml/dict in ./xscan/"
    fi
  fi

  chmod +x ./xscan/xscan

  [[ -f ./xscan/config.yaml ]] || warn "Missing ./xscan/config.yaml"
  if [[ ! -e ./xscan/dict ]]; then
    warn "Missing ./xscan/dict"
  elif [[ -f ./xscan/dict ]]; then
    warn "./xscan/dict is a file, not a directory. Keep this only if your xscan package requires it."
  fi
}

prepare_tools() {
  log "Checking subfinder/httpx"

  for tool in subfinder httpx; do
    target="./tools/$tool"
    if [[ -x "$target" ]]; then
      log "Using local tool: $target"
      continue
    fi

    if has_cmd "$tool"; then
      cp -f "$(command -v "$tool")" "$target"
      chmod +x "$target"
      log "Copied $tool to $target"
    else
      warn "$tool not found. Pipeline fallback will be used."
    fi
  done
}

ensure_config() {
  log "Checking config.json"
  if [[ ! -f ./config.json ]]; then
    cp ./backend/config.json ./config.json
    warn "Generated ./config.json from backend template. Review auth_token and paths."
  fi

  if grep -Eq '"auth_token"[[:space:]]*:[[:space:]]*""' ./config.json; then
    warn "auth_token is empty in ./config.json. API will be publicly accessible."
  fi

  if ! grep -Eq '"xscan_path"[[:space:]]*:[[:space:]]*"./xscan/xscan"' ./config.json; then
    warn "xscan_path is not './xscan/xscan' in ./config.json. Ensure path is correct for current working directory."
  fi
}

build_backend_host() {
  resolve_go_bin
  log "Building backend binary with $GO_BIN"

  pushd ./backend >/dev/null
  if [[ -f ./go.sum ]]; then
    "$GO_BIN" mod download
  else
    "$GO_BIN" mod tidy
  fi
  CGO_ENABLED=1 "$GO_BIN" build -o ../xscan-web-server .
  popd >/dev/null

  chmod +x ./xscan-web-server
  log "Backend binary ready: ./xscan-web-server"
}

build_frontend_static() {
  require_cmd npm
  log "Building frontend static files"

  pushd ./frontend >/dev/null
  npm install
  npm run build
  popd >/dev/null

  rm -rf ./static
  cp -r ./frontend/dist ./static
  log "Static files synced to ./static"
}

write_systemd_service() {
  [[ "$INSTALL_SERVICE" == "true" ]] || return 0
  require_cmd systemctl

  log "Installing/updating systemd service: ${SERVICE_NAME}.service"
  cat >"/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=XScan Web Backend
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${PROJECT_DIR}
ExecStart=${PROJECT_DIR}/xscan-web-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable "${SERVICE_NAME}" >/dev/null
}

restart_backend_if_possible() {
  if has_cmd systemctl && systemctl list-unit-files | grep -q "^${SERVICE_NAME}\.service"; then
    log "Restarting backend service: ${SERVICE_NAME}"
    systemctl restart "${SERVICE_NAME}"
    systemctl is-active "${SERVICE_NAME}" >/dev/null || die "Service ${SERVICE_NAME} is not active."
    return 0
  fi

  warn "Service ${SERVICE_NAME}.service not found. Start backend manually:"
  warn "  cd ${PROJECT_DIR} && ./xscan-web-server"
}

deploy_frontend_hybrid() {
  require_cmd docker
  log "Deploying frontend container (hybrid mode)"

  if docker ps -a --format '{{.Names}}' | grep -qx 'xscan-web-frontend'; then
    docker rm -f xscan-web-frontend >/dev/null || true
  fi

  compose_run up -d --build frontend
}

deploy_full_docker() {
  require_cmd docker
  log "Deploying full stack with docker compose"

  if docker ps -a --format '{{.Names}}' | grep -qx 'xscan-web-frontend'; then
    docker rm -f xscan-web-frontend >/dev/null || true
  fi

  compose_run up -d --build
}

print_summary() {
  cat <<EOF

==========================================
Deploy complete
==========================================
Mode: ${MODE}
Project: ${PROJECT_DIR}
Backend service: ${SERVICE_NAME}
Frontend URL: http://<your-vps-ip>/
Health URL:   http://<your-vps-ip>/health
==========================================
EOF
}

main() {
  ensure_project_root
  prepare_dirs
  prepare_xscan
  prepare_tools
  ensure_config

  case "$MODE" in
    hybrid)
      build_backend_host
      write_systemd_service
      restart_backend_if_possible
      deploy_frontend_hybrid
      ;;
    host)
      build_backend_host
      build_frontend_static
      write_systemd_service
      restart_backend_if_possible
      ;;
    docker)
      deploy_full_docker
      ;;
  esac

  print_summary
}

main "$@"
