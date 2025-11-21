FROM golang:1.25-alpine AS builder
WORKDIR /go/src/github.com/lucky720s/diplomaflow
copy go.mod go.sum ./
RUN go mod download

COPY cmd/auth_service ./cmd/auth_service
COPY internal/auth ./internal/auth
COPY pkg/protobuf ./pkg/protobuf

RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service-bin ./cmd/auth_service

FROM alpine:latest
RUN apk add --no-cache tzdata bash
COPY --from=builder /auth-service-bin /auth-service-bin
COPY deployments/auth_service/boot.yaml /boot.yaml
ENTRYPOINT ["/auth-service-bin"]