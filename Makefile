PREFIX ?= /opt/homebrew
BINDIR ?= $(PREFIX)/bin
BIN := bin/wagon

.PHONY: build install uninstall test

build:
	go build -o $(BIN) ./cmd/wagon

install: build
	mkdir -p "$(BINDIR)"
	ln -sf "$(CURDIR)/$(BIN)" "$(BINDIR)/wagon"

uninstall:
	rm -f "$(BINDIR)/wagon"

test:
	go test ./...
