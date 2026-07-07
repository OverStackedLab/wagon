PREFIX ?= /opt/homebrew
BINDIR ?= $(PREFIX)/bin
BIN := bin/wagon
MODULE := github.com/OverStackedLab/wagon
VERSION ?= 0.1.0-dev
LDFLAGS := -X $(MODULE)/internal/cli.version=$(VERSION)

.PHONY: build install uninstall test release-check

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/wagon

install: build
	mkdir -p "$(BINDIR)"
	ln -sf "$(CURDIR)/$(BIN)" "$(BINDIR)/wagon"

uninstall:
	rm -f "$(BINDIR)/wagon"

test:
	go test ./...

release-check: test
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/wagon
	./$(BIN) version
