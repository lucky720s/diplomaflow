#!/usr/bin/env bash
set -euo pipefail
umask 077

# ---- required env from CI ----
: "${PAT_TOKEN:?PAT_TOKEN is required}"
: "${JWT_SECRET:?JWT_SECRET is required}"
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
: "${REPO:?REPO is required (e.g. lucky720s/diplomaflow)}"
: "${BRANCH:?BRANCH is required (e.g. dev)}"

# ---- constants ----
APP_DIR="/opt/diplomaflow"

# IMPORTANT: keep state OUTSIDE repo dir so git clean can't delete it
STATE_DIR="/opt/diplomaflow_state"

PROJECT_NAME="diplomaflow"
PG_VOLUME="${PROJECT_NAME}_pg_data"

ENV_FILE="${STATE_DIR}/.env"
ENV_DESIRED="${STATE_DIR}/.env.desired"
LAST_SHA_FILE="${STATE_DIR}/last_deploy_sha"

export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

dc() {
  docker compose -p "$PROJECT_NAME" --env-file "$ENV_FILE" "$@"
}

log() {
  echo "==> $*"
}

ensure_dirs() {
  log "Ensure dirs"
  sudo mkdir -p "$APP_DIR"
  sudo chown "$USER":"$USER" "$APP_DIR"

  sudo mkdir -p "$STATE_DIR"
  sudo chown "$USER":"$USER" "$STATE_DIR"
  chmod 700 "$STATE_DIR"
}

sync_repo() {
  log "Sync repository"
  cd "$APP_DIR"

  if [ -d ".git" ]; then
    git remote set-url origin "https://${PAT_TOKEN}@github.com/${REPO}.git"
    git fetch origin "$BRANCH"
    git reset --hard "origin/${BRANCH}"
    git clean -fd
  else
    rm -rf "$APP_DIR"/*
    git clone -b "$BRANCH" "https://${PAT_TOKEN}@github.com/${REPO}.git" .
  fi
}

render_env() {
  log "Validate POSTGRES_PASSWORD is URL-safe"
  if ! echo "$POSTGRES_PASSWORD" | grep -Eq '^[A-Za-z0-9_.~-]+$'; then
    echo "ERROR: POSTGRES_PASSWORD must be URL-safe (A-Za-z0-9_.~-)"
    echo "Fix: change POSTGRES_PASSWORD_DEV or install python3/jq and URL-encode password."
    exit 1
  fi

  local pg_pass_urlenc="$POSTGRES_PASSWORD"

  log "Render desired env"
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
DATABASE_URL=postgres://diplomaflow:${pg_pass_urlenc}@main_postgres:5432/diplomaflow?sslmode=disable

REDIS_ADDR=redis:6379
KAFKA_BROKERS=kafka:29092
KAFKA_GROUP_ID=diplomaflow-group
KAFKA_KRAFT_CLUSTER_ID=MkU3OEVBNTcwNTJENDM2Qk

PORT_GATEWAY=8080
GATEWAY_BIND_ADDR=0.0.0.0
EOF
  chmod 600 "$ENV_DESIRED"

  log "Sync env if changed"
  if [ ! -f "$ENV_FILE" ] || ! cmp -s "$ENV_FILE" "$ENV_DESIRED"; then
    [ -f "$ENV_FILE" ] && cp "$ENV_FILE" "${STATE_DIR}/.env.bak.$(date +%Y%m%d_%H%M%S)"
    mv "$ENV_DESIRED" "$ENV_FILE"
    chmod 600 "$ENV_FILE"
  else
    rm -f "$ENV_DESIRED"
    log "Env unchanged"
  fi
}

validate_compose() {
  log "Validate compose config"
  dc config >/dev/null
}

reset_postgres() {
  log "Reset Postgres volume (dev): ${PG_VOLUME}"
  dc down --remove-orphans || true
  docker volume rm -f "$PG_VOLUME" || true
}

wait_postgres_healthy() {
  log "Wait for Postgres healthy"
  for _ in $(seq 1 90); do
    local cid
    cid="$(dc ps -q main_postgres 2>/dev/null || true)"
    if [ -n "$cid" ]; then
      local st
      st="$(docker inspect -f '{{.State.Health.Status}}' "$cid" 2>/dev/null || true)"
      if [ "$st" = "healthy" ]; then
        return 0
      fi
    fi
    sleep 1
  done

  local cid
  cid="$(dc ps -q main_postgres 2>/dev/null || true)"
  echo "Postgres not healthy"
  [ -n "$cid" ] && docker logs --tail=200 "$cid" || true
  return 1
}

decide_build() {
  log "Decide whether we need to build images (smart)"
  NEW_SHA="$(git rev-parse HEAD)"
  PREV_SHA=""
  if [ -f "$LAST_SHA_FILE" ]; then
    PREV_SHA="$(cat "$LAST_SHA_FILE" 2>/dev/null || true)"
  fi

  need_build=0
  need_build_migrate=0

  if [ -z "$PREV_SHA" ] || ! git cat-file -e "${PREV_SHA}^{commit}" 2>/dev/null; then
    log "No previous deploy SHA found. Will build everything once."
    need_build=1
    need_build_migrate=1
    return 0
  fi

  changed="$(git diff --name-only "${PREV_SHA}..${NEW_SHA}" || true)"
  log "Changed files since last deploy:"
  echo "$changed" | sed '/^$/d' || true

  # Build needed if Go code / docker / compose / per-service config changed
  if echo "$changed" | grep -Eq '^(go\.mod|go\.sum|docker-compose\.yml|cmd/|internal/|pkg/|api/|proto/|\.dockerignore|Dockerfile)'; then
    need_build=1
  fi
  if echo "$changed" | grep -Eq '^cmd/.*/config\.yaml$'; then
    need_build=1
  fi

  # migrate image must be rebuilt if migrations changed (it copies db/migrations into the image)
  if echo "$changed" | grep -Eq '^(db/migrations/|cmd/migrate/|cmd/migrate/Dockerfile)$'; then
    need_build_migrate=1
  fi
}

