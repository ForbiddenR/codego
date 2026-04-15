.PHONY: build test run clean fmt vet

BINARY=codego
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/codego

test:
	go test ./... -v -count=1

run: build
	./bin/$(BINARY)

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf bin/
