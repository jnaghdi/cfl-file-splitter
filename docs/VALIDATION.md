# CFL File Splitter 1.0.1 validation — 19 September 2026

## Scope

This repair addresses the screenshot error **“NUL byte found: use encoded-text
mode for binary or UTF-16 data.”** The v1.0.0 source package was inspected. Its
Windows file chooser selected strict UTF-8 for text-like extensions without
examining content. The repair adds content-validated Auto selection and an
explicit encoded retry in the strict-readable GUI path.

The user's original file was not provided. The error was reproduced using the
old engine path and a harmless supplied UTF-16LE example, not by running the
user's data or the Windows GUI. New Auto splitting and rejoining of the same
example preserved every content byte and its independently calculated SHA-256.

## Environment

Linux x86-64, installed **Go 1.23.2**, Python 3.13.5. See
[environment.txt](test-results/v1.0.1/environment.txt). The same installed Go
version as the preceding build was used; this is not a claim that the compiler
is current. Rebuild reviewed production distributions with a maintained Go
toolchain. An attempt to reach the official download endpoint from the build
container failed DNS resolution; no newer toolchain was downloaded here.

## Executed checks

| Check | Result / evidence |
|---|---|
| Reported-error reproduction | Old strict-readable engine rejected the UTF-16 sample with the matching NUL error; v1.0.1 Auto used Base64 and reconstructed exact original bytes. [Log](test-results/v1.0.1/reproduce-nul.txt). |
| Go tests | 25 top-level tests passed, including 7 new Auto/compatibility test functions; 72 named outcomes including subtests. [Log](test-results/v1.0.1/go-tests.txt). |
| New Auto-mode integration coverage | 23 actual CLI/Python scenarios passed, including UTF-16LE/BE with and without BOM, UTF-32, Windows-1252, NUL/invalid bytes beyond 3 MiB, incomplete characters, strict errors and the new CLI default. [Log](test-results/v1.0.1/auto-mode.txt). |
| Existing Go/Python interoperability | All 25 prior cross-language/rejection scenarios passed. [Log](test-results/v1.0.1/cross-language.txt). |
| Backward compatibility | Python helper unchanged byte-for-byte; v1.0.0 Go joiner reconstructed both new readable and Base64 Auto outputs. [Log](test-results/v1.0.1/backward-compatibility.txt). |
| Race detection | Engine tests passed with `go test -race -count=1 ./internal/core`. [Log](test-results/v1.0.1/go-race.txt). |
| Engine/CLI vet | Passed without diagnostics. [Log](test-results/v1.0.1/go-vet.txt). |
| Larger synthetic file | 268,435,473 source bytes, 18 encoded TXT pieces, each at most 20,000,000 bytes; Python verification and Go reconstruction matched size/hash. [Log](test-results/v1.0.1/streaming.txt). |
| Windows x64 binaries | GUI and CLI cross-compiled successfully; PE32+ x86-64 GUI/console targets inspected. [Log](test-results/v1.0.1/windows-cross-build.txt). |
| Repository checks | Inventory, helper identity, version consistency, old fictional demo reconstruction, source policies and existing workflow/publisher guards checked using `tests/repository_test.py`. All 10 checks passed. [Log](test-results/v1.0.1/repository-checks.txt). |
| Source syntax | Python parsing, Bash syntax, Go formatting and workflow/issue YAML parsing checked. [Log](test-results/v1.0.1/syntax-checks.txt). |

The synthetic UTF-16 source hash is:

```text
5b8443e838c725af8b3db47498362fc6395d58735acef33f93623b4b8ca52010
```

The larger synthetic source hash is:

```text
e947673df05b471e03768ce3c3c8090371b01c9aeaeeb7756afe47965646a9d2
```

## Unresolved checks and limitations

Full Windows-target `go vet ./...` still reports the two pre-existing
`unsafe.Pointer` warnings in clipboard global-memory access and the
WM_GETMINMAXINFO callback bridge. The current line numbers are captured in
[windows-gui-vet-warnings.txt](test-results/v1.0.1/windows-gui-vet-warnings.txt).
These were not suppressed, and a clean full-GUI vet result is not claimed.
The engine/CLI vet gate is separate from these native API bridge warnings.

No interactive Windows GUI test (including the retry dialog), actual source-file
test from the user, Claude upload, PowerShell execution, GitHub Actions run,
remote repository update, Authenticode signing, independent security audit,
multi-terabyte qualification or forensic certification was performed here.
Cross-compilation does not prove interactive GUI operation. Hash-valid
reconstruction does not establish complete substantive document processing.

Auto preserves source content; it is not a universal file-format/encoding
classifier. A file that is NUL-free valid UTF-8 is considered readable transport
without interpreting its semantics. Sources failing that compatibility test
are encoded rather than altered. Known UTF-16 or binary data can be placed in
Encoded text directly to avoid a failed initial readable scan.

The historic v1.0.0 report is retained as
[VALIDATION_1.0.0.md](VALIDATION_1.0.0.md). Older logs under `test-results/` and
`original-release/` are historical; the new results are in `test-results/v1.0.1/`.
