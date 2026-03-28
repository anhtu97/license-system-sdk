## LicenseOS Native SDK — Build Targets
##
## Prerequisites:
##   Linux/macOS:  Go 1.21+, gcc (CGo)
##   Windows DLL:  mingw-w64  (sudo apt install gcc-mingw-w64-x86-64)
##
## Usage:
##   make linux        → dist/linux/liblicenseos.so + licenseos.h
##   make macos        → dist/macos/liblicenseos.dylib + licenseos.h
##   make windows      → dist/windows/licenseos.dll + licenseos.h
##   make all          → all three platforms
##   make clean        → remove dist/

GOFLAGS := CGO_ENABLED=1
LDFLAGS := -ldflags="-w -s"
OUT_DIR := dist

.PHONY: all linux macos windows clean

all: linux macos windows

## ── Linux ────────────────────────────────────────────────────────────────────
linux: $(OUT_DIR)/linux/liblicenseos.so

$(OUT_DIR)/linux/liblicenseos.so:
	@mkdir -p $(OUT_DIR)/linux
	$(GOFLAGS) GOOS=linux GOARCH=amd64 \
	  go build -buildmode=c-shared $(LDFLAGS) \
	  -o $(OUT_DIR)/linux/liblicenseos.so .
	@cp include/licenseos.h $(OUT_DIR)/linux/licenseos.h
	@echo "✓ Linux: $(OUT_DIR)/linux/liblicenseos.so"

## ── macOS ────────────────────────────────────────────────────────────────────
macos: $(OUT_DIR)/macos/liblicenseos.dylib

$(OUT_DIR)/macos/liblicenseos.dylib:
	@mkdir -p $(OUT_DIR)/macos
	$(GOFLAGS) GOOS=darwin GOARCH=amd64 \
	  go build -buildmode=c-shared $(LDFLAGS) \
	  -o $(OUT_DIR)/macos/liblicenseos.dylib .
	@cp include/licenseos.h $(OUT_DIR)/macos/licenseos.h
	@echo "✓ macOS:  $(OUT_DIR)/macos/liblicenseos.dylib"

## ── macOS ARM64 (Apple Silicon) ──────────────────────────────────────────────
macos-arm64: $(OUT_DIR)/macos-arm64/liblicenseos.dylib

$(OUT_DIR)/macos-arm64/liblicenseos.dylib:
	@mkdir -p $(OUT_DIR)/macos-arm64
	$(GOFLAGS) GOOS=darwin GOARCH=arm64 \
	  go build -buildmode=c-shared $(LDFLAGS) \
	  -o $(OUT_DIR)/macos-arm64/liblicenseos.dylib .
	@cp include/licenseos.h $(OUT_DIR)/macos-arm64/licenseos.h
	@echo "✓ macOS ARM64: $(OUT_DIR)/macos-arm64/liblicenseos.dylib"

## ── Windows (cross-compile from Linux/macOS via mingw-w64) ──────────────────
## Install: sudo apt install gcc-mingw-w64-x86-64
##          brew install mingw-w64
windows: $(OUT_DIR)/windows/licenseos.dll

$(OUT_DIR)/windows/licenseos.dll:
	@mkdir -p $(OUT_DIR)/windows
	$(GOFLAGS) GOOS=windows GOARCH=amd64 \
	  CC=x86_64-w64-mingw32-gcc \
	  go build -buildmode=c-shared $(LDFLAGS) \
	  -o $(OUT_DIR)/windows/licenseos.dll .
	@cp include/licenseos.h $(OUT_DIR)/windows/licenseos.h
	@echo "✓ Windows: $(OUT_DIR)/windows/licenseos.dll"

## ── Go dependency sync ───────────────────────────────────────────────────────
tidy:
	go mod tidy

## ── Linux (via Docker — no local toolchain needed) ──────────────────────────
## Produces a statically-linked musl binary compatible with Alpine and glibc distros.
linux-docker:
	@mkdir -p $(OUT_DIR)/linux
	docker run --rm \
	  -v "$(shell pwd)/..:/sdk" \
	  -w /sdk/clib \
	  golang:1.21-alpine \
	  sh -c "apk add --no-cache gcc musl-dev && \
	         go mod tidy && \
	         CGO_ENABLED=1 go build -buildmode=c-shared \
	           -ldflags='-w -s' -o /out/liblicenseos.so ." \
	  -v "$(shell pwd)/$(OUT_DIR)/linux:/out"
	@cp include/licenseos.h $(OUT_DIR)/linux/licenseos.h
	@echo "✓ Linux (Docker): $(OUT_DIR)/linux/liblicenseos.so"

## ── Clean ────────────────────────────────────────────────────────────────────
clean:
	rm -rf $(OUT_DIR)
