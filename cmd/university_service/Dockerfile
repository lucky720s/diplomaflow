FROM golang:1.25-alpine AS builder
WORKDIR /app
copy go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /university-service-bin ./cmd/university_service

FROM alpine:latest
RUN apk add --no-cache tzdata bash
COPY --from=builder /university-service-bin /university-service-bin
COPY deployments/university_service/boot.yaml /boot.yaml

ENTRYPOINT ["/university-service-bin"]
