.DEFAULT_GOAL := help

# ==================== OS DETECTION ====================

ifeq ($(OS),Windows_NT)
	SHELL := cmd.exe
	.SHELLFLAGS := /C
	EXE := .exe
	NULL := NUL
	MKDIR = if not exist "$(1)" mkdir "$(1)"
	RMDIR = if exist "$(1)" rmdir /s /q "$(1)"
	GO_BUILD_ENV = set CGO_ENABLED=0&&
	CROSS_LINUX_ENV = set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&&
	SLEEP5 = timeout /t 5 /nobreak >NUL
	PROTOC ?= C:/Tools/protoc/bin/protoc.exe
	BUILD_TIME := $(shell powershell -NoProfile -Command "Get-Date -Format o" 2>NUL || echo unknown)
else
	EXE :=
	NULL := /dev/null
	MKDIR = mkdir -p "$(1)"
	RMDIR = rm -rf "$(1)"
	GO_BUILD_ENV = CGO_ENABLED=0
	CROSS_LINUX_ENV = CGO_ENABLED=0 GOOS=linux GOARCH=amd64
	SLEEP5 = sleep 5
	PROTOC ?= protoc
	BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo unknown)
endif

# ==================== VARIABLES ====================

GOBIN     := $(shell go env GOPATH)/bin
WIRE      := $(GOBIN)/wire$(EXE)
GOLINT    := $(GOBIN)/golangci-lint$(EXE)
VALIDATE  := $(GOBIN)/protoc-gen-validate$(EXE)

PROTO_DIR := api/proto
PROTO_OUT := pkg/protobuf
THIRD_PARTY := third_party

PROTO_FILES := $(shell go run tools/detect.go proto 2>$(NULL) || echo auth/v1/auth.proto)
WIRE_PACKAGES := $(shell go run tools/detect.go wire 2>$(NULL) || echo ./internal/auth)
SERVICES := $(shell go run tools/detect.go services 2>$(NULL) || echo api_gateway)

VERSION    := $(shell git describe --tags --always --dirty 2>$(NULL) || echo dev)
LDFLAGS    := -ldflags "-s -w -X main.Version=$(VERSION)"

.PHONY: help all deps tools proto wire generate \
	test test-auth test-file test-form test-notif test-project test-workflow coverage \
	build build-linux clean docker-up docker-down docker-build docker-logs docker-restart docker-reset-db \
	lint fmt vet pre-push ci dev-setup \
	migrate-up migrate-down migrate-down-all migrate-create migrate-version migrate-force

# ==================== HELP ====================

help:
	@echo DiplomaFlow Build System
	@echo ========================
	@echo.
	@echo Dependencies:
	@echo   make deps           - Download Go dependencies
	@echo   make tools          - Install dev tools
	@echo.
	@echo Code Generation:
	@echo   make proto          - Generate protobuf code
	@echo   make wire           - Generate Wire DI
	@echo   make generate       - Generate all
	@echo.
	@echo Testing:
	@echo   make test           - Run all unit tests
	@echo   make test-auth      - Run auth service tests
	@echo   make test-file      - Run file service tests
	@echo   make test-form      - Run form service tests
	@echo   make test-notif     - Run notification tests
	@echo   make test-project   - Run project service tests
	@echo   make test-workflow  - Run workflow service tests
	@echo   make coverage       - Tests with coverage report
	@echo.
	@echo Build:
	@echo   make build          - Build all services for current OS
	@echo   make build-linux    - Build all services for Linux amd64
	@echo.
	@echo Docker:
	@echo   make docker-up      - Start Docker containers
	@echo   make docker-down    - Stop Docker containers
	@echo   make docker-build   - Build Docker images
	@echo   make docker-logs    - View Docker logs
	@echo   make docker-restart - Restart with rebuild
	@echo   make docker-reset-db - Reset database volume
	@echo.
	@echo Code Quality:
	@echo   make lint           - Run golangci-lint
	@echo   make fmt            - Format code
	@echo   make vet            - Run go vet
	@echo.
	@echo Workflows:
	@echo   make pre-push       - Run all checks before push
	@echo   make ci             - CI pipeline
	@echo   make dev-setup      - Full dev environment setup
	@echo   make all            - deps + proto + wire + build
	@echo.
	@echo Cleanup:
	@echo   make clean          - Clean build artifacts
	@echo.

# ==================== MAIN ====================

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

# ==================== CODE GENERATION ====================

proto:
	$(call MKDIR,$(PROTO_OUT)/admin/v1)
	$(call MKDIR,$(PROTO_OUT)/auth/v1)
	$(call MKDIR,$(PROTO_OUT)/project/v1)
	$(call MKDIR,$(PROTO_OUT)/team/v1)
	$(call MKDIR,$(PROTO_OUT)/university/v1)
	$(call MKDIR,$(PROTO_OUT)/workflow/v1)
	$(call MKDIR,$(PROTO_OUT)/notification/v1)
	$(call MKDIR,$(PROTO_OUT)/file/v1)
	$(call MKDIR,$(PROTO_OUT)/form/v1)
	$(PROTOC) --plugin=protoc-gen-validate=$(VALIDATE) --proto_path=$(PROTO_DIR) --proto_path=$(THIRD_PARTY) --go_out=$(PROTO_OUT) --go_opt=paths=source_relative --go-grpc_out=$(PROTO_OUT) --go-grpc_opt=paths=source_relative --validate_out="lang=go:$(PROTO_OUT)" --validate_opt=paths=source_relative $(PROTO_FILES)

