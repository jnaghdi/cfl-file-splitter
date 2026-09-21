# Build dependencies and workflow pins

The application has no third-party Go modules. `go.mod` therefore has no
`require` entries and no `go.sum` is needed. The Python helper and tests use
only the standard library. Workflows run on GitHub-hosted Windows/Linux images,
install the provider's `stable` Go toolchain and Python 3.13, and log their
versions. Availability and organisation policy must be checked in the actual
GitHub account. No hosted run was performed during repository preparation.

Actions were resolved from the official release/commit pages when preparing
this project (2026-09-18). They are pinned to full commit IDs, not mutable majors:

| Action | Release | Commit |
|---|---|---|
| actions/checkout | v7.0.1 | 3d3c42e5aac5ba805825da76410c181273ba90b1 |
| actions/setup-go | v7.0.0 | b7ad1dad31e06c5925ef5d2fc7ad053ef454303e |
| actions/setup-python | v7.0.0 | 5fda3b95a4ea91299a34e894583c3862153e4b97 |
| actions/upload-artifact | v7.0.1 | 043fb46d1a93c77aae656e7c1c64a875d1fc6a0a |

Official primary-source references:
- https://github.com/actions/checkout/commit/3d3c42e5aac5ba805825da76410c181273ba90b1
- https://github.com/actions/setup-go/commit/b7ad1dad31e06c5925ef5d2fc7ad053ef454303e
- https://github.com/actions/setup-python/commit/5fda3b95a4ea91299a34e894583c3862153e4b97
- https://github.com/actions/upload-artifact/commit/043fb46d1a93c77aae656e7c1c64a875d1fc6a0a
- https://docs.github.com/en/actions/tutorials/build-and-test-code/go

Dependabot is configured for weekly proposed Action updates. Because workflow
files are part of the explicit source inventory, a maintainer must review any
update and regenerate `SOURCE_SHA256SUMS.txt` before the PR's inventory check
passes. No updater or network downloader is built into the desktop utility.

## Static analysis scope

`go vet ./internal/core ./cmd/cflcli` is the automated vet gate. A full
`GOOS=windows GOARCH=amd64 go vet ./...` also reports two `unsafe.Pointer`
conversion warnings in the existing Win32 GUI bridge (clipboard global-memory
pointer and WM_GETMINMAXINFO callback pointer). They are not suppressed or
represented as a clean full-GUI vet result. Native Win32 API/callback handling
requires manual review and actual Windows runtime tests before qualification.
See the captured warning output and `VALIDATION.md`.
