PKG    = github.com/majewsky/gofu
PREFIX = /usr

APPLETS = i3status rtree prettyprompt

all: build/gofu $(addprefix build/,$(APPLETS))

GO            = GOPATH=$(CURDIR)/.gopath GOBIN=$(CURDIR)/build go
GO_BUILDFLAGS =
GO_LDFLAGS    = -s -w

build/gofu: FORCE
	$(GO) install $(GO_BUILDFLAGS) -ldflags '$(GO_LDFLAGS)' '$(PKG)'
$(addprefix build/,$(APPLETS)):
	ln -s gofu $@

# not using `install -D` here because macOS is stupid
install: FORCE all
	install -d -m 0755 "$(DESTDIR)$(PREFIX)/bin"
	install -m 0755 build/gofu "$(DESTDIR)$(PREFIX)/bin/gofu"
	for APPLET in $(APPLETS); do ln -s gofu "$(DESTDIR)$(PREFIX)/bin/$${APPLET}"; done

# which packages to test with static checkers?
GO_ALLPKGS := $(PKG) $(shell go list $(PKG)/pkg/...)
# which packages to test with `go test`?
GO_TESTPKGS := $(shell go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' $(PKG)/pkg/...)
# which packages to measure coverage for?
GO_COVERPKGS := $(shell go list $(PKG)/pkg/...)
# output files from `go test`
GO_COVERFILES := $(patsubst %,build/%.cover.out,$(subst /,_,$(GO_TESTPKGS)))

# down below, I need to substitute spaces with commas; because of the syntax,
# I have to get these separators from variables
space := $(null) $(null)
comma := ,

check: all static-check build/cover.html FORCE
	@echo -e "\e[1;32m>> All tests successful.\e[0m"
static-check: FORCE
	@if s="$$(gofmt -s -l *.go pkg 2>/dev/null)"                            && test -n "$$s"; then printf ' => %s\n%s\n' gofmt  "$$s"; false; fi
	@if s="$$(golint . && find pkg -type d -exec golint {} \; 2>/dev/null)" && test -n "$$s"; then printf ' => %s\n%s\n' golint "$$s"; false; fi
	$(GO) vet $(GO_ALLPKGS)
build/%.cover.out: FORCE
	$(GO) test $(GO_BUILDFLAGS) -ldflags '$(GO_LDFLAGS)' -coverprofile=$@ -covermode=count -coverpkg=$(subst $(space),$(comma),$(GO_COVERPKGS)) $(subst _,/,$*)
build/cover.out: $(GO_COVERFILES)
	util/gocovcat.go $(GO_COVERFILES) > $@
build/cover.html: build/cover.out
	$(GO) tool cover -html $< -o $@

vendor: FORCE
	golangvend

.PHONY: FORCE
