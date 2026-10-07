-include .env
export

.PHONY: dev test build

dev: web/node_modules
	@trap 'kill 0' EXIT; \
	go run ./cmd/mockapi & \
	go run ./cmd/server & \
	npm --prefix web run dev & \
	wait

test:
	go vet ./...
	go test ./...

build: web/node_modules
	go build ./...
	npm --prefix web run build

web/node_modules: web/package-lock.json
	npm --prefix web ci
	@touch $@
