FROM golang:1.25-alpine AS builder
WORKDIR /go/src/github.com/lucky720s/diplomaflow
copy go.mod go.sum ./
RUN go mod download

COPY cmd/team_service ./cmd/team_service
COPY internal/team ./internal/team
COPY pkg/protobuf ./pkg/protobuf

RUN CGO_ENABLED=0 GOOS=linux go build -o /team-service-bin ./cmd/team_service

FROM alpine:latest
RUN apk add --no-cache tzdata bash
COPY --from=builder /team-service-bin /team-service-bin
COPY deployments/team_service/boot.yaml /boot.yaml

ENTRYPOINT ["/team-service-bin"]
