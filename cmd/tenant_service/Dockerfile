FROM golang:1.25-alpine AS builder
WORKDIR /app
copy go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /tenant-service-bin ./cmd/tenant_service

FROM alpine:latest
RUN apk add --no-cache tzdata bash
COPY --from=builder /tenant-service-bin /tenant-service-bin
COPY deployments/tenant_service/boot.yaml /boot.yaml

ENTRYPOINT ["/tenant-service-bin"]