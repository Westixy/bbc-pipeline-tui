.PHONY: build run test lint clean tidy

# Default target
build:
	go build -o infra-pipeline-ui .

run: build
	./infra-pipeline-ui

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

clean:
	rm -f infra-pipeline-ui coverage.out

fmt:
	go fmt ./...

vet:
	go vet ./...

all: fmt vet test build