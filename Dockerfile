# --- Сборочный этап (Build Stage) ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Устанавливаем утилиты для работы entrypoint
RUN apk add --no-cache curl

# Устанавливаем migrate С ТЕГОМ ДЛЯ POSTGRES
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Компилируем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main ./cmd/server/main.go


# --- Финальный этап (Final Stage) ---
FROM alpine:latest

WORKDIR /app

# Копируем утилиту migrate из сборочного этапа
COPY --from=builder /go/bin/migrate /usr/local/bin/

# Копируем скомпилированный бинарник
COPY --from=builder /app/main .

# Копируем папку с миграциями
COPY ./migrations ./migrations

# Копируем и делаем исполняемым наш entrypoint скрипт
COPY ./docker/entrypoint.sh .
RUN chmod +x ./entrypoint.sh

EXPOSE 8080

# Указываем entrypoint
ENTRYPOINT ["./entrypoint.sh"]

# Указываем команду по умолчанию, которая будет передана в entrypoint
CMD ["./main"]
