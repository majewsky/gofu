PKG    = github.com/majewsky/gofu
PREFIX = /usr

APPLETS = rtree

all: build/gofu $(addprefix build/,$(APPLETS))

GO            = GOPATH=$(CURDIR)/.gopath GOBIN=$(CURDIR)/build go
GO_BUILDFLAGS =
GO_LDFLAGS    = -s -w

build/gofu: FORCE
	$(GO) install $(GO_BUILDFLAGS) -ldflags '$(GO_LDFLAGS)' '$(PKG)'
build/%:
	ln -s gofu $@

install: FORCE all
	install -D -m 0755 build/gofu "$(DESTDIR)$(PREFIX)/bin/gofu"
	for APPLET in $(APPLETS); do ln -s gofu "$(DESTDIR)$(PREFIX)/bin/$${APPLET}"; done

vendor: FORCE
	golangvend

.PHONY: FORCE
