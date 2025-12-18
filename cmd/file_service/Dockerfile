FROM golang:1.23-alpine AS builder
WORKDIR /app
ENV GOWORK=off


COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service ./cmd/file_service

FROM alpine:3.22.2
WORKDIR /app

COPY --from=builder /app/cmd/file_service/config.yaml /app/config.yaml
COPY --from=builder /bin/service /app/service

RUN apk --no-cache add ca-certificates tzdata
RUN mkdir -p /app/uploads
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN chown -R appuser:appgroup /app

USER appuser
ENTRYPOINT ["/app/service"]
