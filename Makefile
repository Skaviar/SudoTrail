.DEFAULT_GOAL := help

GO ?= go
ADDR ?= 127.0.0.1:8080
DATABASE ?= sudoku-3m.csv
DIST ?= dist
WORKERS ?=
LIMIT ?= 0
OUTPUT ?= sudoku-rated.csv

.PHONY: help run benchmark rate web build preview dev test race vet check fmt

help:
	@printf '%s\n' \
	  'Godoku commands:' \
	  '  make dev      Build WASM and start the static website locally' \
	  '  make build    Build the static website into dist/ for Cloudflare' \
	  '  make preview  Build WASM and preview it locally (same as dev)' \
	  '  make web      Run the Go HTTP API and website' \
	  '  make run      Run the terminal app (including benchmark menu)' \
	  '  make benchmark Run the CSV benchmark (WORKERS=N LIMIT=N)' \
	  '  make rate     Export difficulty ratings (WORKERS=N LIMIT=N OUTPUT=path.csv)' \
	  '  make test     Run Go tests' \
	  '  make race     Run Go tests with the race detector' \
	  '  make vet      Run go vet' \
	  '  make check    Run tests and go vet' \
	  '  make fmt      Format Go source files' \
	  '' \
	  'Options: ADDR=127.0.0.1:9090 DIST=dist DATABASE=sudoku-3m.csv WORKERS=8 LIMIT=10000 OUTPUT=sudoku-rated.csv'

run:
	$(GO) run . -database "$(DATABASE)" -output "$(OUTPUT)" $(if $(WORKERS),-workers "$(WORKERS)") -limit "$(LIMIT)"

benchmark:
	$(GO) run . -benchmark -database "$(DATABASE)" $(if $(WORKERS),-workers "$(WORKERS)") -limit "$(LIMIT)"

rate:
	$(GO) run . -rate -database "$(DATABASE)" -output "$(OUTPUT)" $(if $(WORKERS),-workers "$(WORKERS)") -limit "$(LIMIT)"

web:
	$(GO) run ./cmd/web -addr "$(ADDR)" -database "$(DATABASE)"

build:
	$(GO) run ./cmd/buildweb -out "$(DIST)"

# Rebuild before every preview so embedded Go and frontend changes stay in sync.
preview: build
	$(GO) run ./cmd/web -addr "$(ADDR)" -static "$(DIST)"

dev: preview

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

check: test vet

fmt:
	gofmt -w main.go cli.go benchmark.go benchmark_test.go rating.go rating_test.go cmd internal solver web
