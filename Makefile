BINARY := read_lsp

.PHONY: build test race vet fmt install clean

build:
	go build -o $(BINARY) .

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w main.go analysis/*.go diagnostics/*.go lsp/*.go rpc/*.go server/*.go

install:
	go install .

clean:
	rm -f $(BINARY)
