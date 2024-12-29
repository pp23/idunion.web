.PHONY: lint test vendor clean

export GO111MODULE=on
GOCMD=go

default: lint build test

lint:
	golangci-lint run

build: ## Build your project and put the output binary in out/bin/
	mkdir -p out/bin
	GO111MODULE=on $(GOCMD) build -o out/bin/ ./cmd/htmx

build-release: ## Build your project as standalone binary without symbol table and debug information
	mkdir -p out/bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on $(GOCMD) build -ldflags="-w -s" -o out/bin/ ./cmd/htmx

test:
	go test -v -cover ./...

vendor:
	go mod vendor

clean:
	rm -rf ./vendor
