FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service ./cmd/workflow_service

FROM alpine:latest
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /bin/service /app/service
COPY deployments/workflow_service/boot.yaml /app/boot.yaml

ENTRYPOINT ["/app/service"]
