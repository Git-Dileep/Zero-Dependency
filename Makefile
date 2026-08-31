.PHONY: build test clean demo deps-proof

build:
	go build -o miniratchet ./cmd/miniratchet

test:
	go test ./...

demo:
	go run ./cmd/miniratchet --demo all

deps-proof:
	@echo "=== go list -m all ==="
	@go list -m all
	@echo "=== go mod verify ==="
	@go mod verify

clean:
	rm -f miniratchet miniratchet_1 miniratchet_2
