.PHONY: build test race vet web-build run

build:
	go build ./...

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

web-build:
	cd web && npm run build

run:
	go run ./cmd/server

