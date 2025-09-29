#!/bin/sh

# Прерываем выполнение скрипта, если любая команда завершится с ошибкой
set -e

echo "Running database migrations..."
# Запускаем миграции. Путь к файлам и строка подключения берутся из Dockerfile/docker-compose
migrate -database "$POSTGRES_URL" -path /app/migrations up

echo "Migrations applied successfully. Starting the application..."

# Эта команда выполняет то, что было передано в CMD Dockerfile.
# В нашем случае, она запустит ваше Go-приложение.
exec "$@"
