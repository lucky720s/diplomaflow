#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   bootstrap.sh <ENV> <IMAGE_PREFIX> <IMAGE_TAG>
#
# Required env:
#   JWT_SECRET, POSTGRES_PASSWORD
#
# Optional env:
#   POSTGRES_USER, POSTGRES_DB, ACCESS_TOKEN_TTL, REFRESH_TOKEN_TTL
#   GHCR_USERNAME, GHCR_TOKEN (if images are private)

ENV_NAME="${1:?ENV required (e.g. dev)}"
IMAGE_PREFIX="${2:?IMAGE_PREFIX required (e.g. ghcr.io/owner/repo)}"
IMAGE_TAG="${3:?IMAGE_TAG required (e.g. git sha)}"

PROJECT_NAME="diplomaflow"
STATE_DIR="/opt/diplomaflow_state"
DEPLOY_DIR="/opt/diplomaflow_deploy"
ENV_FILE="${STATE_DIR}/.env"
LOCK_FILE="${STATE_DIR}/deploy.lock"

INFRA_FILE="${DEPLOY_DIR}/compose.infra.yml"
APP_FILE="${DEPLOY_DIR}/compose.app.deploy.yml"

log() { echo "==> $*"; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing required command: $1"; exit 1; }
}

dc() {
  IMAGE_PREFIX="$IMAGE_PREFIX" IMAGE_TAG="$IMAGE_TAG" \
  docker compose -p "$PROJECT_NAME" --env-file "$ENV_FILE" \
    -f "$INFRA_FILE" -f "$APP_FILE" "$@"
}

log "Check required commands"
require_cmd docker
require_cmd flock
require_cmd curl
require_cmd df

log "Ensure dirs"
mkdir -p "$STATE_DIR" "$DEPLOY_DIR"
chmod 700 "$STATE_DIR"

# Lock
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  echo "Another deployment is running (lock: $LOCK_FILE). Exiting."
  exit 1
fi

log "Ensure compose files exist on server"
test -f "$INFRA_FILE" || { echo "Missing $INFRA_FILE"; exit 1; }
test -f "$APP_FILE" || { echo "Missing $APP_FILE"; exit 1; }

log "Create .env ONLY if missing"
if [[ ! -f "$ENV_FILE" ]]; then
  : "${JWT_SECRET:?JWT_SECRET is required for bootstrap}"
  : "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required for bootstrap}"

  POSTGRES_USER="${POSTGRES_USER:-diplomaflow}"
  POSTGRES_DB="${POSTGRES_DB:-diplomaflow}"
  ACCESS_TOKEN_TTL="${ACCESS_TOKEN_TTL:-15m}"
  REFRESH_TOKEN_TTL="${REFRESH_TOKEN_TTL:-720h}"

  umask 077
  cat > "$ENV_FILE" <<EOF
ENV=${ENV_NAME}
GIN_MODE=release

JWT_SECRET=${JWT_SECRET}

POSTGRES_USER=${POSTGRES_USER}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=${POSTGRES_DB}

# host network
DATABASE_DSN=host=127.0.0.1 user=${POSTGRES_USER} password=${POSTGRES_PASSWORD} dbname=${POSTGRES_DB} port=5433 sslmode=disable TimeZone=UTC
DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@127.0.0.1:5433/${POSTGRES_DB}?sslmode=disable

REDIS_ADDR=127.0.0.1:6380

ACCESS_TOKEN_TTL=${ACCESS_TOKEN_TTL}
REFRESH_TOKEN_TTL=${REFRESH_TOKEN_TTL}
EOF
  chmod 600 "$ENV_FILE"
  log ".env created at $ENV_FILE"
else
  log ".env already exists, not overwriting: $ENV_FILE"
fi

log "Optional: login to GHCR"
if [[ -n "${GHCR_USERNAME:-}" && -n "${GHCR_TOKEN:-}" ]]; then
  echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin
fi

log "Validate compose"
dc config >/dev/null

log "Bring up infra (postgres/redis)"
dc up -d main_postgres redis

log "Wait for postgres"
for i in $(seq 1 60); do
  if dc exec -T main_postgres pg_isready -h 127.0.0.1 -p 5433 -U "${POSTGRES_USER:-diplomaflow}" >/dev/null 2>&1; then
    echo "Postgres ready!"
    break
  fi
  echo "Waiting for postgres... ($i/60)"
  sleep 2
done

log "Wait for redis"
for i in $(seq 1 30); do
  if dc exec -T redis redis-cli -h 127.0.0.1 -p 6380 ping 2>/dev/null | grep -q PONG; then
    echo "Redis ready!"
    break
  fi
  echo "Waiting for redis... ($i/30)"
  sleep 1
done

log "Pull app images"
dc pull

log "Run migrations"
dc run --rm migrate

log "Start/Update all services"
dc up -d --no-build --remove-orphans

log "Wait for gateway"
for i in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:8080/healthz" >/dev/null 2>&1; then
    echo "Gateway OK"
    break
  fi
  echo "Waiting for gateway... ($i/30)"
  sleep 2
done

log "Status"
dc ps || true

log "Disk usage"
df -h / | tail -1 || true
docker system df || true

log "Bootstrap completed"
