.DEFAULT_GOAL := help

ifeq ($(OS),Windows_NT)
    EXE := .exe
else
    EXE :=
endif

GOBIN     := $(shell go env GOPATH)/bin
PROTOC    := "C:/Tools/protoc/bin/protoc.exe"
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

.PHONY: help all deps tools proto wire generate test coverage build clean docker-up docker-down lint fmt vet migrate-up migrate-down migrate-create migrate-version migrate-force

help:
	@echo.
	@echo DiplomaFlow Build System
	@echo ========================
	@echo.
	@echo Dependencies:
	@echo   make deps          - Download Go dependencies
	@echo   make tools         - Install dev tools (protoc, wire, lint)
	@echo.
	@echo Code Generation:
	@echo   make proto         - Generate protobuf code
	@echo   make wire          - Generate Wire DI
	@echo   make generate      - Generate all (proto + wire)
	@echo.
	@echo Testing:
	@echo   make test          - Run all unit tests
	@echo   make test-auth     - Run auth service tests
	@echo   make test-file     - Run file service tests
	@echo   make test-form     - Run form service tests
	@echo   make test-notif    - Run notification tests
	@echo   make test-role     - Run role service tests
	@echo   make test-project  - Run project service tests
	@echo   make test-workflow - Run workflow service tests
	@echo   make coverage      - Tests with coverage report
	@echo.
	@echo Build:
	@echo   make build         - Build all services (Windows)
	@echo   make build-linux   - Build all services (Linux)
	@echo.
	@echo Docker:
	@echo   make docker-up     - Start Docker containers
	@echo   make docker-down   - Stop Docker containers
	@echo   make docker-build  - Build Docker images
	@echo   make docker-logs   - View Docker logs
	@echo   make docker-restart - Restart with rebuild
	@echo.
	@echo Code Quality:
	@echo   make lint          - Run golangci-lint
	@echo   make fmt           - Format code (go fmt)
	@echo   make vet           - Run go vet
	@echo.
	@echo Workflows:
	@echo   make pre-push      - Run all checks before push
	@echo   make ci            - CI pipeline
	@echo   make dev-setup     - Full dev environment setup
	@echo   make all           - deps + proto + wire + build
	@echo.
	@echo Cleanup:
	@echo   make clean         - Clean build artifacts
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
	@if not exist "$(PROTO_OUT)\admin\v1" mkdir "$(PROTO_OUT)\admin\v1"
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
	$(WIRE) gen ./internal/admin
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

# ==================== TESTING ====================
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

test-project:
	go test -v ./tests/unit/tests_project/...

test-workflow:
	go test -v ./tests/unit/tests_workflow/...

coverage:
	@if not exist "coverage" mkdir coverage
	go test -coverprofile=coverage/coverage.out -covermode=atomic ./tests/unit/...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	go tool cover -func=coverage/coverage.out

# ==================== BUILD ====================
build:
	@if not exist "bin" mkdir bin
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/api_gateway.exe ./cmd/api_gateway
	set CGO_ENABLED=0&& go build $(LDFLAGS) -o bin/admin_service.exe ./cmd/admin_service
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
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/admin_service ./cmd/admin_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/auth_service ./cmd/auth_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/file_service ./cmd/file_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/form_service ./cmd/form_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/notification_service ./cmd/notification_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/project_service ./cmd/project_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/role_service ./cmd/role_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/team_service ./cmd/team_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/university_service ./cmd/university_service
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build $(LDFLAGS) -o bin/linux/workflow_service ./cmd/workflow_service

# ==================== DOCKER ====================
docker-up:
	docker compose up -d main_postgres
	docker compose run --rm migrations
	docker compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

docker-logs:
	docker-compose logs -f

docker-restart:
	docker-compose down
	docker-compose up -d --build

docker-reset-db:
	docker compose down -v
	docker volume rm diplomaflow_diplomaflow_data || true
	docker compose up -d main_postgres
	sleep 5
	docker compose run --rm migrations

# ==================== CODE QUALITY ====================
lint:
	$(GOLINT) run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

# ==================== PRE-PUSH / CI ====================
pre-push: fmt vet lint test build
	@echo.
	@echo ========================================
	@echo   All checks passed! Ready to push.
	@echo ========================================

ci: deps generate lint test build

# ==================== SETUP ====================
dev-setup: deps tools proto wire
	@echo.
	@echo Development environment ready!

# ==================== CLEANUP ====================
clean:
	@if exist "bin" rmdir /s /q bin
	@if exist "coverage" rmdir /s /q coverage
	go clean -cache -testcache
migrate-up:
	go run ./cmd/migrate -cmd=up

migrate-down:
	go run ./cmd/migrate -cmd=down -steps=1

migrate-down-all:
	go run ./cmd/migrate -cmd=down

migrate-version:
	go run ./cmd/migrate -cmd=version

migrate-force:
	@read -p "Version: " version; \
	go run ./cmd/migrate -cmd=force -version=$$version

migrate-create:
	@read -p "Migration name: " name; \
	touch db/migrations/$$(date +%Y%m%d%H%M%S)_$$name.up.sql; \
	touch db/migrations/$$(date +%Y%m%d%H%M%S)_$$name.down.sql; \
	echo "Created migration: $$name"
