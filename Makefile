
BINARY_NAME=vdpacli
PACKAGE=vdpacli
ORG_PATH=github.com/amorenoz
REPO_PATH=$(ORG_PATH)/$(PACKAGE)
GOPATH=$(CURDIR)/.gopath
GOBIN=$(CURDIR)/bin
BUILDDIR=$(CURDIR)/build
BASE=$(GOPATH)/src/$(REPO_PATH)
GOFILES = $(shell find . -name *.go | grep -vE "(\/vendor\/)|(_test.go)")

export GOPATH
export GOBIN

.PHONY: all
all: vdpacli

vendor:
	@glide up

$(BASE):
	@mkdir -p $(dir $@)
	@ln -sf $(CURDIR) $@

vdpacli: $(GOFILES) vendor | $(BASE) 
	@cd $(BASE) && go build -o $(BINARY_NAME)

.PHONY: clean
clean: 
	@rm -rf vendors
	@rm -rf $(GOPATH)
	@rm -rf $(BINARY_NAME)
