CLI_BINARY = bin/gamux
GUI_BINARY = bin/gamux-gui
PREFIX ?= $(HOME)/.local

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -s -w -X main.Version=$(VERSION)
GUI_TAGS = -tags desktop,production,webkit2_41

.PHONY: all build build-all build-cli build-gui build-frontend appimage clean clean-all test lint run run-gui install uninstall tidy update-steamless

all: test build

build: build-all

build-all: build-cli build-gui

build-cli:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o $(CLI_BINARY) ./cmd/gamux
	@cp -f $(CLI_BINARY) gamux

build-frontend:
	@echo "Building frontend dist assets..."
	@cd frontend && ([ -f package-lock.json ] && npm ci || npm install) && npm run build

build-gui: build-frontend
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" $(GUI_TAGS) -o $(GUI_BINARY) ./cmd/gamux-gui

appimage: build-gui
	@./scripts/build-appimage.sh $(GUI_BINARY) bin/gamux-gui-x86_64.AppImage

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
	rm -rf frontend/dist frontend/node_modules

test:
	go test -v ./...

lint:
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || true

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

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/gamux
	rm -f $(DESTDIR)$(PREFIX)/bin/gamux-gui
	rm -f $(DESTDIR)$(PREFIX)/share/applications/io.github.staernid.gamux.desktop
	rm -f $(DESTDIR)$(PREFIX)/share/icons/hicolor/scalable/apps/io.github.staernid.gamux.svg
