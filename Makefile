# Makefile for the project

# tools
CUR_DIR=$(shell pwd)
INSTALL_PREFIX=$(CUR_DIR)/bin
VENDOR_DIR=vendor
BINARY_NAME=osio
ifeq ($(OS),Windows_NT)
BINARY_PATH=$(INSTALL_PREFIX)/$(BINARY_NAME).exe
else
BINARY_PATH=$(INSTALL_PREFIX)/$(BINARY_NAME)
endif

# Call this function with $(call log-info,"Your message")
define log-info =
@echo "INFO: $(1)"
endef


.PHONY: help
# Based on https://gist.github.com/rcmachado/af3db315e31383502660
## Display this help text.
help:/
	$(info Available targets)
	$(info -----------------)
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		helpCommand = substr($$1, 0, index($$1, ":")-1); \
		if (helpMessage) { \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			gsub(/##/, "\n                                     ", helpMessage); \
		} else { \
			helpMessage = "(No documentation)"; \
		} \
		printf "%-35s - %s\n", helpCommand, helpMessage; \
		lastLine = "" \
	} \
	{ hasComment = match(lastLine, /^## (.*)/); \
          if(hasComment) { \
            lastLine=lastLine$$0; \
	  } \
          else { \
	    lastLine = $$0 \
          } \
        }' $(MAKEFILE_LIST)

.PHONY: deps
## Download build dependencies.
deps: $(VENDOR_DIR)

$(VENDOR_DIR):
	dep ensure

$(INSTALL_PREFIX):
# Build artifacts dir
	@mkdir -p $(INSTALL_PREFIX)

.PHONY: prebuild-checks
## Check that all tools where found
prebuild-checks: $(INSTALL_PREFIX)

.PHONY: build
## build the binary executable from CLI
build: $(INSTALL_PREFIX) deps 
	$(eval BUILD_COMMIT:=$(shell git rev-parse --short HEAD))
	$(eval BUILD_TAG:=$(shell git tag --contains $(BUILD_COMMIT)))
	@echo "building $(BINARY_PATH) (commit:$(BUILD_COMMIT) / tag:$(BUILD_TAG))"
	@go build -ldflags \
	  " -X github.com/fabric8-services/fabric8-auth-cli/cmd.BinaryName=$(BINARY_NAME)\
	    -X github.com/fabric8-services/fabric8-auth-cli/cmd.BuildCommit=$(BUILD_COMMIT) \
	    -X github.com/fabric8-services/fabric8-auth-cli/cmd.BuildTag=$(BUILD_TAG)" \
	  -o $(BINARY_PATH) \
	  main*.go


lint:
	@go get -v github.com/golangci/golangci-lint/cmd/golangci-lint
	@golangci-lint run -E gofmt,golint,megacheck,misspell ./...


.PHONY: install
## installs the binary executable in the $GOPATH/bin directory
install: build
	@cp $(BINARY_PATH) $(GOPATH)/bin
