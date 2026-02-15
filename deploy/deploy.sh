#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   deploy.sh <IMAGE_PREFIX> <IMAGE_TAG>
#
# Optional env:
#   GHCR_USERNAME, GHCR_TOKEN (if images are private)

IMAGE_PREFIX="${1:?IMAGE_PREFIX required (e.g. ghcr.io/owner/repo)}"
IMAGE_TAG="${2:?IMAGE_TAG required (e.g. git sha)}"

PROJECT_NAME="diplomaflow"
STATE_DIR="/opt/diplomaflow_state"
DEPLOY_DIR="/opt/diplomaflow_deploy"
ENV_FILE="${STATE_DIR}/.env"
LOCK_FILE="${STATE_DIR}/deploy.lock"

INFRA_FILE="${DEPLOY_DIR}/compose.infra.yml"
APP_FILE="${DEPLOY_DIR}/compose.app.deploy.yml"

export DOCKER_CLIENT_TIMEOUT=600
export COMPOSE_HTTP_TIMEOUT=600

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

log "Ensure dirs"
mkdir -p "$STATE_DIR" "$DEPLOY_DIR"
chmod 700 "$STATE_DIR"

# Lock
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  echo "Another deployment is running (lock: $LOCK_FILE). Exiting."
  exit 1
fi

log "Ensure required files exist"
test -f "$ENV_FILE" || { echo "Missing $ENV_FILE. Run bootstrap first."; exit 1; }
test -f "$INFRA_FILE" || { echo "Missing $INFRA_FILE"; exit 1; }
test -f "$APP_FILE" || { echo "Missing $APP_FILE"; exit 1; }

log "Optional: login to GHCR"
if [[ -n "${GHCR_USERNAME:-}" && -n "${GHCR_TOKEN:-}" ]]; then
  echo "$GHCR_TOKEN" | docker login ghcr.io -u "$GHCR_USERNAME" --password-stdin
fi

log "Validate compose"
dc config >/dev/null

log "Pull new images"
dc pull

log "Ensure infra is up"
dc up -d main_postgres redis

log "Run migrations"
dc run --rm migrate

log "Up services (no build)"
dc up -d --no-build --remove-orphans

log "Healthcheck gateway"
for i in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:8080/healthz" >/dev/null 2>&1; then
    echo "Gateway OK"
    break
  fi
  echo "Waiting for gateway... ($i/30)"
  sleep 2
done

log "Done"
dc ps || true
