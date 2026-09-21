# CFL File Splitter 1.0.1 — NUL-byte / UTF-16 mode-selection fix

Fixes the dead-end error seen when NUL-containing or non-UTF-8 data was assigned
Readable UTF-8 mode. Source selection no longer assumes `.txt`/`.log` means UTF-8.

**Auto (recommended)** is now the GUI and CLI default: it validates the entire
source for NUL-free UTF-8, otherwise selects lossless Base64. It never deletes
NULs or converts the source. The actual encoding is reported in the log.
Explicit strict-readable mode offers an encoded retry in the GUI. Real I/O,
cancellation and integrity failures remain errors rather than being ignored.

The transport format and Python helper are unchanged. Existing v1.0.0 sets
remain supported. Scripts requiring Base64 should use `--encoding base64`
explicitly because the CLI default is now Auto.

**Unsigned build.** See `docs/VALIDATION.md` for checks actually performed and
`docs/NUL_BYTE_FIX.md` for upgrade instructions and a reproducible UTF-16 example.
Windows GUI interaction and actual Claude uploads still require testing in those
environments. The user's original source file was not supplied.

Encoded pieces must be decoded/rejoined with code before document analysis;
this fix does not remove account, context, file-count or execution-resource
limits. No automatic upload, signature, encryption, PDF-page extraction or
forensic evidence-format parser is added. Source content is preserved, not
filesystem metadata or chain of custody. Review licensing before distribution.
