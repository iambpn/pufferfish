.PHONY: run dev build test vet fmt tidy clean wasm-build serve package \
        package-linux package-windows package-darwin package-all help

# Must match ID in FyneApp.toml (fyne-cross rejects a bare app-id for the
# Windows target, so both use this reverse-DNS form). The installed
# .desktop matches windows via StartupWMClass=pufferfish (Fyne's Name field).
APP_NAME   := pufferfish
APP_ID     := com.iambpn.pufferfish
BIN_DIR    := bin
AIR        := $(shell go env GOPATH)/bin/air
FYNE_CROSS := $(shell go env GOPATH)/bin/fyne-cross
# go.mod pins a newer Go than the fyne-cross images ship, so let the
# container fetch the matching toolchain.
FYNE_CROSS_ENV := -env GOTOOLCHAIN=auto -app-id=$(APP_ID)

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
	@echo "  package         Package the app for the current OS via fyne CLI"
	@echo "  package-linux   Cross-package a Linux build (fyne-cross + Docker)"
	@echo "  package-windows Cross-package a Windows build (fyne-cross + Docker)"
	@echo "  package-darwin  Cross-package a macOS build (needs FYNE_CROSS_MACOS_SDK)"
	@echo "  package-all     Cross-package Linux + Windows + macOS"

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

$(FYNE_CROSS):
	go install github.com/fyne-io/fyne-cross@latest

package-linux: $(FYNE_CROSS)
	$(FYNE_CROSS) linux -arch=amd64,arm64 $(FYNE_CROSS_ENV)

package-windows: $(FYNE_CROSS)
	$(FYNE_CROSS) windows -arch=amd64 $(FYNE_CROSS_ENV)

# macOS cross-builds need an Apple macOSX SDK; point FYNE_CROSS_MACOS_SDK
# at an extracted SDK dir (Apple's licence only allows this on a Mac).
package-darwin: $(FYNE_CROSS)
	$(FYNE_CROSS) darwin -arch=amd64,arm64 $(FYNE_CROSS_ENV) \
		-macosx-sdk-path "$(FYNE_CROSS_MACOS_SDK)"

package-all: package-linux package-windows package-darwin
