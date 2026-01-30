.PHONY: build install test clean run-index run-query run-list

build:
	go build -o vectcode ./cmd/vectcode

install:
	go install ./cmd/vectcode

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f vectcode coverage.out coverage.html

fmt:
	go fmt ./...
	go vet ./...

deps:
	go mod download
	go mod tidy

run-index:
	go run ./cmd/vectcode index --path ~/projects/example --name example

run-query:
	go run ./cmd/vectcode query --query "where is the authentication handler?"

run-list:
	go run ./cmd/vectcode list

dev: build
	./vectcode --help
