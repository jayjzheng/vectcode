.PHONY: build install test clean run-index run-query run-list

build:
	go build -o codegraph ./cmd/codegraph

install:
	go install ./cmd/codegraph

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f codegraph coverage.out coverage.html

fmt:
	go fmt ./...
	go vet ./...

deps:
	go mod download
	go mod tidy

run-index:
	go run ./cmd/codegraph index --path ~/projects/example --name example

run-query:
	go run ./cmd/codegraph query --query "where is the authentication handler?"

run-list:
	go run ./cmd/codegraph list

dev: build
	./codegraph --help