wire:
	$(WIRE) gen ./internal/admin
	$(WIRE) gen ./internal/auth
	$(WIRE) gen ./internal/file
	$(WIRE) gen ./internal/form
	$(WIRE) gen ./internal/gateway
	$(WIRE) gen ./internal/notification
	$(WIRE) gen ./internal/project
	$(WIRE) gen ./internal/task
	$(WIRE) gen ./internal/team
	$(WIRE) gen ./internal/university
	$(WIRE) gen ./internal/workflow

generate: proto wire

# ==================== TESTING ====================

test:
	go test -count=1 -v ./tests/unit/...

test-auth:
	go test -count=1 -v ./tests/unit/tests_auth/...

test-file:
	go test -count=1 -v ./tests/unit/tests_file/...

test-form:
	go test -count=1 -v ./tests/unit/tests_form/...

test-notif:
	go test -count=1 -v ./tests/unit/tests_notification/...

test-project:
	go test -count=1 -v ./tests/unit/tests_project/...

test-workflow:
	go test -count=1 -v ./tests/unit/tests_workflow/...

coverage:
	$(call MKDIR,coverage)
	go test -count=1 -coverprofile=coverage/coverage.out -covermode=atomic ./tests/unit/...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	go tool cover -func=coverage/coverage.out

# ==================== BUILD ====================

define build_service
	$(GO_BUILD_ENV) go build $(LDFLAGS) -o bin/$(1)$(EXE) ./cmd/$(1)
endef

define build_linux_service
	$(CROSS_LINUX_ENV) go build $(LDFLAGS) -o bin/linux/$(1) ./cmd/$(1)
endef

build:
	$(call build_service,api_gateway)
	$(call build_service,admin_service)
	$(call build_service,auth_service)
	$(call build_service,file_service)
	$(call build_service,form_service)
	$(call build_service,notification_service)
	$(call build_service,project_service)
	$(call build_service,task_service)
	$(call build_service,team_service)
	$(call build_service,university_service)
	$(call build_service,workflow_service)

build-linux:
	$(call MKDIR,bin/linux)
	$(call build_linux_service,api_gateway)
	$(call build_linux_service,admin_service)
	$(call build_linux_service,auth_service)
	$(call build_linux_service,file_service)
	$(call build_linux_service,form_service)
	$(call build_linux_service,notification_service)
	$(call build_linux_service,project_service)
	$(call build_linux_service,task_service)
	$(call build_linux_service,team_service)
	$(call build_linux_service,university_service)
	$(call build_linux_service,workflow_service)

# ==================== DOCKER ====================

docker-up:
	docker compose up -d main_postgres
	docker compose run --rm migrations
	docker compose up -d

docker-down:
	docker compose down

docker-build:
	docker compose build

docker-logs:
	docker compose logs -f

docker-restart:
	docker compose down
	docker compose up -d --build

docker-reset-db:
	docker compose down -v
	-docker volume rm diplomaflow_diplomaflow_data
	docker compose up -d main_postgres
	$(SLEEP5)
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
	$(call RMDIR,bin)
	$(call RMDIR,coverage)
	go clean -cache -testcache

# ==================== MIGRATIONS ====================

migrate-up:
	go run ./cmd/migrate -cmd=up

migrate-down:
	go run ./cmd/migrate -cmd=down -steps=1

migrate-down-all:
	go run ./cmd/migrate -cmd=down

migrate-version:
	go run ./cmd/migrate -cmd=version

ifeq ($(OS),Windows_NT)

migrate-force:
	@cmd /V:ON /C "set /p version=Version: && go run ./cmd/migrate -cmd=force -version=!version!"

migrate-create:
	@powershell -NoProfile -Command "$$name=Read-Host 'Migration name'; $$ts=Get-Date -Format 'yyyyMMddHHmmss'; New-Item -ItemType Directory -Force db/migrations | Out-Null; New-Item -ItemType File -Path ('db/migrations/'+$$ts+'_'+$$name+'.up.sql') -Force | Out-Null; New-Item -ItemType File -Path ('db/migrations/'+$$ts+'_'+$$name+'.down.sql') -Force | Out-Null; Write-Host ('Created migration: '+$$name)"

else

migrate-force:
	@read -p "Version: " version; \
	go run ./cmd/migrate -cmd=force -version=$$version

migrate-create:
	@read -p "Migration name: " name; \
	mkdir -p db/migrations; \
	touch db/migrations/$$(date +%Y%m%d%H%M%S)_$$name.up.sql; \
	touch db/migrations/$$(date +%Y%m%d%H%M%S)_$$name.down.sql; \
	echo "Created migration: $$name"

endif