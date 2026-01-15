FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk --no-cache add git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/auth_service ./cmd/auth_service

FROM alpine:3.22.2
RUN apk --no-cache add ca-certificates tzdata wget \
 && wget -qO /usr/local/bin/grpc_health_probe \
    https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/v0.4.26/grpc_health_probe-linux-amd64 \
 && chmod +x /usr/local/bin/grpc_health_probe

WORKDIR /app
COPY cmd/auth_service/config.yaml /app/config.yaml
COPY --from=build /out/auth_service /usr/local/bin/auth_service

EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/auth_service"]
