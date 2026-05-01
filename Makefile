.PHONY: help probe-yahoo probe-edgar probe-fx test test-live tidy build

help:
	@echo "Targets:"
	@echo "  probe-yahoo   Hit Yahoo Finance and capture fixtures (testdata/yahoo)"
	@echo "  probe-edgar   Hit SEC EDGAR and capture fixtures (testdata/edgar)"
	@echo "  probe-fx      Hit FX provider and capture fixtures (testdata/fx)"
	@echo "  test          Run unit + integration tests (no live network)"
	@echo "  test-live     Re-run all probes against live APIs (drift detection)"
	@echo "  tidy          go mod tidy"
	@echo "  build         Build all commands"

probe-yahoo:
	go run ./cmd/probes/yahoo

probe-edgar:
	@if [ -z "$$EDGAR_CONTACT_EMAIL" ]; then \
		echo "EDGAR_CONTACT_EMAIL is required (SEC enforces a User-Agent containing it)"; \
		echo "  e.g.: EDGAR_CONTACT_EMAIL=you@example.com make probe-edgar"; \
		exit 1; \
	fi
	go run ./cmd/probes/edgar

probe-fx:
	go run ./cmd/probes/fx

test:
	go test ./...

test-live: probe-yahoo probe-edgar probe-fx
	@echo "Live probes complete. See testdata/ and docs/providers/ for findings."

tidy:
	go mod tidy

build:
	go build ./...
