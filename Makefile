PREFIX  = /usr
DESTDIR =
APPLETS = mdedit rtree prettyprompt

all: build/gofu $(addprefix build/,$(APPLETS))

GO_BUILDFLAGS = -mod vendor
GO_LDFLAGS = 
GO_TESTENV = 

build/gofu: FORCE
	go build $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -o build/gofu .
$(addprefix build/,$(APPLETS)):
	ln -s gofu $@

# not using `install -D` here because macOS is stupid
install: FORCE all
	install -d -m 0755 "$(DESTDIR)$(PREFIX)/bin"
	install -m 0755 build/gofu "$(DESTDIR)$(PREFIX)/bin/gofu"
	for APPLET in $(APPLETS); do ln -s gofu "$(DESTDIR)$(PREFIX)/bin/$${APPLET}"; done

# which packages to test with static checkers
GO_ALLPKGS := $(shell go list ./...)
# which files to test with static checkers (this contains a list of globs)
GO_ALLFILES := $(addsuffix /*.go,$(patsubst $(shell go list .),.,$(shell go list ./...)))
# which packages to test with "go test"
GO_TESTPKGS := $(shell go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./...)
# which packages to measure coverage for
GO_COVERPKGS := $(shell go list ./... | grep -Ev '/plugins')
# to get around weird Makefile syntax restrictions, we need variables containing a space and comma
space := $(null) $(null)
comma := ,

check: all static-check build/cover.html FORCE
	@printf "\e[1;32m>> All checks successful.\e[0m\n"

static-check: FORCE
	@if ! hash golint 2>/dev/null; then printf "\e[1;36m>> Installing golint...\e[0m\n"; GO111MODULE=off go get -u golang.org/x/lint/golint; fi
	@printf "\e[1;36m>> gofmt\e[0m\n"
	@if s="$$(gofmt -s -d $(GO_ALLFILES) 2>/dev/null)" && test -n "$$s"; then echo "$$s"; false; fi
	@printf "\e[1;36m>> golint\e[0m\n"
	@if s="$$(golint $(GO_ALLPKGS) 2>/dev/null)" && test -n "$$s"; then echo "$$s"; false; fi
	@printf "\e[1;36m>> go vet\e[0m\n"
	@go vet $(GO_BUILDFLAGS) $(GO_ALLPKGS)

build/cover.out: FORCE
	@printf "\e[1;36m>> go test\e[0m\n"
	@env $(GO_TESTENV) go test $(GO_BUILDFLAGS) -ldflags '-s -w $(GO_LDFLAGS)' -p 1 -coverprofile=$@ -covermode=count -coverpkg=$(subst $(space),$(comma),$(GO_COVERPKGS)) $(GO_TESTPKGS)

build/cover.html: build/cover.out
	@printf "\e[1;36m>> go tool cover > build/cover.html\e[0m\n"
	@go tool cover -html $< -o $@

vendor: FORCE
	go mod tidy
	go mod vendor
	go mod verify

.PHONY: FORCE
