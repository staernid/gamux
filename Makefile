BINARY_NAME=gamux
CMD_PATH=.
PREFIX ?= ~/.local

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -s -w -X main.Version=$(VERSION)

.PHONY: all build clean test lint run install uninstall tidy update-steamless

all: test build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_PATH)

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
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

lint:
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || true

run:
	go run -ldflags "$(LDFLAGS)" $(CMD_PATH)

tidy:
	go mod tidy

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BINARY_NAME) $(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)
