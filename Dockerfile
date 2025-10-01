# --- Стадия сборки ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Устанавливаем зависимости для сборки и migrate
RUN apk add --no-cache git build-base

# Ставим migrate
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Копируем go.mod и go.sum и устанавливаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем бинарь
RUN go build -o main ./cmd/server
