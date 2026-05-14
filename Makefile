BINARY      := reverb
MODULE      := github.com/amarilz/reverb
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_FLAGS := -trimpath -ldflags "-s -w -X reverb/internal/app.Version=$(VERSION)"

CMD := ./cmd/reverb

# ── Development ───────────────────────────────────────────────────────────────

.PHONY: build
build:
	go build $(BUILD_FLAGS) -o $(BINARY) $(CMD)

.PHONY: run
run: build
	./$(BINARY)

.PHONY: test
test:
	go test ./... -race -count=1

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: tidy
tidy:
	go mod tidy

# ── Install ───────────────────────────────────────────────────────────────────

.PHONY: install
install:
	go install $(BUILD_FLAGS) $(CMD)

# ── Cross-platform release builds ─────────────────────────────────────────────

DIST := dist

.PHONY: release
release: clean
	mkdir -p $(DIST)

	GOOS=darwin  GOARCH=amd64  go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-darwin-amd64       $(CMD)
	GOOS=darwin  GOARCH=arm64  go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-darwin-arm64       $(CMD)

	GOOS=linux   GOARCH=amd64  go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-linux-amd64        $(CMD)
	GOOS=linux   GOARCH=arm64  go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-linux-arm64        $(CMD)

	GOOS=windows GOARCH=amd64  go build $(BUILD_FLAGS) -o $(DIST)/$(BINARY)-windows-amd64.exe  $(CMD)

.PHONY: clean
clean:
	rm -rf $(BINARY) $(DIST)

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build    — build for current platform"
	@echo "  run      — build and run"
	@echo "  test     — run tests with race detector"
	@echo "  vet      — run go vet"
	@echo "  lint     — run golangci-lint"
	@echo "  tidy     — go mod tidy"
	@echo "  install  — install to GOPATH/bin"
	@echo "  release  — cross-compile for all targets into dist/"
	@echo "  clean    — remove build artifacts"
