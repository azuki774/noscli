GO            ?= go
GOFMT         ?= gofmt
STATICCHECK   ?= staticcheck
PKG           ?= ./...
CMD_PACKAGE   ?= ./cmd/noscli
BIN_DIR       ?= bin
BINARY_NAME   ?= noscli
GOFILES       := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: build test gofmt-check

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_PACKAGE)

test: gofmt-check
	$(GO) test -v $(PKG)
	$(STATICCHECK) $(PKG)

gofmt-check:
	@fmt_files=`$(GOFMT) -l $(GOFILES)`; \
	if [ -n "$$fmt_files" ]; then \
		printf 'gofmt required for:\n%s\n' "$$fmt_files"; \
		exit 1; \
	fi

