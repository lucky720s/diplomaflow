FROM golang:1.25-alpine AS builder
WORKDIR /go/src/github.com/lucky720s/diplomaflow
copy go.mod go.sum ./
RUN go mod download

COPY cmd/university_service ./cmd/university_service
COPY internal/university ./internal/university
COPY pkg/protobuf ./pkg/protobuf

RUN CGO_ENABLED=0 GOOS=linux go build -o /university-service-bin ./cmd/university_service

FROM alpine:latest
RUN apk add --no-cache tzdata bash
COPY --from=builder /university-service-bin /university-service-bin
COPY deployments/university_service/boot.yaml /boot.yaml

ENTRYPOINT ["/university-service-bin"]
