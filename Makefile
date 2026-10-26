CLI_BINARY = bin/gamux
GUI_BINARY = bin/gamux-gui
PREFIX ?= $(HOME)/.local

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -s -w -X main.Version=$(VERSION)
GUI_TAGS = -tags desktop,production,gtk3,webkit2_41

.PHONY: all bindings build build-all build-cli build-gui build-frontend package clean clean-all test test-go test-go-fast test-frontend lint fmt fmt-check check check-fast check-go check-ui run run-cli run-gui install uninstall tidy update-steamless gui-audit gui-trace gui-dump gui-snap

all: test build

build: build-all

build-all: build-cli build-gui

build-cli:
	@mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(CLI_BINARY) ./cmd/gamux

build-frontend:
	@echo "Building frontend dist assets..."
	@cd frontend && ([ -d node_modules ] || ([ -f package-lock.json ] && npm ci --no-audit || npm install --no-audit)) && npm run check && npm run build

build-gui: build-frontend
	@mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" $(GUI_TAGS) -o $(GUI_BINARY) ./cmd/gamux-gui

package: build-all
	@mkdir -p bin/pkg/gamux/assets
	@rm -rf bin/pkg/gamux/*
	@mkdir -p bin/pkg/gamux/assets
	@cp $(CLI_BINARY) $(GUI_BINARY) bin/pkg/gamux/
	@cp scripts/install.sh bin/pkg/gamux/
	@find assets -maxdepth 1 -type f ! -name "*.go" -exec cp {} bin/pkg/gamux/assets/ \;
	@cp LICENSE README.md bin/pkg/gamux/
	@tar -czf bin/gamux_linux_amd64.tar.gz -C bin/pkg/gamux .
	@rm -rf bin/pkg
	@echo "==> Successfully packaged bin/gamux_linux_amd64.tar.gz"

update-steamless:
	@echo "Fetching latest Steamless release binaries from GitHub..."
	@mkdir -p steamless/bin
	curl -fsSL https://api.github.com/repos/staernid/steamless-rs/releases/latest | \
	grep "browser_download_url" | \
	cut -d '"' -f 4 | \
	xargs -n 1 wget -q -N -P steamless/bin/ || true
	chmod +x steamless/bin/steamless-linux-* steamless/bin/steamless-macos-* 2>/dev/null || true
	@echo "Successfully updated steamless/bin release assets."

clean:
	go clean
	rm -rf bin/ gamux

clean-all: clean
	rm -rf frontend/dist/* frontend/node_modules

test-go:
	go test -race $(GUI_TAGS) ./...

test-go-fast:
	go test $(GUI_TAGS) ./...

test-frontend:
	@cd frontend && npm test
	@node ./scripts/gui-eval.js audit

gui-audit:
	@node ./scripts/gui-eval.js audit

gui-trace:
	@node ./scripts/gui-eval.js trace

gui-dump:
	@node ./scripts/gui-eval.js dump

gui-snap:
	@node ./scripts/gui-eval.js snap

test: test-go test-frontend

fmt:
	gofmt -w $(shell git ls-files '*.go')

bindings:
	wails3 generate bindings -ts -d frontend/src/api/gen -f "$(GUI_TAGS)" ./cmd/gamux-gui ./gui

fmt-check:
	@out=$$(gofmt -l $(shell git ls-files '*.go')); \
	if [ -n "$$out" ]; then echo "gofmt needed (run 'make fmt'):"; echo "$$out"; exit 1; fi

lint: fmt-check
	go vet $(GUI_TAGS) ./...
	@cd frontend && npm run check
	@if which golangci-lint > /dev/null 2>&1; then golangci-lint run --build-tags desktop,production,gtk3,webkit2_41; else echo "golangci-lint not installed; skipping"; fi

check-go: fmt-check
	go vet $(GUI_TAGS) ./...
	@if which golangci-lint > /dev/null 2>&1; then golangci-lint run --build-tags desktop,production,gtk3,webkit2_41; else echo "golangci-lint not installed; skipping"; fi
	go test $(GUI_TAGS) ./...

check-ui:
	@cd frontend && npm run check && npm test
	@node ./scripts/gui-eval.js audit

check-fast: check-go check-ui
	@echo "⚡ Fast verification passed cleanly!"

check: lint test-go test-frontend build
	@echo "✨ All Go and Frontend tests, lints, and builds passed cleanly!"

run: run-cli

run-cli: build-cli
	./$(CLI_BINARY)

run-gui: build-gui
	./$(GUI_BINARY)

tidy:
	go mod tidy

install: build-all
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(CLI_BINARY) $(DESTDIR)$(PREFIX)/bin/gamux
	install -m 755 $(GUI_BINARY) $(DESTDIR)$(PREFIX)/bin/gamux-gui
	install -d $(DESTDIR)$(PREFIX)/share/applications
	install -m 644 assets/io.github.staernid.gamux.desktop $(DESTDIR)$(PREFIX)/share/applications/io.github.staernid.gamux.desktop
	install -d $(DESTDIR)$(PREFIX)/share/icons/hicolor/scalable/apps
	install -m 644 assets/io.github.staernid.gamux.svg $(DESTDIR)$(PREFIX)/share/icons/hicolor/scalable/apps/io.github.staernid.gamux.svg
	install -d $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps
	install -m 644 assets/io.github.staernid.gamux.png $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps/io.github.staernid.gamux.png

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/gamux
	rm -f $(DESTDIR)$(PREFIX)/bin/gamux-gui
	rm -f $(DESTDIR)$(PREFIX)/share/applications/io.github.staernid.gamux.desktop
	rm -f $(DESTDIR)$(PREFIX)/share/icons/hicolor/scalable/apps/io.github.staernid.gamux.svg
	rm -f $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps/io.github.staernid.gamux.png
