# Third-party notices

The application uses Go's standard library; `go.mod` has no third-party module
requirements. Go-built executables incorporate Go runtime/standard-library code.
The Go project license supplied with the original Go 1.23.2 build toolchain is
included at [third-party/GO-LICENSE.txt](third-party/GO-LICENSE.txt).

The release-packaging script adds `runtime-notices/GO-LICENSE.txt` from the
packaging Go toolchain and records the actual binary build metadata. The original
source notice is retained unchanged. Review any additional notices required by future dependencies or tooling
before distributing new builds. Python scripts use the Python standard library;
no Python interpreter is bundled.

GitHub Actions are external build-time services, not bundled runtime libraries.
Their pinned revisions are recorded in [docs/BUILD_DEPENDENCIES.md](docs/BUILD_DEPENDENCIES.md).

Claude, Anthropic, GitHub, Go, Python, Microsoft and Windows names are used for
identification. This independent utility is not endorsed by those organisations.
The original project code is licensed under MIT; see [LICENSE](LICENSE).
The Go runtime retains its own licence. MIT does not relicense third-party code.
Copyright (c) 2026 Computer Forensics Lab Ltd.
Project websites: https://cflab.uk and https://e-discovery.uk.
