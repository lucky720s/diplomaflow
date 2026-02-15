#!/usr/bin/env bash
set -euo pipefail

# Safe cleanup to run by cron (e.g. nightly)
echo "==> docker system df"
docker system df || true

echo "==> prune old images/containers/networks (keep volumes)"
docker image prune -af --filter "until=168h" || true
docker container prune -f || true
docker network prune -f || true

# builder prune лучше делать реже, иначе убьёте build-cache в CI runner'ах это не влияет,
# но на сервере тоже может быть полезно оставить. Если надо — включите раз в неделю:
# docker builder prune -af --filter "until=336h" || true

echo "==> done"
docker system df || true
