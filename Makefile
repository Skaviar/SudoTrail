.DEFAULT_GOAL := help

GO ?= go
ADDR ?= 127.0.0.1:8080
DATABASE ?= sudoku-3m.csv
DIST ?= dist

WORKERS ?=
LIMIT ?= 0
OUTPUT ?= sudoku-rated.csv
PER_LEVEL ?= 500

.PHONY: help dev build web benchmark rate bank test race vet check fmt

help:
	@printf '%s\n' \
	  'Web:' \
	  '  make dev        Build WASM and run the web UI (ADDR=127.0.0.1:8080)' \
	  '  make build      Build dist/ for Cloudflare (DIST=dist)' \
	  '  make web        Run the Go API and embedded UI' \
	  '' \
	  'Puzzle tools:' \
	  '  make benchmark  Benchmark the CSV (WORKERS=N LIMIT=N DATABASE=path.csv)' \
	  '  make rate       Export difficulty ratings (WORKERS=N LIMIT=N OUTPUT=path.csv)' \
	  '  make bank       Rebuild the browser bank (PER_LEVEL=500 OUTPUT=sudoku-rated.csv)' \
	  '' \
	  'Checks:' \
	  '  make test       Run Go tests' \
	  '  make race       Run Go tests with the race detector' \
	  '  make vet        Run go vet' \
	  '  make check      Run tests and go vet' \
	  '  make fmt        Format Go source'

dev: build
	$(GO) run ./cmd/web -addr "$(ADDR)" -static "$(DIST)"

build:
	$(GO) run ./cmd/buildweb -out "$(DIST)"

web:
	$(GO) run ./cmd/web -addr "$(ADDR)" -database "$(DATABASE)"

benchmark:
	$(GO) run . -benchmark -database "$(DATABASE)" $(if $(WORKERS),-workers "$(WORKERS)") -limit "$(LIMIT)"

rate:
	$(GO) run . -rate -database "$(DATABASE)" -output "$(OUTPUT)" $(if $(WORKERS),-workers "$(WORKERS)") -limit "$(LIMIT)"

bank:
	$(GO) run ./cmd/buildbank -in "$(OUTPUT)" -per-level "$(PER_LEVEL)"

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

check: test vet

fmt:
	gofmt -w *.go cmd internal solver web
