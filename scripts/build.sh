#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
command -v go >/dev/null || { echo 'Install a supported Go toolchain.' >&2; exit 1; }
# Do not inherit a cross-compilation target for native tests.
unset GOOS GOARCH
export CGO_ENABLED=0
go version
go test ./...
go vet ./internal/core ./cmd/cflcli
mkdir -p bin
go build -trimpath -buildvcs=false -o bin/cflsplit ./cmd/cflcli
GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags='-s -w' -o bin/CFL_FileSplitter_CLI.exe ./cmd/cflcli
GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags='-s -w -H=windowsgui' -o "bin/CFL FileSplitter For Uploading Large Files To Claude.exe" ./cmd/cflsplit
printf '%s\n' 'Built native CLI and Windows x64 GUI/CLI in bin/. GUI runtime testing is still required.'
