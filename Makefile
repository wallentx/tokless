SHELL := /bin/sh

.DEFAULT_GOAL := help
.NOTPARALLEL:

BIN ?= tokless
CMD_PATH ?= ./cmd/tokless
GO ?= go
TIMEOUT ?= 300s
TOOLS_DIR ?= $(CURDIR)/.tools
TOOLS_BIN ?= $(TOOLS_DIR)/bin
MODULE_PATH := github.com/wallentx/tokless
LATEST_TAG ?= $(shell git describe --tags --abbrev=0 2>/dev/null || printf '%s' v0.0.0)
GIT_SHORT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf '%s' unknown)
DEV_VERSION ?= $(LATEST_TAG)-$(GIT_SHORT_SHA)
RAW_VERSION ?= $(shell if [ -n "$$RELEASE_TAG" ]; then printf '%s' "$$RELEASE_TAG"; else printf '%s' "$(DEV_VERSION)"; fi)
VERSION ?= $(patsubst v%,%,$(RAW_VERSION))
DIST_DIR ?= dist
PKGS ?= ./...
GO_LDFLAGS ?= -s -w -X $(MODULE_PATH)/internal/util.Version=$(VERSION)
PAGER ?= cat
GIT_PAGER ?= cat
LESS ?= -FRX

export PAGER GIT_PAGER LESS

GOFILES := $(shell find . -type f -name '*.go' -not -path './vendor/*' -not -path './.tools/*' -not -path './dist/*')

GOIMPORTS := $(shell command -v goimports 2>/dev/null || printf '%s' '$(TOOLS_BIN)/goimports')
STATICCHECK := $(shell command -v staticcheck 2>/dev/null || printf '%s' '$(TOOLS_BIN)/staticcheck')
GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null || printf '%s' '$(TOOLS_BIN)/golangci-lint')
ERRCHECK := $(shell command -v errcheck 2>/dev/null || printf '%s' '$(TOOLS_BIN)/errcheck')
GOSEC := $(shell command -v gosec 2>/dev/null || printf '%s' '$(TOOLS_BIN)/gosec')
GOVULNCHECK := $(shell command -v govulncheck 2>/dev/null || printf '%s' '$(TOOLS_BIN)/govulncheck')

GOIMPORTS_PKG := golang.org/x/tools/cmd/goimports@v0.42.0
STATICCHECK_PKG := honnef.co/go/tools/cmd/staticcheck@v0.7.0
GOLANGCI_LINT_PKG := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4
ERRCHECK_PKG := github.com/kisielk/errcheck@v1.10.0
GOSEC_PKG := github.com/securego/gosec/v2/cmd/gosec@v2.24.6
GOVULNCHECK_PKG := golang.org/x/vuln/cmd/govulncheck@v1.3.0

COLOR ?= auto
COLOR_ENABLED :=
ifeq ($(COLOR),always)
COLOR_ENABLED := 1
else ifeq ($(COLOR),never)
COLOR_ENABLED :=
else ifneq ($(NO_COLOR),)
COLOR_ENABLED :=
else ifneq ($(MAKE_TERMOUT),)
COLOR_ENABLED := 1
endif

