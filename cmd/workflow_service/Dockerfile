FROM golang:1.25-alpine AS builder
WORKDIR /go/src/github.com/lucky720s/diplomaflow
copy go.mod go.sum ./
RUN go mod download

COPY cmd/workflow_service ./cmd/workflow_service
COPY internal/workflow ./internal/workflow
COPY pkg/protobuf ./pkg/protobuf

RUN CGO_ENABLED=0 GOOS=linux go build -o /workflow-service-bin ./cmd/workflow_service

FROM alpine:latest
RUN apk add --no-cache tzdata bash
COPY --from=builder /workflow-service-bin /workflow-service-bin
COPY deployments/workflow_service/boot.yaml /boot.yaml

ENTRYPOINT ["/workflow-service-bin"]
