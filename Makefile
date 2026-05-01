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
	@echo "TODO: edgar probe not implemented yet"
	@exit 1

probe-fx:
	@echo "TODO: fx probe not implemented yet"
	@exit 1

test:
	go test ./...

test-live: probe-yahoo
	@echo "Live probes complete. See testdata/ and docs/providers/ for findings."

tidy:
	go mod tidy

build:
	go build ./...
