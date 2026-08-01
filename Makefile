BINARY_NAME=gbe_fork_helper
CMD_PATH=./cmd/gbe_fork_helper
PREFIX ?= ~/.local

.PHONY: all build clean test lint run install uninstall

all: test build

build:
	go build -o $(BINARY_NAME) $(CMD_PATH)

clean:
	go clean
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

lint:
	golangci-lint run

run:
	go run $(CMD_PATH)

tidy:
	go mod tidy

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BINARY_NAME) $(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY_NAME)
