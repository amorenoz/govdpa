# Credit:
#   This makefile was adapted from: https://github.com/vincentbernat/hellogopher/blob/feature/glide/Makefile
#

PACKAGE=govdpa
ORG_PATH=github.com/k8snetworkplumbingwg
REPO_PATH=$(ORG_PATH)/$(PACKAGE)
GOPATH=$(CURDIR)/.gopath
GOBIN=$(CURDIR)/bin
BASE=$(GOPATH)/src/$(REPO_PATH)
PKGS=$(or $(PKG),$(shell cd $(BASE) && env GOPATH=$(GOPATH) $(GO) list ./... | grep -v "^$(PACKAGE)/vendor/"))
GOFILES = $(shell find . -name *.go | grep -vE "(\/vendor\/)|(_test.go)")
TESTPKGS = $(shell env GOPATH=$(GOPATH) $(GO) list -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' $(PKGS))

GOFILES = $(shell find . -name *.go | grep -vE "(\/vendor\/)|(_test.go)")
BUILDDIR=$(CURDIR)/build
BINARY_NAME=uvdpa-cli kvdpa-cli
BINARY_PATH=$(patsubst %, $(BUILDDIR)/%, $(BINARY_NAME))
LIBRARIES=kvdpa uvdpa


export GOPATH
export GOBIN

# Go tools
GO      = go
GODOC   = godoc
GOFMT   = gofmt
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

build: $(BINARY_PATH) ## Build binaries
	$(info Done!)

_test: | $(BASE) ; $(info Running tests...) @ ## Run tests
	$(Q) ret=0 && for pkg in $(TESTPKGS); do \
		export INTEGRATION=$(INTEGRATION); \
		cd $(GOPATH)/src/$$pkg && go test || ret=1 ; \
	done ; test $$ret -eq 0

test: INTEGRATION=no
test: _test

integration: INTEGRATION=yes
integration: _test

$(BUILDDIR)/%: $(GOFILES) | $(BUILDDIR); $(info Building $* )
	@cd cmd/$* && $(GO) build -o $(BUILDDIR)/$*


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

.PHONY: clean
clean: ; $(info  Cleaning...)	@ ## Cleanup everything
	@$(GO) clean -modcache
	@rm -rf $(GOPATH)
	@rm -rf $(BUILDDIR)
# @rm -rf test/tests.* test/coverage.*

.PHONY: help
help: ## Show help
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

