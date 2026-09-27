.PHONY: lint test tidy vendor up

lint:
	golangci-lint run

test:
	go test ./...

tidy:
	go mod tidy

vendor:
	go mod vendor

up:
	docker compose up --build
