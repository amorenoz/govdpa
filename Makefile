# Credit:
#   This makefile was adapted from: https://github.com/vincentbernat/hellogopher/blob/feature/glide/Makefile
#

BINARY_NAME=vdpacli
PACKAGE=govdpa
ORG_PATH=github.com/amorenoz
REPO_PATH=$(ORG_PATH)/$(PACKAGE)
GOPATH=$(CURDIR)/.gopath
GOBIN=$(CURDIR)/bin
BUILDDIR=$(CURDIR)/build
BASE=$(GOPATH)/src/$(REPO_PATH)
PKGS=$(or $(PKG),$(shell cd $(BASE) && env GOPATH=$(GOPATH) $(GO) list ./... | grep -v "^$(PACKAGE)/vendor/"))
GOFILES = $(shell find . -name *.go | grep -vE "(\/vendor\/)|(_test.go)")
TESTPKGS = $(shell env GOPATH=$(GOPATH) $(GO) list -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' $(PKGS))

GLIDE=glide
GOFILES = $(shell find . -name *.go | grep -vE "(\/vendor\/)|(_test.go)")

export GOPATH
export GOBIN

# Go tools
GO      = go
GODOC   = godoc
GOFMT   = gofmt
GLIDE   = glide
TIMEOUT = 15
V = 0
Q = $(if $(filter 1,$V),,@)

.PHONY: all
all: fmt lint build

$(BASE): ; $(info  setting GOPATH...)
	@mkdir -p $(dir $@)
	@ln -sf $(CURDIR) $@

$(GOBIN):
	@mkdir -p $@

$(BUILDDIR): | $(BASE) ; $(info Creating build directory...)
	@cd $(BASE) && mkdir -p $@

build: $(BUILDDIR)/$(BINARY_NAME) | ; $(info Building $(BINARY_NAME)...)
	$(info Done!)

$(BUILDDIR)/$(BINARY_NAME): $(GOFILES) | $(BUILDDIR)
	@cd $(BASE)/cmd/$(BINARY_NAME) && CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BUILDDIR)/$(BINARY_NAME) -tags no_openssl -v

# Tools

GOLINT = $(GOBIN)/golint
$(GOBIN)/golint: | $(BASE) ; $(info  building golint...)
	$Q go get -u golang.org/x/lint/golint
.PHONY: lint
lint: | $(BASE) $(GOLINT) ; $(info  running golint...) @ ## Run golint
	$Q cd $(BASE) && ret=0 && for pkg in $(PKGS); do \
		test -z "$$($(GOLINT) $$pkg | tee /dev/stderr)" || ret=1 ; \
	 done ; exit $$ret

.PHONY: fmt
fmt: ; $(info  running gofmt...) @ ## Run gofmt on all source files
	@ret=0 && for d in $$($(GO) list -f '{{.Dir}}' ./... | grep -v /vendor/); do \
		$(GOFMT) -l -w $$d/*.go || ret=$$? ; \
	 done ; exit $$ret

# Dependency management
glide.lock: glide.yaml | $(BASE) ; $(info  updating dependencies...)
	$Q cd $(BASE) && $(GLIDE) update -v
	@touch $@

vendor: glide.lock | $(BASE) ; $(info  retrieving dependencies...)
	$Q cd $(BASE) && $(GLIDE) --quiet install -v
	@ln -nsf . vendor/src
	@touch $@


# Misc
.PHONY: clean
clean: ; $(info  Cleaning...)	@ ## Cleanup everything
	@rm -rf $(GOPATH)
	@rm -rf $(BUILDDIR)/$(BINARY_NAME)
# @rm -rf test/tests.* test/coverage.*

.PHONY: help
help:
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
