BINARY_NAME=gbe_fork_helper
CMD_PATH=./cmd/gbe_fork_helper

.PHONY: all build clean test lint run

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
