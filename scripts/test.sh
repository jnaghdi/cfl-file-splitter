#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
if [[ $# -gt 1 || ( $# -eq 1 && "$1" != '--streaming' ) ]]; then
  echo 'Usage: bash scripts/test.sh [--streaming]' >&2; exit 2
fi
unset GOOS GOARCH
export PYTHONUTF8=1
go version
go test -count=1 ./...
go vet ./internal/core ./cmd/cflcli
python3 tests/repository_test.py
python3 tests/cross_language_test.py
python3 tests/auto_mode_test.py
if [[ "${1:-}" == '--streaming' ]]; then python3 tests/streaming_test.py; fi
printf '%s\n' 'Requested automated tests passed. GUI/Claude tests require their actual environments.'
