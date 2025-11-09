FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /project-service-bin ./cmd/project_service
FROM alpine:latest
RUN apk add --no-cache tzdata bash
COPY --from=builder /project-service-bin /project-service-bin
COPY deployments/project_service/boot.yaml /boot.yaml
ENTRYPOINT ["/project-service-bin"]