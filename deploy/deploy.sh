#!/bin/bash
set -euo pipefail

APP_DIR="$1"
PAT_TOKEN="$2"
JWT_SECRET="$3"
POSTGRES_PASSWORD="$4"
REPO="$5"
BRANCH="$6"

PROJECT_NAME="diplomaflow"
STATE_DIR="/opt/diplomaflow_state"
ENV_FILE="${STATE_DIR}/.env"

export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1
export DOCKER_CLIENT_TIMEOUT=600
export COMPOSE_HTTP_TIMEOUT=600

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
  git fetch origin "$BRANCH"
  git reset --hard "origin/${BRANCH}"
  git clean -fd
else
  rm -rf "$APP_DIR"/* "$APP_DIR"/.[!.]* 2>/dev/null || true
  git clone -b "$BRANCH" "https://${PAT_TOKEN}@github.com/${REPO}.git" .
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
KAFKA_BROKERS=127.0.0.1:29092
KAFKA_GROUP_ID=diplomaflow-group
KAFKA_KRAFT_CLUSTER_ID=MkU3OEVBNTcwNTJENDM2Qk
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
EOF
chmod 600 "$ENV_FILE"

echo "==> Validate compose"
dc config > /dev/null

echo "==> Stop old containers"
dc down --remove-orphans || true

echo "==> Build images"
dc build

echo "==> Start infra"
dc up -d main_postgres redis kafka

echo "==> Wait for postgres on port 5433"
for i in $(seq 1 60); do
  if pg_isready -h 127.0.0.1 -p 5433 -U diplomaflow 2>/dev/null; then
    echo "Postgres ready on port 5433"
    break
  fi
  echo "Waiting for postgres... ($i/60)"
  sleep 2
done

echo "==> Wait for redis on port 6380"
for i in $(seq 1 30); do
  if redis-cli -h 127.0.0.1 -p 6380 ping 2>/dev/null | grep -q PONG; then
    echo "Redis ready on port 6380"
    break
  fi
  sleep 1
done

echo "==> Wait for kafka"
sleep 30

echo "==> Run migrations"
dc run --rm migrate

echo "==> Start all services"
dc up -d --no-build

echo "==> Wait for gateway"
sleep 30
curl -fsS "http://127.0.0.1:8080/healthz" > /dev/null && echo "Gateway OK" || echo "Gateway not ready yet"

echo "==> Status"
dc ps

echo "✅ Deployment completed!"
