.PHONY: dev backend frontend test build

backend:
	go run ./cmd/server

frontend:
	pnpm --dir web dev

test:
	go test ./...
	pnpm --dir web typecheck

build:
	pnpm --dir web build
	go build ./cmd/server

