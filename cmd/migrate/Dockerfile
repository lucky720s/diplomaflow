# syntax=docker/dockerfile:1.6

FROM golang:1.25.5-alpine3.20 AS builder
WORKDIR /app
ENV GOWORK=off

RUN apk --no-cache add git ca-certificates

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd/migrate ./cmd/migrate
COPY pkg ./pkg
COPY internal ./internal

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /bin/migrate ./cmd/migrate

FROM alpine:3.22.2
WORKDIR /app

RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/migrate /app/migrate

ENTRYPOINT ["/app/migrate"]
