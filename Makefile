default: build

test:
	go test

build:
	go build -o bin/azure-dns-manager
