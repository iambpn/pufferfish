.PHONY: run dev build test vet fmt tidy clean wasm-build serve package help

APP_NAME := pufferfish
BIN_DIR  := bin
AIR      := $(shell go env GOPATH)/bin/air

help:
	@echo "Available targets:"
	@echo "  run          Run the app locally"
	@echo "  dev          Run the app with hot reload (rebuilds on .go changes)"
	@echo "  build        Build a native binary into $(BIN_DIR)/"
	@echo "  test         Run go test ./..."
	@echo "  vet          Run go vet ./..."
	@echo "  fmt          Run gofmt on all source files"
	@echo "  tidy         Run go mod tidy"
	@echo "  clean        Remove build artifacts"
	@echo "  wasm-build   Build the WebAssembly target into wasm/"
	@echo "  serve   			Serve the WebAssembly build with hot reload at http://localhost:8090"
	@echo "  package      Package the app for the current OS via fyne CLI"

run:
	go run .

dev:
	@command -v $(AIR) >/dev/null || go install github.com/air-verse/air@latest
	$(AIR)

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR) wasm tmp

wasm-build:
	go tool fyne build -os wasm -o wasm/$(APP_NAME).wasm

serve:
	@command -v $(AIR) >/dev/null || go install github.com/air-verse/air@latest
	$(AIR) -c .air.serve.toml

package:
	go tool fyne package -os $(shell go env GOOS)
