SHELL := /bin/bash

MODULE := github.com/thomaslaurenson/cellmate
BINARY := cellmate
VERSION := $(shell git describe --tags --always --dirty --match 'v*' 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(MODULE)/cmd.Version=$(VERSION)
# The browser build does not link cmd, so it carries its own version in main
WASM_LDFLAGS := -s -w -X main.version=$(VERSION)
GOIMPORTS := go run golang.org/x/tools/cmd/goimports@latest -local $(MODULE)
# Inlining duplicates code, and duplicated code compresses badly. Turning it
# off grows the .wasm by 9% but shrinks what the browser actually downloads by
# 4%, and a patience game has frame time to spare. Browser build only.
WASM_GCFLAGS := -gcflags=all=-l
# The budget check_wasm_size holds the browser build to, compressed the way
# GitHub Pages serves it. It is deliberately tight: the size regressions worth
# catching are whole standard library packages arriving unnoticed behind a new
# feature, and any one of those costs more than the headroom here. Raise it
# deliberately, in the commit that needs the room.
WASM_MAX_BYTES ?= 1675000
COVERPKG := ./internal/...
WEB_DIR := dist/web

##@ BUILD

.PHONY: help
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^##@ / {printf "\n%s\n", substr($$0, 5)} \
		/^[a-zA-Z_-]+:.*## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the desktop binary into dist/
	go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY) .

.PHONY: build_wasm
build_wasm: ## Build the browser version into dist/web/
	mkdir -p $(WEB_DIR)
	GOOS=js GOARCH=wasm go build -trimpath $(WASM_GCFLAGS) -ldflags="$(WASM_LDFLAGS)" \
		-o $(WEB_DIR)/$(BINARY).wasm .
	cp web/index.html $(WEB_DIR)/
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" $(WEB_DIR)/

.PHONY: serve_wasm
serve_wasm: build_wasm ## Serve the browser version on http://localhost:8080
	go run ./internal/devserve -dir $(WEB_DIR)

.PHONY: snapshot
snapshot: ## Build every release platform locally with goreleaser
	goreleaser build --snapshot --clean

##@ TEST

.PHONY: test
test: ## Run unit and functional tests
	go test -race -count=1 ./...

# The browser half of internal/gfx is behind a js build tag, so an ordinary
# go test never compiles it, let alone runs it. This target does both, for
# internal/gfx alone. The other browser-only code, main_js.go and the local
# storage store, needs a page that Node does not have, as gfx's drawing code
# does; what that code decides lives in untagged files the ordinary run tests.
.PHONY: test_wasm
test_wasm: ## Run the browser build's own tests under Node
	GOOS=js GOARCH=wasm go test -count=1 \
		-exec="$$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./internal/gfx/

.PHONY: test_integration
test_integration: ## Run tests that need resources CI cannot provide
	go test -race -count=1 -tags=integration ./...

.PHONY: test_coverage
test_coverage: ## Report test coverage of the game logic
	go test -race -count=1 -tags=integration -coverpkg=$(COVERPKG) -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	rm coverage.out

##@ LINT

.PHONY: format
format: ## Format source and group imports
	$(GOIMPORTS) -w .

.PHONY: check_format
check_format: ## Fail if any file needs formatting
	@out="$$($(GOIMPORTS) -l .)"; \
	if [[ -n "$$out" ]]; then printf 'needs formatting:\n%s\n' "$$out" >&2; exit 1; fi

.PHONY: check_mod
check_mod: ## Fail if go.mod or go.sum is not tidy
	go mod tidy && git diff --exit-code go.mod go.sum

.PHONY: vet
vet: ## Run go vet, including integration-tagged files
	go vet ./...
	go vet -tags=integration ./...

.PHONY: check_cross
check_cross: ## Type-check the windows, darwin and browser builds
	GOOS=windows go vet ./...
	GOOS=darwin go vet ./...
	GOOS=js GOARCH=wasm go vet ./...

.PHONY: check_wasm_size
check_wasm_size: build_wasm ## Fail if the browser build is over WASM_MAX_BYTES
	@size="$$(gzip -9 -c "$(WEB_DIR)/$(BINARY).wasm" | wc -c)"; \
	printf 'browser build: %s bytes gzipped, budget %s\n' "$$size" "$(WASM_MAX_BYTES)"; \
	if (( size > $(WASM_MAX_BYTES) )); then \
	  printf 'check_wasm_size: over budget by %s bytes; find what was linked in with\n' \
	    "$$(( size - $(WASM_MAX_BYTES) ))" >&2; \
	  printf '  GOOS=js GOARCH=wasm go list -deps . | grep -vE "^(internal/|$(MODULE))"\n' >&2; \
	  exit 1; \
	fi

# Both builds embed the card drawings. The compiler only proves the files are
# there; this proves every card has one, that the rasteriser draws each, and
# that every drawing is on the card palette.
.PHONY: check_embed
check_embed: ## Check the embedded card drawings
	go test -count=1 -run '^Test(EveryCard|CardFaces|NewCardSprites)' ./internal/ui/
	@out="$$(go run ./internal/recolour -n)"; \
	printf '%s\n' "$$out"; \
	if ! grep -q ', 0 repainted' <<<"$$out"; then \
	  printf 'check_embed: drawings are off the palette; run make gen_card_colours\n' >&2; \
	  exit 1; \
	fi

.PHONY: check_all
check_all: check_format check_mod vet check_cross check_embed check_wasm_size ## Run static checks

.PHONY: vuln
vuln: ## Scan for reachable vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

##@ GENERATE

# The deck was cut from a drawing made in an EGA palette. This repaints it
# onto card colours and is safe to run again: a deck already painted comes
# out of it unchanged, so re-cutting a card from source needs no more than
# another run.
.PHONY: gen_card_colours
gen_card_colours: ## Repaint the card drawings onto the card palette
	go run ./internal/recolour

##@ GET

.PHONY: get_version
get_version: ## Print the version a build would stamp
	@echo "$(VERSION)"

.PHONY: get_changelog
get_changelog: ## Print release notes for TAG to stdout (TAG=v1.0.0)
	@tag="$(TAG)"; tag="$${tag#v}"; \
	if [[ -z "$$tag" ]]; then \
	  printf 'get_changelog: TAG is empty; pass TAG=v1.0.0\n' >&2; \
	  exit 1; \
	fi; \
	notes="$$(awk -v tag="$$tag" ' \
	  /^## / { if (found) exit; if (index($$0,"## "tag" ")==1 || $$0=="## "tag) found=1; next } \
	  found { lines[n++]=$$0 } \
	  END { \
	    s=0; while (s<n && lines[s]~/^[[:space:]]*$$/) s++; \
	    e=n-1; while (e>=s && lines[e]~/^[[:space:]]*$$/) e--; \
	    for (i=s;i<=e;i++) print lines[i] \
	  }' CHANGELOG.md)"; \
	if [[ -z "$$notes" ]]; then \
	  printf 'get_changelog: no CHANGELOG entry for %s\n' "$$tag" >&2; \
	  exit 1; \
	fi; \
	printf '%s\n' "$$notes"

##@ CI

.PHONY: ci
ci: check_all test test_wasm ## Run everything CI runs

.PHONY: clean
clean: ## Remove build output and release artefacts
	rm -rf dist/ coverage.out install.sh install.ps1 checksums.txt checksums.txt.sigstore.json
