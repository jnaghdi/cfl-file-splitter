# Build and test

## Toolchains

The original module declares `go 1.23` and the supplied engine was first tested
with Go 1.23.2. For new distribution builds use a currently supported Go release
compatible with that language level; a minimum-language declaration is not a
recommendation to deploy an old toolchain. The workflows request `stable`,
record `go version`, and disable module caching because there is no `go.sum`.

The Python helper and tests require Python 3.9+; the CI matrix uses 3.13. No
third-party Python packages are required. Go and Python are needed to build/test,
not to run the generated Windows EXE files. Git and GitHub CLI are only needed
to publish/manage the repository.

Official downloads and documentation:
- https://go.dev/dl/
- https://www.python.org/downloads/
- https://git-scm.com/downloads
- https://cli.github.com/

## Windows PowerShell

```powershell
.\scripts\build.ps1
.\scripts\test.ps1 -Streaming
python .\scripts\package_release.py
```

Do not disable endpoint security or change machine-wide execution policy. If
scripts are blocked, use your organisation's approved review/unblocking process,
or execute these equivalent build commands individually:

```powershell
go test ./...
go vet ./internal/core ./cmd/cflcli
New-Item -ItemType Directory -Force bin | Out-Null
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -buildvcs=false -ldflags="-s -w" -o bin/CFL_Splitter_CLI.exe ./cmd/cflcli
go build -trimpath -buildvcs=false -ldflags="-s -w -H=windowsgui" -o bin/CFL_File_Splitter.exe ./cmd/cflsplit
```

The scripts restore environment changes when finished; when entering equivalent
commands manually, restore your previous environment settings afterwards.
The GUI targets Windows x64. ARM64 and 32-bit GUI builds are not included or
qualified. Unsupported operating systems do not get a GUI executable.

## Linux/macOS

```bash
bash scripts/test.sh --streaming
bash scripts/build.sh
python3 scripts/package_release.py
```

`build.sh` also writes a native CLI at `bin/cflsplit` for local tests. The Windows
builds use `CGO_ENABLED=0`, `GOOS=windows`, `GOARCH=amd64` and the Go standard
library. No cross C compiler is required for these executables.

## Integration tests

```text
python tests/repository_test.py
python tests/cross_language_test.py
python tests/auto_mode_test.py
python tests/streaming_test.py
```

Each interoperability/streaming script builds its own native CLI into a temporary
directory by default. Set `CFL_CLI` to the absolute path of a suitable **native**
CLI to reuse one. Never point Linux tests at a Windows EXE. Test sources/outputs
are temporary synthetic data, not anything from your cases.

The streaming case creates 256 MiB + 17 bytes and its encoded/reconstructed
copies; allow at least 1 GiB of temporary free disk space. It is a functional test,
not a throughput benchmark or multi-terabyte qualification.

## Distribution

`package_release.py` requires both Windows EXEs in bin/ and refuses to overwrite
an existing release ZIP. It creates a `CFL_File_Splitter_Windows_v1.0.1.zip` and
sidecar SHA-256 in dist/. The ZIP includes source, tests, scripts, documentation,
fictional examples, Go licence and helper. It excludes .git, credentials, caches,
build directories and any files not present in the source SHA-256 inventory.

Build metadata includes Go version and, when available, Git commit and dirty
state. `-trimpath`/`-buildvcs=false` reduce incidental path/VCS metadata, but builds
from different Go releases are **not** promised to be byte-identical.

For repository edits, regenerate SOURCE_SHA256SUMS.txt using
`python scripts/source_manifest.py --write` after reviewing the changed paths.
The publisher, tests and release packager verify that inventory.
