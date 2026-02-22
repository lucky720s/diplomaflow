#!/bin/bash
set -euo pipefail

APP_DIR="${1:?APP_DIR required}"
PAT_TOKEN="${2:?PAT_TOKEN required}"
JWT_SECRET="${3:?JWT_SECRET required}"
POSTGRES_PASSWORD="${4:?POSTGRES_PASSWORD required}"
REPO="${5:?REPO required}"
BRANCH="${6:?BRANCH required}"
GRAFANA_ADMIN_PASSWORD="${7:?GRAFANA_ADMIN_PASSWORD is required}"

PROJECT_NAME="diplomaflow"
STATE_DIR="/opt/diplomaflow_state"
ENV_FILE="${STATE_DIR}/.env"
LOCK_FILE="${STATE_DIR}/deploy.lock"

export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_CLIENT_TIMEOUT=600
export COMPOSE_HTTP_TIMEOUT=600

dc() {
  docker compose -p "$PROJECT_NAME" --env-file "$ENV_FILE" "$@"
}

log() {
  echo "==> $*"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing required command: $1"; exit 1; }
}

log "Check required commands"
require_cmd git
require_cmd docker
require_cmd df
require_cmd curl

log "Ensure dirs"
mkdir -p "$APP_DIR" "$STATE_DIR"
chmod 700 "$STATE_DIR"

# Lock to avoid parallel deployments
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  echo "Another deployment is running (lock: $LOCK_FILE). Exiting."
  exit 1
fi

log "Sync repository"
cd "$APP_DIR"
git config --global --add safe.directory "$APP_DIR"

if [ -d ".git" ]; then
  git remote set-url origin "https://${PAT_TOKEN}@github.com/${REPO}.git"
  git fetch --depth=1 origin "$BRANCH"
  git reset --hard "origin/${BRANCH}"
  git clean -fd
else
  rm -rf "$APP_DIR"/* "$APP_DIR"/.[!.]* 2>/dev/null || true
  git clone --depth=1 -b "$BRANCH" "https://${PAT_TOKEN}@github.com/${REPO}.git" .
fi

log "Create .env"
umask 077
cat > "$ENV_FILE" <<EOF
METRICS_PORT=9091
GRAFANA_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD}
ENV=dev
GIN_MODE=release

JWT_SECRET=${JWT_SECRET}

POSTGRES_USER=diplomaflow
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=diplomaflow

DATABASE_DSN=host=127.0.0.1 user=diplomaflow password=${POSTGRES_PASSWORD} dbname=diplomaflow port=5433 sslmode=disable TimeZone=UTC
DATABASE_URL=postgres://diplomaflow:${POSTGRES_PASSWORD}@127.0.0.1:5433/diplomaflow?sslmode=disable

REDIS_ADDR=127.0.0.1:6380

ACCESS_TOKEN_TTL=24h
REFRESH_TOKEN_TTL=720h
EOF
chmod 600 "$ENV_FILE"

log "Validate compose"
dc config > /dev/null

log "Disk usage before"
df -h / | tail -1 || true
docker system df || true

log "Build images (cached)"
dc build --parallel

log "Start/ensure infra is up"
dc up -d main_postgres redis

log "Wait for postgres"
for i in $(seq 1 60); do
  if dc exec -T main_postgres pg_isready -h 127.0.0.1 -p 5433 -U diplomaflow >/dev/null 2>&1; then
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

log "Run migrations"
dc run --rm migrate

log "Update/start all services"
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

# ============================================================
# CLEANUP — агрессивная, но безопасная очистка ПОСЛЕ деплоя
# ============================================================
log "Cleanup: dangling images (old builds)"
docker image prune -f || true

log "Cleanup: build cache older than 16h (keep recent for speed)"
docker builder prune -af --filter "until=16h" || true

log "Cleanup: stopped containers"
docker container prune -f || true

log "Cleanup: unused networks"
docker network prune -f || true

# Удаляем ВСЕ образы без контейнеров, старше 16 часов
# Это убьёт старые версии сервисов, но оставит текущие (они привязаны к контейнерам)
log "Cleanup: unused images older than 16h"
docker image prune -af --filter "until=16h" || true

# DO NOT prune volumes — Postgres data, file_uploads, grafana, prometheus живут там!

log "Status"
dc ps || true

log "Disk usage after"
df -h / | tail -1 || true
docker system df || true

echo "Deployment completed!"
