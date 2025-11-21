FROM golang:1.25-alpine AS builder
WORKDIR /go/src/github.com/lucky720s/diplomaflow
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/role_service ./cmd/role_service
COPY internal/role ./internal/role
COPY pkg/protobuf ./pkg/protobuf

RUN CGO_ENABLED=0 GOOS=linux go build -o /role-service-bin ./cmd/role_service

FROM alpine:latest
RUN apk add --no-cache tzdata bash
COPY --from=builder /role-service-bin /role-service-bin
COPY deployments/role_service/boot.yaml /boot.yaml

ENTRYPOINT ["/role-service-bin"]
