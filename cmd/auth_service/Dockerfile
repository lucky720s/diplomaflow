FROM golang:1.25-alpine AS builder
WORKDIR /app

ENV GOWORK=off

RUN go install github.com/google/wire/cmd/wire@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN wire gen ./internal/auth

RUN go generate ./...

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service ./cmd/auth_service

FROM alpine:3.22.2
WORKDIR /app
COPY --from=builder /app/cmd/auth_service/config.yaml /app/config.yaml

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
COPY --from=builder /bin/service /app/service
RUN chown -R appuser:appgroup /app
USER appuser

ENTRYPOINT ["/app/service"]
