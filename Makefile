BIN := bin/assaio-agent
LDFLAGS := -X github.com/assaio/assaio/internal/version.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
FUZZTIME := 20s

.PHONY: build test lint fmt tidy snapshot hooks vuln fuzz
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/assaio-agent
test:
	go test ./...
# Runs each parser's fuzzer sequentially for FUZZTIME (default 20s).
# Longer runs: make fuzz FUZZTIME=5m
fuzz:
	go test ./internal/parser/claude/ -fuzz FuzzParse -fuzztime $(FUZZTIME)
	go test ./internal/parser/codex/ -fuzz FuzzParse -fuzztime $(FUZZTIME)
	go test ./internal/parser/gemini/ -fuzz FuzzParse -fuzztime $(FUZZTIME)
	go test ./internal/parser/cline/ -fuzz FuzzParseTask -fuzztime $(FUZZTIME)
	go test ./internal/plugin/ -fuzz FuzzMetricResult -fuzztime $(FUZZTIME)
lint:
	gofmt -l .
	go vet ./...
	golangci-lint run
fmt:
	golangci-lint fmt
tidy:
	go mod tidy
# Requires goreleaser installed locally (https://goreleaser.com/install/).
snapshot:
	goreleaser release --snapshot --clean
# Opt-in. Requires lefthook: `brew install lefthook` or
# `go install github.com/evilmartians/lefthook@latest`.
hooks:
	lefthook install
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# --- releasing (see RELEASING.md) ---
LATEST_TAG = $(shell git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)

release-patch: ## tag next patch release (requires CONFIRM=yes)
	@$(MAKE) release VERSION=$(shell echo $(LATEST_TAG) | awk -F. '{printf "%s.%s.%d", $$1, $$2, $$3+1}') CONFIRM=$(CONFIRM)

release-minor: ## tag next minor release (requires CONFIRM=yes)
	@$(MAKE) release VERSION=$(shell echo $(LATEST_TAG) | awk -F. '{printf "%s.%d.0", $$1, $$2+1}') CONFIRM=$(CONFIRM)

release: ## tag VERSION=vX.Y.Z (requires CONFIRM=yes); push the tag to trigger the release workflow
ifndef VERSION
	$(error VERSION is required, e.g. make release VERSION=v0.2.0 CONFIRM=yes)
endif
ifneq ($(CONFIRM),yes)
	@echo "Would tag $(VERSION) (latest: $(LATEST_TAG)). Re-run with CONFIRM=yes."
else
	@test -z "$$(git status --porcelain)" || { echo "working tree not clean"; exit 1; }
	@grep -q "^\#\# \[$(patsubst v%,%,$(VERSION))\]" CHANGELOG.md || { \
		echo "CHANGELOG.md has no '## [$(patsubst v%,%,$(VERSION))]' section."; \
		echo "Move the [Unreleased] entries under it first -- see RELEASING.md."; exit 1; }
	@awk '/^\#\# \[Unreleased\]/{f=1;next} /^\#\# \[/{f=0} f && /^- /{found=1} END{exit found}' CHANGELOG.md || { \
		echo "CHANGELOG.md [Unreleased] still holds entries; they would be lost from $(VERSION)'s notes."; \
		echo "Move them under '## [$(patsubst v%,%,$(VERSION))]' first -- see RELEASING.md."; exit 1; }
	$(MAKE) test lint
	CGO_ENABLED=0 go build ./...
	git tag -a $(VERSION) -m "release $(VERSION)"
	@echo "Tagged $(VERSION). Push it with: git push origin $(VERSION)"
endif
