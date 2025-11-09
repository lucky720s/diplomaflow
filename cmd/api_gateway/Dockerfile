FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api-gateway-bin ./cmd/api_gateway

FROM alpine:latest
COPY --from=builder /api-gateway-bin /api-gateway-bin
ENTRYPOINT ["/api-gateway-bin"]