#!/usr/bin/env bash
set -euo pipefail
umask 077

: "${PAT_TOKEN:?}"
: "${JWT_SECRET:?}"
: "${POSTGRES_PASSWORD:?}"
: "${REPO:?}"
: "${BRANCH:?}"

APP_DIR="/opt/diplomaflow"
STATE_DIR="${APP_DIR}/.state"

PROJECT_NAME="diplomaflow"
PG_VOLUME="${PROJECT_NAME}_pg_data"

ENV_FILE="${STATE_DIR}/.env"
ENV_DESIRED="${STATE_DIR}/.env.desired"
LAST_SHA_FILE="${STATE_DIR}/last_deploy_sha"

export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

dc() { docker compose -p "$PROJECT_NAME" --env-file "$ENV_FILE" "$@"; }

echo "==> Ensure app dir"
sudo mkdir -p "$APP_DIR"
sudo chown "$USER":"$USER" "$APP_DIR"
mkdir -p "$STATE_DIR"
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

echo "==> Ensure POSTGRES_PASSWORD is URL-safe"
if ! echo "$POSTGRES_PASSWORD" | grep -Eq '^[A-Za-z0-9_.~-]+$'; then
  echo "ERROR: POSTGRES_PASSWORD_DEV must be URL-safe (A-Za-z0-9_.~-)"
  exit 1
fi
PG_PASS_URLENC="$POSTGRES_PASSWORD"

echo "==> Render env"
cat > "$ENV_DESIRED" <<EOF
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
GATEWAY_BIND_ADDR=0.0.0.0
EOF
chmod 600 "$ENV_DESIRED"

if [ ! -f "$ENV_FILE" ] || ! cmp -s "$ENV_FILE" "$ENV_DESIRED"; then
  [ -f "$ENV_FILE" ] && cp "$ENV_FILE" "${STATE_DIR}/.env.bak.$(date +%Y%m%d_%H%M%S)"
  mv "$ENV_DESIRED" "$ENV_FILE"
else
  rm -f "$ENV_DESIRED"
fi

dc config >/dev/null

reset_postgres() {
  dc down --remove-orphans || true
  docker volume rm -f "$PG_VOLUME" || true
}

wait_postgres_healthy() {
  for i in $(seq 1 90); do
    cid="$(dc ps -q main_postgres 2>/dev/null || true)"
    if [ -n "$cid" ]; then
      st="$(docker inspect -f '{{.State.Health.Status}}' "$cid" 2>/dev/null || true)"
      [ "$st" = "healthy" ] && return 0
    fi
    sleep 1
  done
  return 1
}

NEW_SHA="$(git rev-parse HEAD)"
PREV_SHA="$(cat "$LAST_SHA_FILE" 2>/dev/null || true)"

need_build=0
need_build_migrate=0

if [ -z "$PREV_SHA" ] || ! git cat-file -e "${PREV_SHA}^{commit}" 2>/dev/null; then
  need_build=1
  need_build_migrate=1
else
  changed="$(git diff --name-only "${PREV_SHA}..${NEW_SHA}" || true)"
  if echo "$changed" | grep -Eq '^(go\.mod|go\.sum|docker-compose\.yml|cmd/|internal/|pkg/|api/|proto/|\.dockerignore|Dockerfile|cmd/.*/config\.yaml$)'; then
    need_build=1
  fi
  if echo "$changed" | grep -Eq '^(db/migrations/|cmd/migrate/|cmd/migrate/Dockerfile)$'; then
    need_build_migrate=1
  fi
fi

dc up -d main_postgres redis kafka
wait_postgres_healthy

if [ "$need_build" -eq 1 ]; then
  dc build --parallel
else
  [ "$need_build_migrate" -eq 1 ] && dc build migrate || true
fi

set +e
mig_out="$(dc run --rm migrate 2>&1)"
mig_rc=$?
set -e
if [ "$mig_rc" -ne 0 ]; then
  echo "$mig_out"
  if echo "$mig_out" | grep -Eq 'password authentication failed|Dirty database|dirty database'; then
    reset_postgres
    dc up -d main_postgres redis kafka
    wait_postgres_healthy
    dc build migrate
    dc run --rm migrate
  else
    exit 1
  fi
fi

dc up -d --remove-orphans --no-build
sleep 10
curl -fsS "http://localhost:8080/healthz" >/dev/null

dc ps
dc ps | grep -q unhealthy && { dc logs --tail=200; exit 1; }

echo "$NEW_SHA" > "$LAST_SHA_FILE"
chmod 600 "$LAST_SHA_FILE"
