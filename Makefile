PKG    = github.com/majewsky/rtree
PREFIX = /usr

all: build/rtree

GO            = GOPATH=$(CURDIR)/.gopath GOBIN=$(CURDIR)/build go
GO_BUILDFLAGS =
GO_LDFLAGS    = -s -w

build/rtree: FORCE
	$(GO) install $(GO_BUILDFLAGS) -ldflags '$(GO_LDFLAGS)' '$(PKG)'

install: FORCE all
	install -D -m 0755 build/rtree "$(DESTDIR)$(PREFIX)/bin/rtree"

vendor: FORCE
	golangvend

.PHONY: FORCE
