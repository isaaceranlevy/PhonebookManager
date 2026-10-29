#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
go mod verify
go vet ./...
go test -race ./...
mkdir -p bin
go build -trimpath -o bin/phonebook ./cmd/phonebook
GOOS=windows GOARCH=amd64 go build -trimpath -o bin/phonebook.exe ./cmd/phonebook
