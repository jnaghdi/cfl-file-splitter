# Validation — Windows 1.0.2

## Release identification

**CFL FileSplitter For Uploading Large Files To Claude** 1.0.2<br>
**Licence:** MIT — Copyright (c) 2026 Computer Forensics Lab Ltd<br>
**Websites:** https://cflab.uk and https://e-discovery.uk<br>
**Validation date:** 21 September 2026<br>
**On-disk format:** CFLSPLIT/1, unchanged

This Windows-focused refresh starts from the supplied Windows v1.0.1 source ZIP.
The baseline was imported as an explicitly labelled local snapshot, not as a
reconstruction of prior development history. The core, Windows GUI and Python
helper in that ZIP matched those in the later combined Windows/Mac bundle before
modification. The earlier Mac distribution has not been rebuilt or replaced.

## Executed checks

| Check | Result and scope |
|---|---|
| Go tests | **27 top-level tests passed**, including further subtests. |
| Go race detector | Passed for the engine on Linux. |
| Go static analysis | Engine and CLI passed; Windows GUI findings are disclosed below. |
| Cross-language reconstruction | **25 Go/Python scenarios passed**, including rejection of corrupt, missing, duplicate and mixed pieces. |
| Encoding regression | **23 scenarios passed**, including NUL bytes, UTF-16/UTF-32, invalid UTF-8, valid Unicode and strict-readable rejection. |
| Branding and licensing | **9 tests passed** for the exact name, both websites, full MIT notice, CLI flags, helper synchronization and UTF-16 round trip. |
| Repository checks | **10 tests passed** for the inventory, fixtures, helper synchronization, version, publish-script safeguards and workflow configuration. |
| Synthetic streaming | **268,435,473 bytes** split into **18 encoded TXT pieces**, each at most **20,000,000 bytes**; independent Python verification and byte-exact Go reconstruction passed. |
| Windows build | GUI and CLI compiled as Windows x64 PE executables. GUI/console subsystem fields verified. |
| Windows package inspection | **24 static checks passed**: safe unique paths, ZIP CRCs, sidecar/internal/source hashes, MIT contents, helper copies, actual compiled branding strings, metadata and PE architecture. |

The larger test's original and reconstructed SHA-256 were:

```text
e947673df05b471e03768ce3c3c8090371b01c9aeaeeb7756afe47965646a9d2
```

The Python helper's executable abstract syntax tree is identical to the previous
helper; only the full MIT comment block was added. `internal/core/core.go` is
byte-identical to its baseline after normalising the version constant and
handover heading. The splitting, ordering, hashing, verification and joining
algorithms have not been changed. Existing fictional split sets still verify
and reconstruct successfully in the new CLI.

Logs for this release are under [`test-results/v1.0.2/`](test-results/v1.0.2/).
Earlier reports and test logs remain explicitly historical. A static package
check does not execute a Windows program.

## Build environment and unresolved checks

The build/test host was Linux x86-64, with **Go 1.23.2** and Python 3.13.5.
Go 1.23.2 is the available compiler, not a currently maintained production
recommendation. Fetching a newer toolchain was unavailable in this environment.
The GitHub workflows request `stable`; use a maintained Go release and run the
Windows tests before production distribution. `BUILD_INFO.json` records the
actual packaging/compiler and binary metadata rather than claiming another
compiler was used.

The native Windows GUI retains **two pre-existing `go vet` unsafe.Pointer
findings**, at clipboard-memory access and the WM_GETMINMAXINFO callback.
They are not silently counted as a clean GUI static-analysis result. Review
[`windows-gui-vet-warnings.txt`](test-results/v1.0.2/windows-gui-vet-warnings.txt)
and validate those paths on Windows; this release makes no security-audit claim.

**Not performed here:** interactive Windows GUI testing, DPI/accessibility
qualification, execution of the PowerShell scripts on Windows, tests inside a
Claude account, actual GitHub Actions runs, Authenticode signing, antivirus
certification or independent security audit. No Mac build is included in this
Windows release. The EXEs are unsigned development binaries.

## Before routine use

Extract into a new folder, start with fictional demo data and follow your normal
software-review process. Confirm the full new title, v1.0.2 and About & licence.
Check header/footer layout at your display scale. Test Auto with the supplied
UTF-16 example, the explicit readable-mode retry, file/folder selection, drag and
drop, Cancel, Verify, Rejoin, no-overwrite and copying instructions. Do not
disable endpoint protections to run an unsigned application.

A successful local join does not guarantee that an upload will be accepted or
that Claude can analyse the recovered format. Only file content is preserved;
this is not a forensic acquisition or metadata-preservation utility.

## Licensing and publication

The project root LICENSE and embedded licence carry MIT text. A complete MIT
notice is also inside each standalone Python helper. The Go runtime's existing
licence is retained separately. The new README links both requested websites.
No current project document leaves the licensing decision pending. Historical
release reports remain history, not the licence for this release.

Preparing the source ZIP and offline Git bundle does not create or update a live
GitHub repository. Review the source before publishing; use the existing-repository
guidance rather than force-pushing or removing unrelated Mac source files.
