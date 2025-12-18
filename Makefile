.DEFAULT_GOAL := help

ifeq ($(OS),Windows_NT)
    EXE := .exe
else
    EXE :=
endif

GOBIN     := $(shell go env GOPATH)/bin
PROTOC    := protoc$(EXE)
WIRE      := $(GOBIN)/wire$(EXE)
GOLINT    := $(GOBIN)/golangci-lint$(EXE)
VALIDATE  := $(GOBIN)/protoc-gen-validate$(EXE)

PROTO_DIR := api/proto
PROTO_OUT := pkg/protobuf
THIRD_PARTY := third_party

PROTO_FILES := $(shell go run tools/detect.go proto 2>NUL || echo auth/v1/auth.proto)
WIRE_PACKAGES := $(shell go run tools/detect.go wire 2>NUL || echo ./internal/auth)
SERVICES := $(shell go run tools/detect.go services 2>NUL || echo api_gateway)


VERSION    := $(shell git describe --tags --always --dirty 2>NUL || echo dev)
BUILD_TIME := $(shell echo %DATE% %TIME%)
LDFLAGS    := -ldflags "-s -w -X main.Version=$(VERSION)"

.PHONY: help all deps tools proto wire generate test coverage build clean docker-up docker-down lint

help:
	@echo.
	@echo DiplomaFlow Build System
	@echo ========================
	@echo.
	@echo   make deps      - Download dependencies
	@echo   make tools     - Install dev tools
	@echo   make proto     - Generate protobuf
	@echo   make wire      - Generate Wire DI
	@echo   make generate  - Generate all
	@echo   make test      - Run tests
	@echo   make coverage  - Tests with coverage
	@echo   make build     - Build services
	@echo   make clean     - Clean artifacts
	@echo   make docker-up - Start Docker
	@echo   make lint      - Run linter
	@echo.

all: deps proto wire build

deps:
	go mod download
	go mod tidy

tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/envoyproxy/protoc-gen-validate@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

proto:
	@if not exist "$(PROTO_OUT)\auth\v1" mkdir "$(PROTO_OUT)\auth\v1"
	@if not exist "$(PROTO_OUT)\role\v1" mkdir "$(PROTO_OUT)\role\v1"
	@if not exist "$(PROTO_OUT)\project\v1" mkdir "$(PROTO_OUT)\project\v1"
	@if not exist "$(PROTO_OUT)\team\v1" mkdir "$(PROTO_OUT)\team\v1"
	@if not exist "$(PROTO_OUT)\university\v1" mkdir "$(PROTO_OUT)\university\v1"
	@if not exist "$(PROTO_OUT)\workflow\v1" mkdir "$(PROTO_OUT)\workflow\v1"
	@if not exist "$(PROTO_OUT)\notification\v1" mkdir "$(PROTO_OUT)\notification\v1"
	@if not exist "$(PROTO_OUT)\file\v1" mkdir "$(PROTO_OUT)\file\v1"
	@if not exist "$(PROTO_OUT)\form\v1" mkdir "$(PROTO_OUT)\form\v1"
	$(PROTOC) --plugin=protoc-gen-validate=$(VALIDATE) --proto_path=$(PROTO_DIR) --proto_path=$(THIRD_PARTY) --go_out=$(PROTO_OUT) --go_opt=paths=source_relative --go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative --validate_out="lang=go:$(PROTO_OUT)" --validate_opt=paths=source_relative $(PROTO_FILES)

wire:
	$(WIRE) gen ./internal/auth
	$(WIRE) gen ./internal/file
	$(WIRE) gen ./internal/form
	$(WIRE) gen ./internal/gateway
	$(WIRE) gen ./internal/notification
	$(WIRE) gen ./internal/project
	$(WIRE) gen ./internal/role
	$(WIRE) gen ./internal/team
	$(WIRE) gen ./internal/university
	$(WIRE) gen ./internal/workflow

generate: proto wire

test:
	go test -v ./tests/unit/...

test-auth:
	go test -v ./tests/unit/tests_auth/...

test-file:
	go test -v ./tests/unit/tests_file/...

test-form:
	go test -v ./tests/unit/tests_form/...

test-notif:
	go test -v ./tests/unit/tests_notification/...

test-role:
	go test -v ./tests/unit/tests_role/...

coverage:
	@if not exist "coverage" mkdir coverage
	go test -coverprofile=coverage/coverage.out -covermode=atomic ./tests/unit/...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	go tool cover -func=coverage/coverage.out

build:
	@if not exist "bin" mkdir bin
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/api_gateway.exe ./cmd/api_gateway
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/auth_service.exe ./cmd/auth_service
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/file_service.exe ./cmd/file_service
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/form_service.exe ./cmd/form_service
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/notification_service.exe ./cmd/notification_service
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/project_service.exe ./cmd/project_service
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/role_service.exe ./cmd/role_service
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/team_service.exe ./cmd/team_service
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/university_service.exe ./cmd/university_service
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/workflow_service.exe ./cmd/workflow_service

build-linux:
	@if not exist "bin\linux" mkdir "bin\linux"
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/api_gateway ./cmd/api_gateway
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/auth_service ./cmd/auth_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/file_service ./cmd/file_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/form_service ./cmd/form_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/notification_service ./cmd/notification_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/project_service ./cmd/project_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/role_service ./cmd/role_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/team_service ./cmd/team_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/university_service ./cmd/university_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/workflow_service ./cmd/workflow_service

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

docker-logs:
	docker-compose logs -f

lint:
	$(GOLINT) run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	@if exist "bin" rmdir /s /q bin
	@if exist "coverage" rmdir /s /q coverage
	go clean -cache -testcache

dev-setup: deps tools proto wire

ci: deps generate lint test build
