FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk --no-cache add git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/migrate ./cmd/migrate


FROM alpine:3.22.2

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /out/migrate /usr/local/bin/migrate

COPY db/migrations /app/db/migrations

ENTRYPOINT ["/usr/local/bin/migrate"]