ifeq ($(COLOR_ENABLED),1)
COLOR_STEP := \033[1;36m
COLOR_OK := \033[1;32m
COLOR_WARN := \033[1;33m
COLOR_FAIL := \033[1;31m
COLOR_TITLE := \033[1;37m
COLOR_TARGET := \033[36m
COLOR_DIM := \033[2m
COLOR_RESET := \033[0m
endif

PRINT_STEP = @printf '$(COLOR_STEP)==>$(COLOR_RESET) %s\n' '$(1)'
PRINT_OK = @printf '$(COLOR_OK)OK:$(COLOR_RESET) %s\n' '$(1)'

.PHONY: all
all: fix check ## Apply mechanical fixes, then run the standard local gate

.PHONY: check
check: verify-modules format-check tidy-check scripts-check diff-check vet test build ## Run the standard local gate

.PHONY: full-check
full-check: check strict-lint security ## Run the standard gate plus strict lint and security scanners

.PHONY: ci
ci: scripts-check vet test build-all ## Run the same core checks as GitHub Actions

.PHONY: qa
qa: format-check tidy-check scripts-check diff-check vet ## Run quality checks without tests or build

.PHONY: fix
fix: format tidy-fix ## Apply mechanical formatting/import/module fixes

.PHONY: lint
lint: vet ## Run baseline static analysis checks

.PHONY: strict-lint
strict-lint: staticcheck golangci-lint errcheck ## Run optional strict static analysis checks

.PHONY: security
security: gosec govulncheck ## Run heavier security scanners

.PHONY: format
format: fmt-fix imports-fix ## Format code with gofmt and goimports

.PHONY: fmt
fmt: format ## Alias for format

.PHONY: format-check
format-check: fmt-check imports-check ## Check formatting with gofmt and goimports

.PHONY: verify-modules
verify-modules: ## Download and verify modules
	$(call PRINT_STEP,Module verification)
	@$(GO) mod download
	@$(GO) mod verify
	$(call PRINT_OK,go mod verified)

.PHONY: fmt-check
fmt-check: ## Check gofmt formatting
	$(call PRINT_STEP,gofmt check)
	@if [ -n "$(GOFILES)" ]; then \
		issues="$$(gofmt -l -s $(GOFILES))"; \
		if [ -n "$$issues" ]; then \
			printf '$(COLOR_FAIL)FAIL:$(COLOR_RESET) %s\n' 'gofmt issues found (run make fmt-fix or make fix)' >&2; \
			printf '%s\n' "$$issues"; \
			exit 1; \
		fi; \
	fi
	$(call PRINT_OK,gofmt clean)

.PHONY: fmt-fix
fmt-fix: ## Apply gofmt formatting
	$(call PRINT_STEP,gofmt fix)
	@if [ -n "$(GOFILES)" ]; then gofmt -w -s $(GOFILES); fi
	$(call PRINT_OK,gofmt applied)

.PHONY: imports-check
imports-check: $(GOIMPORTS) ## Check goimports formatting
	$(call PRINT_STEP,goimports check)
	@if [ -n "$(GOFILES)" ]; then \
		issues="$$($(GOIMPORTS) -l $(GOFILES))"; \
		if [ -n "$$issues" ]; then \
			printf '$(COLOR_FAIL)FAIL:$(COLOR_RESET) %s\n' 'goimports issues found (run make imports-fix or make fix)' >&2; \
			printf '%s\n' "$$issues"; \
			exit 1; \
		fi; \
	fi
	$(call PRINT_OK,goimports clean)

.PHONY: imports-fix
imports-fix: $(GOIMPORTS) ## Apply goimports formatting
	$(call PRINT_STEP,goimports fix)
	@if [ -n "$(GOFILES)" ]; then $(GOIMPORTS) -w $(GOFILES); fi
	$(call PRINT_OK,goimports applied)

.PHONY: tidy-check
tidy-check: ## Check go.mod/go.sum tidiness without leaving edits behind
	$(call PRINT_STEP,go mod tidy check)
	@tmpdir="$$(mktemp -d "$${TMPDIR:-/tmp}/tokless-tidy.XXXXXX")"; \
	had_sum=0; \
	cp go.mod "$$tmpdir/go.mod"; \
	if [ -f go.sum ]; then cp go.sum "$$tmpdir/go.sum"; had_sum=1; fi; \
	restore() { \
		cp "$$tmpdir/go.mod" go.mod; \
		if [ "$$had_sum" = 1 ]; then cp "$$tmpdir/go.sum" go.sum; else rm -f go.sum; fi; \
		rm -rf "$$tmpdir"; \
	}; \
	trap restore EXIT INT TERM; \
	$(GO) mod tidy; \
	status=0; \
	diff -q "$$tmpdir/go.mod" go.mod >/dev/null 2>&1 || status=1; \
	if [ "$$had_sum" = 1 ]; then \
		diff -q "$$tmpdir/go.sum" go.sum >/dev/null 2>&1 || status=1; \
	elif [ -f go.sum ]; then \
		status=1; \
	fi; \
	if [ "$$status" -ne 0 ]; then \
		printf '$(COLOR_FAIL)FAIL:$(COLOR_RESET) %s\n' 'go.mod/go.sum not tidy (run make tidy-fix or make fix)' >&2; \
		exit 1; \
	fi
	$(call PRINT_OK,go mod tidy clean)

.PHONY: tidy-fix
tidy-fix: ## Apply go mod tidy
	$(call PRINT_STEP,go mod tidy fix)
	@$(GO) mod tidy
	$(call PRINT_OK,go mod tidy applied)

.PHONY: scripts-check
scripts-check: sh-check ps-check ## Parse-check shell and PowerShell scripts where available

.PHONY: sh-check
sh-check: ## Parse-check shell scripts
	$(call PRINT_STEP,shell script syntax)
	@for script in scripts/*.sh; do \
		[ -e "$$script" ] || continue; \
		bash -n "$$script"; \
	done
	$(call PRINT_OK,shell scripts parse)

.PHONY: ps-check
ps-check: ## Parse-check PowerShell scripts when pwsh is available
	$(call PRINT_STEP,PowerShell script syntax)
	@if command -v pwsh >/dev/null 2>&1; then \
		pwsh -NoProfile -Command '$$errs=$$null; $$tokens=$$null; [System.Management.Automation.Language.Parser]::ParseFile("$(CURDIR)/scripts/install.ps1", [ref]$$tokens, [ref]$$errs) | Out-Null; if ($$errs) { $$errs | ForEach-Object { Write-Error $$_.Message }; exit 1 }; Write-Output "install.ps1 parses OK"'; \
	else \
		printf '$(COLOR_WARN)Skipping$(COLOR_RESET) %s\n' 'PowerShell parse check: pwsh not found'; \
	fi
	$(call PRINT_OK,PowerShell check complete)

.PHONY: diff-check
diff-check: ## Check for whitespace errors in the git diff
	$(call PRINT_STEP,git diff --check)
	@git diff --check
	$(call PRINT_OK,diff clean)

.PHONY: vet
vet: ## Run go vet
	$(call PRINT_STEP,go vet)
	@$(GO) vet $(PKGS)
	$(call PRINT_OK,go vet passed)

.PHONY: staticcheck
staticcheck: $(STATICCHECK) ## Run staticcheck
	$(call PRINT_STEP,staticcheck)
	@$(STATICCHECK) $(PKGS)
	$(call PRINT_OK,staticcheck passed)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Run golangci-lint
	$(call PRINT_STEP,golangci-lint)
	@$(GOLANGCI_LINT) run
	$(call PRINT_OK,golangci-lint passed)

.PHONY: errcheck
errcheck: $(ERRCHECK) ## Run errcheck
	$(call PRINT_STEP,errcheck)
	@$(ERRCHECK) $(PKGS)
	$(call PRINT_OK,errcheck passed)

.PHONY: gosec
gosec: $(GOSEC) ## Run gosec
	$(call PRINT_STEP,gosec)
	@$(GOSEC) -exclude-dir=.tools $(PKGS)
	$(call PRINT_OK,gosec passed)

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK) ## Run govulncheck
	$(call PRINT_STEP,govulncheck)
	@$(GOVULNCHECK) -test $(PKGS)
	$(call PRINT_OK,govulncheck passed)

.PHONY: test
test: ## Run tests
	$(call PRINT_STEP,go test)
	@$(GO) test -timeout $(TIMEOUT) $(PKGS)
	$(call PRINT_OK,tests passed)

.PHONY: race
race: ## Run the race detector where supported
	$(call PRINT_STEP,go test -race)
	@if [ "$$($(GO) env GOOS)" = "android" ]; then \
		printf '$(COLOR_WARN)Skipping$(COLOR_RESET) race detector: unsupported on %s/%s\n' "$$($(GO) env GOOS)" "$$($(GO) env GOARCH)"; \
	else \
		$(GO) test -race -timeout $(TIMEOUT) $(PKGS); \
		printf '$(COLOR_OK)OK:$(COLOR_RESET) %s\n' 'race detector clean'; \
	fi

.PHONY: build-all
build-all: ## Compile every package
	$(call PRINT_STEP,go build ./...)
	@$(GO) build $(PKGS)
	$(call PRINT_OK,packages build)

.PHONY: build
build: ## Build the tokless CLI binary
	$(call PRINT_STEP,go build)
	@$(GO) build -trimpath -ldflags "$(GO_LDFLAGS)" -o "$(BIN)" "$(CMD_PATH)"
	@printf '$(COLOR_OK)OK:$(COLOR_RESET) Build successful: %s\n' "$(BIN)"
	@printf '\n$(COLOR_DIM)Run:$(COLOR_RESET) ./%s\n' "$(BIN)"

.PHONY: smoke
smoke: build ## Build and smoke-test help/version output
	$(call PRINT_STEP,smoke test)
	@./$(BIN) --help >/dev/null
	@./$(BIN) --version >/dev/null
	$(call PRINT_OK,smoke test passed)

.PHONY: run
run: build ## Build and run tokless with ARGS="..."
	@./$(BIN) $(ARGS)

.PHONY: doctor
doctor: build ## Build and run offline doctor
	@./$(BIN) doctor --offline

.PHONY: install
install: ## Install tokless into GOBIN or GOPATH/bin
	$(call PRINT_STEP,go install)
	@$(GO) install -trimpath -ldflags "$(GO_LDFLAGS)" "$(CMD_PATH)"
	$(call PRINT_OK,tokless installed)

.PHONY: release
release: test ## Cross-compile release binaries into dist/release
	$(call PRINT_STEP,release build)
	@bash scripts/build-release.sh "$(VERSION)"

.PHONY: tools
tools: $(GOIMPORTS) $(STATICCHECK) $(GOLANGCI_LINT) $(ERRCHECK) $(GOSEC) $(GOVULNCHECK) ## Install local toolchain helpers

$(TOOLS_BIN):
	@mkdir -p "$@"

$(TOOLS_BIN)/goimports: | $(TOOLS_BIN)
	$(call PRINT_STEP,installing goimports)
	@env GOBIN="$(TOOLS_BIN)" $(GO) install $(GOIMPORTS_PKG)

$(TOOLS_BIN)/staticcheck: | $(TOOLS_BIN)
	$(call PRINT_STEP,installing staticcheck)
	@env GOBIN="$(TOOLS_BIN)" $(GO) install $(STATICCHECK_PKG)

$(TOOLS_BIN)/golangci-lint: | $(TOOLS_BIN)
	$(call PRINT_STEP,installing golangci-lint)
	@env GOBIN="$(TOOLS_BIN)" $(GO) install $(GOLANGCI_LINT_PKG)

$(TOOLS_BIN)/errcheck: | $(TOOLS_BIN)
	$(call PRINT_STEP,installing errcheck)
	@env GOBIN="$(TOOLS_BIN)" $(GO) install $(ERRCHECK_PKG)

$(TOOLS_BIN)/gosec: | $(TOOLS_BIN)
	$(call PRINT_STEP,installing gosec)
	@env GOBIN="$(TOOLS_BIN)" $(GO) install $(GOSEC_PKG)

$(TOOLS_BIN)/govulncheck: | $(TOOLS_BIN)
	$(call PRINT_STEP,installing govulncheck)
	@env GOBIN="$(TOOLS_BIN)" $(GO) install $(GOVULNCHECK_PKG)

.PHONY: clean
clean: ## Remove local build artifacts and tool cache
	$(call PRINT_STEP,clean)
	@rm -rf "$(TOOLS_DIR)" "$(BIN)" "$(DIST_DIR)" coverage.out coverage.html
	$(call PRINT_OK,clean complete)

.PHONY: help
help: ## Show Makefile targets
	@printf '$(COLOR_TITLE)%s$(COLOR_RESET)\n' 'tokless Development Commands'
	@printf '$(COLOR_DIM)%s$(COLOR_RESET)\n' '============================'
	@awk -v c='$(COLOR_TARGET)' -v r='$(COLOR_RESET)' 'BEGIN {FS = ":.*## "}; /^[A-Za-z0-9_.-]+:.*## / {printf "%s%-18s%s %s\n", c, $$1, r, $$2}' $(MAKEFILE_LIST) | sort
