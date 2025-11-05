FROM golang:1.25-alpine AS builder
WORKDIR /app
copy go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service-bin ./cmd/auth_service

FROM alpine:latest
RUN apk add --no-cache tzdata bash
COPY --from=builder /auth-service-bin /auth-service-bin
COPY deployments/auth_service/boot.yaml /boot.yaml
ENTRYPOINT ["/auth-service-bin"]