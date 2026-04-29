.PHONY: build run test lint clean

BINARY=grablenta
CONFIG=config.yaml

build:
	go build -o $(BINARY) ./cmd/parser

run: build
	PROXY_URL=$(PROXY_URL) ./$(BINARY) -config $(CONFIG)

test:
	go test ./... -v -race -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	@which golangci-lint > /dev/null || (echo "install golangci-lint first" && exit 1)
	golangci-lint run ./...

clean:
	rm -f $(BINARY) products.csv parser.log coverage.out coverage.html
