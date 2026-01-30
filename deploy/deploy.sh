#!/bin/bash
set -euo pipefail

# ============================================
# Arguments
# ============================================
APP_DIR="$1"
STATE_DIR="$2"
PAT_TOKEN="$3"
JWT_SECRET="$4"
POSTGRES_PASSWORD="$5"
REPO="$6"
BRANCH="$7"

PROJECT_NAME="diplomaflow"
PG_VOLUME="${PROJECT_NAME}_pg_data"
ENV_FILE="${STATE_DIR}/.env"
LAST_SHA_FILE="${STATE_DIR}/last_deploy_sha"

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

echo "==> Sync repository"
if [ -d ".git" ]; then
  git remote set-url origin "https://${PAT_TOKEN}@github.com/${REPO}.git"
  git fetch origin "$BRANCH"
  git reset --hard "origin/${BRANCH}"
  git clean -fd
else
  rm -rf "$APP_DIR"/*
  git clone -b "$BRANCH" "https://${PAT_TOKEN}@github.com/${REPO}.git" .
fi

echo "==> Validate required secrets"
: "${JWT_SECRET:?Missing JWT_SECRET}"
: "${POSTGRES_PASSWORD:?Missing POSTGRES_PASSWORD}"

echo "==> Ensure POSTGRES_PASSWORD is URL-safe"
if echo "$POSTGRES_PASSWORD" | grep -Eq '^[A-Za-z0-9_.~-]+$'; then
  PG_PASS_URLENC="$POSTGRES_PASSWORD"
else
  echo "ERROR: POSTGRES_PASSWORD has special chars. Use only A-Za-z0-9_.~-"
  exit 1
fi

echo "==> Render env file"
cat > "$ENV_FILE" <<EOF
# managed-by-cd: true
ENV_FILE_VERSION=2

ENV=dev
GIN_MODE=release

JWT_SECRET=${JWT_SECRET}

POSTGRES_USER=diplomaflow
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=diplomaflow
POSTGRES_PORT=5432
POSTGRES_HOST_PORT=5433

DATABASE_DSN=host=main_postgres user=diplomaflow password=${POSTGRES_PASSWORD} dbname=diplomaflow port=5432 sslmode=disable TimeZone=UTC
DATABASE_URL=postgres://diplomaflow:${PG_PASS_URLENC}@main_postgres:5432/diplomaflow?sslmode=disable

REDIS_ADDR=redis:6379

KAFKA_BROKERS=kafka:29092
KAFKA_GROUP_ID=diplomaflow-group
KAFKA_KRAFT_CLUSTER_ID=MkU3OEVBNTcwNTJENDM2Qk

PORT_GATEWAY=8080
PORT_UNIVERSITY=8081
PORT_AUTH=8082
PORT_PROJECT=8083
PORT_TEAM=8084
PORT_WORKFLOW=8085
PORT_ROLE=8086
PORT_NOTIFICATION=8087
PORT_FILE=8088
PORT_FORM=8089
PORT_ADMIN=8090
PORT_TASK=8091

GATEWAY_BIND_ADDR=0.0.0.0

SERVICES_AUTH_ADDR=auth_service:8082
SERVICES_UNIVERSITY_ADDR=university_service:8081
SERVICES_PROJECT_ADDR=project_service:8083
SERVICES_TEAM_ADDR=team_service:8084
SERVICES_WORKFLOW_ADDR=workflow_service:8085
SERVICES_ROLE_ADDR=role_service:8086
SERVICES_NOTIFICATION_ADDR=notification_service:8087
SERVICES_FILE_ADDR=file_service:8088
SERVICES_FORM_ADDR=form_service:8089
SERVICES_ADMIN_ADDR=admin_service:8090
SERVICES_TASK_ADDR=task_service:8091

ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
EOF
chmod 600 "$ENV_FILE"

echo "==> Validate compose config"
dc config >/dev/null

wait_postgres_healthy() {
  echo "==> Wait for Postgres healthy"
  for i in $(seq 1 90); do
    cid="$(dc ps -q main_postgres 2>/dev/null || true)"
    if [ -n "$cid" ]; then
      st="$(docker inspect -f '{{.State.Health.Status}}' "$cid" 2>/dev/null || true)"
      [ "$st" = "healthy" ] && return 0
    fi
    sleep 1
  done
  echo "Postgres not healthy"
  cid="$(dc ps -q main_postgres 2>/dev/null || true)"
  [ -n "$cid" ] && docker logs --tail=200 "$cid" || true
  return 1
}

echo "==> Smart build detection"
NEW_SHA="$(git rev-parse HEAD)"
PREV_SHA=""
if [ -f "$LAST_SHA_FILE" ]; then
  PREV_SHA="$(cat "$LAST_SHA_FILE" 2>/dev/null || true)"
fi

need_build=0
need_build_migrate=0

if [ -z "$PREV_SHA" ] || ! git cat-file -e "${PREV_SHA}^{commit}" 2>/dev/null; then
  echo "==> No previous deploy SHA found. Will build everything."
  need_build=1
  need_build_migrate=1
else
  changed="$(git diff --name-only "${PREV_SHA}..${NEW_SHA}" || true)"
  echo "==> Changed files since last deploy:"
  echo "$changed" | sed '/^$/d' || true

  if echo "$changed" | grep -Eq '^(go\.mod|go\.sum|docker-compose\.yml|cmd/|internal/|pkg/|api/|proto/|\.dockerignore|Dockerfile)'; then
    need_build=1
  fi
  if echo "$changed" | grep -Eq '^(db/migrations/|cmd/migrate/)'; then
    need_build_migrate=1
  fi
fi

echo "==> Start infra"
dc up -d main_postgres redis kafka
wait_postgres_healthy

echo "==> Build images (smart)"
if [ "$need_build" -eq 1 ]; then
  dc build
else
  echo "==> Skip build (no build-related changes)"
  if [ "$need_build_migrate" -eq 1 ]; then
    echo "==> Rebuild migrate image"
    dc build migrate
  fi
fi

echo "==> Run migrations"
dc run --rm migrate

echo "==> Start app services"
dc up -d --remove-orphans --no-build

echo "==> Healthcheck gateway"
sleep 15
curl -fsS "http://localhost:8080/healthz" >/dev/null || {
  echo "Gateway healthcheck failed!"
  dc logs --tail=50 api_gateway
  exit 1
}

echo "==> Status"
dc ps

echo "==> Check for unhealthy services"
if dc ps | grep -q unhealthy; then
  echo "Some services are unhealthy:"
  dc logs --tail=100
  exit 1
fi

echo "==> Mark successful deploy SHA"
echo "$NEW_SHA" > "$LAST_SHA_FILE"
chmod 600 "$LAST_SHA_FILE"

echo "✅ Deployment completed!"