start_infra() {
  log "Start infra"
  dc up -d main_postgres redis kafka
  wait_postgres_healthy
}

build_images() {
  log "Build images (smart)"
  if [ "$need_build" -eq 1 ]; then
    dc build --parallel
  else
    log "Skip build (no build-related changes)"
    if [ "$need_build_migrate" -eq 1 ]; then
      log "Rebuild migrate image (migrations changed)"
      dc build migrate
    fi
  fi
}

run_migrations_with_autofix() {
  log "Run migrations (auto-fix dev: retry once on known errors)"
  set +e
  mig_out="$(dc run --rm migrate 2>&1)"
  mig_rc=$?
  set -e

  if [ "$mig_rc" -eq 0 ]; then
    return 0
  fi

  echo "$mig_out"
  log "Migrations failed. main_postgres logs:"
  dc logs --tail=200 main_postgres || true

  if echo "$mig_out" | grep -Eq 'password authentication failed|Dirty database|dirty database'; then
    log "Auto-fix: reset Postgres volume and retry migrations once"
    reset_postgres
    start_infra
    dc build migrate
    dc run --rm migrate
    return 0
  fi

  echo "Migrations failed for unknown reason; not auto-resetting."
  return 1
}

start_app() {
  log "Start app services (no-build)"
  dc up -d --remove-orphans --no-build
}

healthcheck() {
  log "Healthcheck gateway"
  sleep 15
  curl -fsS "http://localhost:8080/healthz" >/dev/null

  log "Status"
  dc ps

  if dc ps | grep -q unhealthy; then
    echo "Some services are unhealthy. Last logs:"
    dc logs --tail=200
    exit 1
  fi
}

mark_success() {
  log "Mark successful deploy SHA"
  echo "$NEW_SHA" > "$LAST_SHA_FILE"
  chmod 600 "$LAST_SHA_FILE"
}

main() {
  ensure_dirs
  sync_repo

  # important: ensure state dir exists even if someone cleaned it manually
  mkdir -p "$STATE_DIR"
  chmod 700 "$STATE_DIR"

  render_env
  validate_compose

  decide_build
  start_infra
  build_images
  run_migrations_with_autofix
  start_app
  healthcheck
  mark_success

  log "Deploy completed successfully"
}

main "$@"
