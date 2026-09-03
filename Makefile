# Build tooling for the expected-load parser plugin. See specs/ for the
# contracts each target satisfies.

BINARY   := infracost-parser-plugin-expectedload
VERSION  ?= $(shell ./scripts/version.sh)
LDFLAGS  := -s -w -X main.version=$(VERSION)
DIST     := dist

# Mirrors the Infracost CLI's own release matrix.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

FIXTURES := terraform typescript python golang java rust errors

.PHONY: build build-debug build-all test validate release clean

## build: compile the plugin for the host platform
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/$(BINARY)

## build-debug: debug build; the -debug suffix makes the CLI ignore it
build-debug:
	go build -o $(BINARY)-debug ./cmd/$(BINARY)

## build-all: cross-compile every supported platform into dist/
build-all:
	@set -e; for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe" || true; \
		out="$(DIST)/$(BINARY)_$(VERSION)_$${os}_$${arch}/$(BINARY)$$ext"; \
		echo "building $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -trimpath -ldflags "$(LDFLAGS)" -o "$$out" ./cmd/$(BINARY); \
	done

test:
	go test ./...

## validate: run the Infracost CLI's binary conformance checks
# The gate greps the subcommand listing because the CLI (cobra) prints help and
# exits 0 for unknown subcommands, so `infracost plugin validate --help` alone
# would pass spuriously on CLIs without validate support.
validate: build
	@if ! command -v infracost >/dev/null 2>&1 || \
		! infracost plugin --help 2>/dev/null | grep -qE '^[[:space:]]+validate([[:space:]]|$$)'; then \
		echo "error: your infracost CLI does not support 'plugin validate'." >&2; \
		echo "Install or build a current Infracost CLI (see README.md)." >&2; \
		exit 1; \
	fi
	infracost plugin validate ./$(BINARY)

## release: test, build all platforms, package archives + checksums into dist/
release:
	./scripts/release.sh

clean:
	rm -rf $(DIST) $(BINARY) $(BINARY).exe $(BINARY)-debug
