#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p bin
GOOS=windows GOARCH=amd64 go build -trimpath -o bin/phonebook.exe ./cmd/phonebook
python3 scripts/render-guides.py
"${MAKENSIS:-makensis}" -V3 packaging/phonebook.nsi
