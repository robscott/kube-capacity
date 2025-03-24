# default task since it's first
.PHONY: all
all: build lint test

.PHONY: build
build:
	go build -o kubectl-resource_capacity main.go

.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run --fix

.PHONY: test
test:
	go test -v ./...

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# keep version in sync with .github/workflows/golangci-lint.yaml
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= 1.61.0

.PHONY: golangci-lint
golangci-lint: $(LOCALBIN) ## Download golangci-lint (replace existing if incorrect version). Should only be used locally, not in CI.
	@(test -f $(GOLANGCI_LINT) && $(GOLANGCI_LINT) --version | grep " $(GOLANGCI_LINT_VERSION) " >/dev/null) || \
	(rm -f $(GOLANGCI_LINT) && echo "Installing $(GOLANGCI_LINT) $(GOLANGCI_LINT_VERSION)" && \
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) -d v$(GOLANGCI_LINT_VERSION))
