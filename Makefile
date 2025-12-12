# --- Переменные ---
APP_NAME    := cognitive-server
CMD_PATH    := ./cmd/server/main.go
BUILD_DIR   := ./bin

GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
    BIN_SUFFIX := .exe
else
    BIN_SUFFIX :=
endif

BIN_NAME    := $(APP_NAME)$(BIN_SUFFIX)
BIN_PATH    := $(BUILD_DIR)/$(BIN_NAME)

# --- Цели Make (Phony) ---
.PHONY: all run build test lint fmt clean tools

all: build

# --- Сборка и Запуск ---

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

$(BIN_PATH): $(CMD_PATH) $(BUILD_DIR)
	@echo "Building $(APP_NAME) for $(GOOS)/$(GOARCH)..."
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BIN_PATH) $(CMD_PATH)

build: $(BIN_PATH)
	@echo "Successfully built to $(BIN_PATH)"

run: build
	@echo "Running $(APP_NAME) from $(BIN_PATH)..."
	@./$(BIN_PATH)

# --- Инструменты Разработки ---

test:
	@echo "Running tests (with race detector)..."
	go test -v -race ./...

lint: fmt
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo >&2 "Warning: golangci-lint not found. Install it for linting."; }
	golangci-lint run

fmt:
	@echo "Formatting code (go fmt)..."
	@go fmt ./...

# --- Очистка ---

clean:
	@echo "Cleaning up build directory $(BUILD_DIR)..."
	@rm -rf $(BUILD_DIR)

tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest