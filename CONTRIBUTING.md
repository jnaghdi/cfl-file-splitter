# Contributing / maintainer workflow

This project is distributed under the MIT License; see [LICENSE](LICENSE).
Submit only code and documentation you have the right to contribute under MIT.
Retain third-party notices. Never include real client data in contributions.

Use a feature branch and a reviewed pull request. Include a small **synthetic**
reproduction for bugs; never attach case material. Run `scripts/test.ps1
-Streaming` on Windows or `bash scripts/test.sh --streaming` on Linux. Run
`gofmt` on modified Go files. New behaviour needs regression tests and a changelog
entry. Existing CFLSPLIT/1 data must remain readable unless a new format version
is explicitly introduced and documented.

`internal/core/assets/CLAUDE_JOINER.txt` and `tools/claude_joiner.py` must be byte
identical. Update both and run the cross-language tests. Do not rewrite demo part
files or let editors normalise their payloads. When intentionally regenerating
fixtures, record the new independent original hash and verify both consumers.

Keep source changes distinct from formatting and build changes. Do not claim a
Windows GUI test, Claude upload or CI run passed unless that run actually
occurred and its evidence is recorded. See `docs/RELEASE_CHECKLIST.md`.
