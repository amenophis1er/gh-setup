BINARY := gh-setup
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-s -w -X github.com/amenophis1er/gh-setup/cmd.version=$(VERSION)"

.PHONY: build install test vet clean

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

all: vet test build
