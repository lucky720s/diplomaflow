#!/bin/bash
set -euo pipefail

APP_DIR="$1"
PAT_TOKEN="$2"
JWT_SECRET="$3"
POSTGRES_PASSWORD="$4"
REPO="$5"
BRANCH="$6"
IMAGE_BASE="$7"     # e.g. ghcr.io/owner/repo
IMAGE_TAG="$8"      # e.g. dev
GHCR_USERNAME="${9:-}"
GHCR_TOKEN="${10:-}"

PROJECT_NAME="diplomaflow"
STATE_DIR="/opt/diplomaflow_state"
ENV_FILE="${STATE_DIR}/.env"

dc() {
  docker compose -p "$PROJECT_NAME" --env-file "$ENV_FILE" "$@"
}

echo "==> Ensure dirs"
mkdir -p "$APP_DIR" "$STATE_DIR"
chmod 700 "$STATE_DIR"
cd "$APP_DIR"

git config --global --add safe.directory "$APP_DIR"

echo "==> Sync repository"
if [ -d ".git" ]; then
  git remote set-url origin "https://${PAT_TOKEN}@github.com/${REPO}.git"
  git fetch origin "$BRANCH" --depth=1
  git reset --hard "origin/${BRANCH}"
  git clean -fd
else
  rm -rf "$APP_DIR"/* "$APP_DIR"/.[!.]* 2>/dev/null || true
  git clone -b "$BRANCH" --depth=1 "https://${PAT_TOKEN}@github.com/${REPO}.git" .
fi

echo "==> Create .env"
cat > "$ENV_FILE" <<EOF
ENV=dev
GIN_MODE=release

JWT_SECRET=${JWT_SECRET}

POSTGRES_USER=diplomaflow
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=diplomaflow

DATABASE_DSN=host=127.0.0.1 user=diplomaflow password=${POSTGRES_PASSWORD} dbname=diplomaflow port=5433 sslmode=disable TimeZone=UTC
DATABASE_URL=postgres://diplomaflow:${POSTGRES_PASSWORD}@127.0.0.1:5433/diplomaflow?sslmode=disable

REDIS_ADDR=127.0.0.1:6380

ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h

IMAGE_BASE="$(echo "$IMAGE_BASE" | tr '[:upper:]' '[:lower:]')"
IMAGE_TAG=${IMAGE_TAG}
EOF
chmod 600 "$ENV_FILE"

echo "==> Validate compose"
dc config > /dev/null

echo "==> Login to GHCR (if token provided)"
if [ -n "$GHCR_USERNAME" ] && [ -n "$GHCR_TOKEN" ]; then
  echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin >/dev/null
fi

echo "==> Pull images"
dc pull

echo "==> Start infra"
dc up -d --no-build main_postgres redis

echo "==> Wait for postgres (via docker exec)"
for i in $(seq 1 60); do
  if docker exec diplomaflow-main_postgres-1 pg_isready -h 127.0.0.1 -p 5433 -U diplomaflow >/dev/null 2>&1; then
    echo "Postgres ready!"
    break
  fi
  echo "Waiting for postgres... ($i/60)"
  sleep 2
done

echo "==> Wait for redis (via docker exec)"
for i in $(seq 1 30); do
  if docker exec diplomaflow-redis-1 redis-cli -h 127.0.0.1 -p 6380 ping 2>/dev/null | grep -q PONG; then
    echo "Redis ready!"
    break
  fi
  echo "Waiting for redis... ($i/30)"
  sleep 1
done

echo "==> Run migrations"
dc run --rm migrate

echo "==> Start/update all services (no build)"
dc up -d --no-build --remove-orphans

echo "==> Quick check"
sleep 10
curl -fsS "http://127.0.0.1:8080/healthz" >/dev/null && echo "Gateway OK" || echo "Gateway not ready yet"

echo "==> Cleanup old unused images (safe, doesn't touch volumes)"
docker image prune -af --filter "until=168h" || true
docker container prune -f || true
docker network prune -f || true

echo "==> Status"
dc ps
docker system df || true

echo "✅ Deployment completed!"
