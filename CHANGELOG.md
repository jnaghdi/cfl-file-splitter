# Changelog

## 1.0.2 — 2026-09-21

- Renamed the Windows application to **CFL FileSplitter For Uploading Large Files To Claude**. The full name is used in the window title, header, EXE filename, help/about dialog, CLI identification and new Claude handover instructions.
- Added **https://cflab.uk** and **https://e-discovery.uk** to README.md, the About dialog and new handovers.
- Adopted the **MIT License** for project code at the owner's request. Retained the separate Go runtime licence. The standalone/embedded Python helper now includes the full MIT notice.
- Added About & licence, CLI `--version`, CLI `--help` and CLI `--license`.
- Retained the Auto/NUL-byte/UTF-16 fix, the CFLSPLIT/1 format, all integrity checks and the no-overwrite policy. Only branding/version text changed in the core algorithm file.
- Rebuilt the Windows x64 distribution. This is a Windows-only refresh of the supplied Windows v1.0.1 source snapshot; it does not replace the earlier Mac distribution or its repository.
- Windows runtime testing, Claude upload testing and GitHub Actions execution still require those environments. See docs/VALIDATION.md.

## 1.0.1 — 2026-09-19

### Fixed

- Removed extension-based assumptions that `.txt` and `.log` imply UTF-8.
- Added Auto as the GUI and CLI default. It validates the full source as
  NUL-free UTF-8, falling back to Base64 on text-compatibility errors only.
- Preserved every byte, including NULs, byte-order marks and original encoding;
  no lossy conversion or evidence alteration is used to silence the error.
- Classified text-compatibility errors so I/O failures, cancellation and integrity
  errors are not masked by the fallback.
- Added an explicit encoded retry prompt when strict readable mode is selected.
- Clear previous-output links when a new split starts to avoid stale instructions.
- Report actual selected encoding and display the current version in the GUI.
- Stop recommending TXT uploads after generating binary `.cflpart` output.

### Tests and compatibility

- Added full-content, late-NUL, invalid/truncated UTF-8, UTF-16/UTF-32,
  Windows-1252, streaming-boundary, cancellation and byte-exact reconstruction
  regression coverage. See the current validation report for executed results.
- Added a harmless UTF-16LE fixture for reproducing the reported error.
- Added Auto-mode regression tests to local scripts and the existing CI definition.
- CFLSPLIT/1 remains the on-disk format; Auto is not written into part headers.
  Existing pieces and the unchanged Python helper remain compatible.
- CLI default changes from Base64 to Auto. Use `--encoding base64` explicitly
  for scripts that require encoded transport even for UTF-8 text.


## 1.0.0 — 2026-09-18

Initial native Windows splitter, verifier and joiner using the CFLSPLIT/1 format.
Three payload encodings: binary, Base64 text and readable UTF-8 text. Includes
SHA-256 checks, metadata-driven ordering, no-overwrite handling, cancellation,
a Go CLI and a standalone Python joiner.

### Repository preparation

- Moved the supplied Go module to the repository root without changing application behaviour.
- Adapted integration tests to build/use the native CLI on Windows or Linux.
- Added Windows/Linux test workflows and a Windows draft-release workflow.
- Added build, test, release-packaging and private GitHub-publishing scripts.
- Protected byte-exact demo fixtures with Git attributes.
- Added documentation, security guidance, issue/PR templates and dependency update configuration.
- Preserved the original test report separately from newly executed checks.
- Left the public licensing decision pending; no automatic public publication.

The original Windows binaries are not checked into source history. Build outputs
will differ when rebuilt with different Go versions or build flags. The file
format and application version remain 1.0.0.
