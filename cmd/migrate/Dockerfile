FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/migrate ./cmd/migrate

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /bin/migrate /app/migrate
RUN apk --no-cache add ca-certificates
ENTRYPOINT ["/app/migrate"]
