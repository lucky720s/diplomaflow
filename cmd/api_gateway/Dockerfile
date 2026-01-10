# syntax=docker/dockerfile:1.6

FROM golang:1.25.5-alpine3.20 AS builder
WORKDIR /app
ENV GOWORK=off

RUN apk --no-cache add git ca-certificates

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY internal ./internal
COPY pkg ./pkg
COPY api ./api
COPY cmd/api_gateway ./cmd/api_gateway

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /bin/gateway ./cmd/api_gateway

FROM alpine:3.22.2
WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata \
 && addgroup -S appgroup \
 && adduser -S appuser -G appgroup

COPY cmd/api_gateway/config.yaml /app/config.yaml
COPY --from=builder /bin/gateway /app/gateway

RUN chown -R appuser:appgroup /app
USER appuser

ENTRYPOINT ["/app/gateway"]
