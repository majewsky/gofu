PKG    = github.com/majewsky/gofu
PREFIX = /usr

all: build/gofu

GO            = GOPATH=$(CURDIR)/.gopath GOBIN=$(CURDIR)/build go
GO_BUILDFLAGS =
GO_LDFLAGS    = -s -w

build/gofu: FORCE
	$(GO) install $(GO_BUILDFLAGS) -ldflags '$(GO_LDFLAGS)' '$(PKG)'

install: FORCE all
	install -D -m 0755 build/gofu "$(DESTDIR)$(PREFIX)/bin/gofu"
	ln -s gofu "$(DESTDIR)$(PREFIX)/bin/rtree"

vendor: FORCE
	golangvend

.PHONY: FORCE
