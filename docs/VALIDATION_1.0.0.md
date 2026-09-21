# Repository validation — 18 September 2026

## Scope and environment

The supplied Windows v1.0 ZIP was used as the source. Its Go module was moved to
this repository root and formatted with gofmt; application behaviour was not
intentionally changed. Python integration tests were adapted to use a host-native
CLI rather than a fixed Windows EXE path. Source/docs/build automation were added.

Checks here ran on **Linux x86-64, Go 1.23.2 and Python 3.13.5**. New distribution
builds should use a maintained Go toolchain. GitHub workflow definitions request
`stable` Go, but those hosted jobs have not run during this preparation.

## Executed checks

| Check | Result / evidence |
|---|---|
| Go tests (`go test -count=1 -v ./...`) | Passed; 18 top-level engine tests with additional subcases. [Output](test-results/go-tests.txt). |
| Engine race detector (`go test -race -count=1 ./internal/core`) | Passed in this environment. [Output](test-results/go-race.txt). |
| Engine/CLI vet (`go vet ./internal/core ./cmd/cflcli`) | Passed. [Output](test-results/go-vet.txt). |
| Go/Python interoperability and rejection paths | 25 scenarios passed. [Output](test-results/cross-language.txt). |
| Larger synthetic file | 268,435,473 bytes split into 18 encoded TXT pieces; every piece within 20,000,000 bytes. Python verification and Go reconstruction matched original size and SHA-256. [Output](test-results/streaming.txt). |
| Windows x64 GUI and CLI | Cross-compiled successfully; PE32+ x86-64 target inspected. [Output](test-results/windows-cross-build.txt). |
| Repository checks | 10 standard-library checks passed: inventory, helper identity, version, both demo round trips, byte-preserving attributes, workflow pins/token scope, private publisher defaults, required files, blocked artifact types and unsafe inventory paths. |
| YAML / Python / Bash syntax | Workflow and issue YAML parsed; Python sources compiled; Bash scripts passed `bash -n`. This is not a hosted Actions execution or PowerShell runtime test. |

Synthetic streaming source SHA-256:

```text
e947673df05b471e03768ce3c3c8090371b01c9aeaeeb7756afe47965646a9d2
```

## Known static-analysis warnings

A separate full Windows-target `go vet ./...` is **not clean**. It reports two
possible `unsafe.Pointer` misuse warnings in the pre-existing native GUI bridge:
clipboard global-memory access and WM_GETMINMAXINFO callback handling. They are
recorded in [windows-gui-vet-warnings.txt](test-results/windows-gui-vet-warnings.txt).
The automated vet gate intentionally covers the engine and CLI, not these Win32
conversions. No warning suppression or claim of full-GUI static-analysis success
is made. These sites require review and Windows runtime validation.

## Not performed / not claimed

No interactive Windows GUI test, Windows-specific filesystem qualification,
actual Claude attachment upload/reconstruction, PowerShell script execution,
authenticated GitHub repository creation, GitHub Actions run, independent security
audit, Authenticode signing, multi-terabyte qualification or forensic certification
was performed here. The publishing and release scripts are supplied for review
and execution in their intended authenticated environment.

The original release's own report is retained under
[original-release/TEST_REPORT.txt](original-release/TEST_REPORT.txt); it is not a
substitute for the fresh results above. Do not conflate a successful compile with
interactive testing or hash-valid reconstruction with a complete substantive
review of a document's contents.
