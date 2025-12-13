# --- Project ---
APP_NAME    := cognitive-server
MODULE_PATH := cognitive-server
CMD_PATH    := ./cmd/server/main.go
BUILD_DIR   := ./bin

# --- Go env ---
GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
	BIN_SUFFIX := .exe
else
	BIN_SUFFIX :=
endif

BIN_NAME := $(APP_NAME)$(BIN_SUFFIX)
BIN_PATH := $(BUILD_DIR)/$(BIN_NAME)

# --- Build metadata ---
BUILD_DATE   := $(shell date -u +%Y-%m-%d)
GIT_COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
GIT_BRANCH   := $(shell git branch --show-current 2>/dev/null || echo unknown)
BUILD_SYSTEM := $(shell echo $$CI)

LDFLAGS := \
	-X '$(MODULE_PATH)/internal/version.BuildDate=$(BUILD_DATE)' \
	-X '$(MODULE_PATH)/internal/version.BuildCommit=$(GIT_COMMIT)' \
	-X '$(MODULE_PATH)/internal/version.BuildBranch=$(GIT_BRANCH)' \
	-X '$(MODULE_PATH)/internal/version.BuildCI=$(BUILD_SYSTEM)'

# --- Phony targets ---
.PHONY: all build run test lint fmt clean tools

all: build

# --- Build ---
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

$(BIN_PATH): $(CMD_PATH) | $(BUILD_DIR)
	@echo "Building $(APP_NAME) for $(GOOS)/$(GOARCH)"
	GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags "$(LDFLAGS)" \
		-o $(BIN_PATH) $(CMD_PATH)

build: $(BIN_PATH)
	@echo "Built: $(BIN_PATH)"

run: build
	@echo "Running $(APP_NAME)"
	@./$(BIN_PATH)

# --- Dev ---
test:
	@echo "Running tests"
	go test -v -race ./...

fmt:
	@echo "Formatting"
	go fmt ./...

lint: fmt
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found; skipping lint"; exit 0; }
	golangci-lint run

tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# --- Clean ---
clean:
	@echo "Cleaning $(BUILD_DIR)"
	rm -rf $(BUILD_DIR)
