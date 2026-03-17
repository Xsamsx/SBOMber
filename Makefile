APP_NAME := sbomber

.PHONY: build test vet fmt tidy ci

build:
	go build -o ./bin/$(APP_NAME) ./cmd/$(APP_NAME)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w ./cmd ./internal

tidy:
	go mod tidy

ci: fmt vet test
