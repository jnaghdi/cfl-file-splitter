# Fix for “NUL byte found” — v1.0.1

## What was wrong

The reported screenshot showed a readable-mode compatibility rejection:

```text
NUL byte found: use encoded-text mode for binary or UTF-16 data
```

Inspection of v1.0.0 found that selecting a `.txt`, `.log`, `.jsonl` or `.ndjson`
file automatically selected Readable UTF-8 without verifying the actual encoding.
The readable validator rejected byte 0x00, leaving only an error dialog. The
screenshot alone does not identify the original file's encoding or prove damage;
the user's original file was not available for inspection.

NUL is legal UTF-8 for U+0000. Rejecting it is this application's deliberate
readable-transport policy, not a claim that all NUL-containing files are corrupt.
UTF-16, UTF-32 and binary files can contain zero bytes as ordinary content. Do
not delete zero bytes or overwrite the source with a different encoding.

## Immediate workaround in the older build

Select **Encoded text (.txt) — any file; rejoin before analysis**, then split
again. This transport supports arbitrary bytes and preserves the original hash.
Do not select Readable UTF-8 for a source that fails its compatibility check.

## Upgrade on Windows

1. Close the older application.
2. Extract `CFL_File_Splitter_Windows_v1.0.1.zip` into a new folder.
3. Run `CFL_File_Splitter.exe` from that folder and confirm **v1.0.1** is displayed.
4. Select the source and an existing output parent folder.
5. Leave **Auto (recommended)** selected. Use 20 MB as the initial piece-size
   setting; this is a design default, not a provider upload guarantee.
6. Click **Split & verify**, read the transport confirmation, and continue.
7. Check the log for the actual encoding and successful verification.

No Python, Go or .NET installation is needed to run the Windows EXEs. They remain
unsigned. Do not disable endpoint protection to run them; use your approved
software-review process and begin with the harmless included example.

## What the fix changes

Auto validates the complete source rather than sampling its beginning or relying
on its extension. It uses readable transport only after NUL-free UTF-8 validation
passes. A NUL, invalid sequence, incomplete final character or incompatible text
boundary makes it rewind and re-plan in Base64. Only compatibility failures
trigger this retry; real read errors, cancellation and integrity failures still
stop the operation. No part files are created before a complete plan is ready.

This is a byte-preserving transport decision, not an infallible detector of every
possible document type or character encoding. Bytes already forming NUL-free
valid UTF-8 are treated as readable transport; semantic content may still be a
structured format. Files chosen manually for strict readable-only mode are not
silently changed to another transport: the GUI asks before retrying encoded.

The original file is opened for reading. All original content bytes, including
BOMs, NULs and line endings, survive splitting/reassembly. Per-part and final
SHA-256 checks remain in place. Filesystem metadata is outside this guarantee.
The application and source package version is 1.0.1; the data format remains
CFLSPLIT/1, with actual `utf8`, `base64` or `binary` in every part header.

Auto can reread a large portion of a file if an incompatible byte occurs near
the end. Selecting Encoded text directly avoids that failed readable scan when
the input is already known to be binary or UTF-16.

## Reproduce the fix with harmless data

Use `examples/Encoding_Regression/UTF16LE_example.txt`. It is fictional text
encoded in UTF-16LE with a BOM and includes ordinary zero bytes.

In v1.0.0, selecting this `.txt` file caused the GUI to choose strict readable
mode; its validator produced the same NUL-byte error class. In v1.0.1, leave
Auto selected. The log should show a switch to Encoded text, successful saved-part
verification, and an actual payload encoding of `base64`.

Rejoin into a NEW filename and compare bytes/hashes. The sample is intentionally
not converted to UTF-8; encoded transport reconstructs the exact UTF-16 file.
The actual GUI retry dialog must still be smoke-tested on Windows.

## Claude processing

Attach the generated TXT parts and their matching `CLAUDE_JOINER.txt` and
`CLAUDE_INSTRUCTIONS.txt`. Encoded parts are Base64 transport, not independently
readable sections. A code-enabled environment must get their original bytes,
verify and rejoin them, then parse the reconstructed original in its actual
encoding/format. Never claim a complete review merely because joining succeeds.
See `CLAUDE_USAGE.md`. This release was not uploaded into a Claude account during
validation, and no upload acceptance or processing-limit bypass is promised.

## Source repository update

The v1.0.1 source ZIP contains the full revised repository. For an existing
checkout, review/copy its source files (including hidden configuration files),
without replacing your `.git` directory or adding the EXEs or client data.
Regenerate the inventory only after reviewing any additional local changes.
Run the tests, inspect your diff and commit through your normal workflow.
No remote GitHub repository was changed as part of this repair.

## Technical references

These describe encoding fundamentals, not proof of the user's file format:

- Go UTF-8 validation: https://pkg.go.dev/unicode/utf8
- Microsoft character-encoding guidance:
  https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_character_encoding

Validation and fresh logs: `VALIDATION.md` and `test-results/v1.0.1/`.
