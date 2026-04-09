.PHONY: build install clean run test test-verbose test-cover

BINARY_NAME=lazyconfigs
INSTALL_PATH=$(HOME)/.local/bin
VERSION ?= dev
LDFLAGS=-s -w -X lazyconfigs/internal/version.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/lazyconfigs

install: build
	mkdir -p $(INSTALL_PATH)
	cp $(BINARY_NAME) $(INSTALL_PATH)/

clean:
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

test:
	go test $$(go list ./... | grep -v cmd/ | grep -v version) -count=1

test-verbose:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out
