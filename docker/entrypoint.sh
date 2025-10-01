#!/bin/sh
set -e

echo "Running database migrations..."
migrate -database "$POSTGRES_URL" -path /app/migrations up

echo "Migrations applied successfully. Starting the application..."
exec "$@"
