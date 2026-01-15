FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk --no-cache add git ca-certificates
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api_gateway ./cmd/api_gateway

FROM alpine:3.22.2
RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app
COPY cmd/api_gateway/config.yaml /app/config.yaml
COPY --from=build /out/api_gateway /usr/local/bin/api_gateway

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api_gateway"]
