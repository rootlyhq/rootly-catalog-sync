.PHONY: build test lint vet clean snapshot

build:
	go build -o rootly-catalog-sync ./cmd/rootly-catalog-sync

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run

clean:
	rm -f rootly-catalog-sync
	rm -rf dist/

snapshot:
	goreleaser release --snapshot --clean
