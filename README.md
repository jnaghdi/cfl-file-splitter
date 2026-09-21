# CFL File Splitter

**Split, verify and rejoin large files on Windows — with self-identifying pieces
and SHA-256 integrity checks.**

A native Windows desktop application, a command-line utility, and a standalone
Python reassembly helper for code-enabled environments such as Claude. The
Windows application needs no installed Python, .NET or Go runtime.

**Version:** 1.0.1 · **Format:** CFLSPLIT/1 · **Desktop target:** Windows x64.

> **Status:** unsigned maintenance build. Linux engine/interoperability checks and
> Windows cross-compilation have been performed. Interactive Windows GUI operation
> and uploads inside a Claude account have **not** been validated here. GitHub
> workflows are supplied, not represented as already run. Start with fictional
> demo data and working copies. See [validation](docs/VALIDATION.md).

## NUL-byte / UTF-16 fix in v1.0.1

Version 1.0.0 could automatically select strict UTF-8 from a `.txt`/`.log`
extension. Files containing NUL bytes or other encodings then stopped with a
compatibility error. **v1.0.1 starts in Auto**, checking actual content rather
than assuming the extension proves the encoding. It chooses readable UTF-8
only after successful full-source validation, otherwise it uses lossless Base64.
No NULs are deleted and the source is never re-saved in another encoding.
Explicit strict-readable mode still validates, but the GUI offers an encoded
retry rather than only displaying a dead-end error.

See [the fix, upgrade steps and reproducible example](docs/NUL_BYTE_FIX.md).
The CFLSPLIT/1 on-disk format and Python joiner are unchanged.

## First use

A source checkout does not contain prebuilt EXE files. Build locally with
[the build instructions](docs/BUILD.md), download a tested Windows artifact from
an actual successful CI run, or use a release approved by the repository owner.
Do not assume a release exists just because this README is present.

In the extracted Windows distribution, run `CFL_File_Splitter.exe`, select a
source file and an existing output parent folder, leave **Auto (recommended)**
selected and choose a maximum piece size, then click **Split & verify**. The default is **20 MB** including
the header (1 MB = 1,000,000 bytes). The application makes a new subfolder and
independently reads the saved pieces back to verify them.

| Mode | Intended content | How to process it |
|---|---|---|
| Auto (`auto`, default) | Unknown or mixed input types | Checks complete content for NUL-free valid UTF-8; otherwise preserves every byte in Base64. Check the reported actual encoding. |
| Readable UTF-8 only (`utf8`, strict) | UTF-8 text, logs, text exports | Text remains readable after the header. Line boundaries are preferred, not guaranteed. |
| Encoded text (`base64`) | Arbitrary file bytes, including PDF/DOCX/ZIP | Decode and reconstruct with code **before** processing the original. Base64 is not readable document text. |
| Binary (`binary`) | Local splitting/transfer without Base64 expansion | Use the Go joiner or Python helper. Do not assume `.cflpart` is accepted by a chat uploader. |

Readable mode preserves bytes but is not a CSV-record, JSON-object, PDF-page or
Word-section splitter. Base64 adds roughly one third to payload size. Splitting
does **not** bypass file counts, context windows, token limits, format support,
sandbox resources or an account's upload restrictions.

## What makes a piece identifiable?

Every piece embeds its set ID, original filename, original size and SHA-256,
part number/count, decoded byte offset/length, encoding, and payload SHA-256.
The joiner uses this metadata, **not filenames**, to determine ordering. It
rejects missing, duplicate, mixed, truncated and corrupted sets, and compares the
reconstructed content to the stored whole-file hash. Existing output files are
not intentionally overwritten.

A hash is **not a digital signature**. This release has no digital signing,
encryption, recovery parity, compression, automatic upload or evidence-format
parsing. Keep the original hash in an independent trusted location when the
risk of substitution matters. See [security](SECURITY.md) and
[format specification](docs/FORMAT.md).

## Build on Windows

Install a currently supported Go toolchain compatible with `go 1.23` or later.
Then run in PowerShell from this repository:

```powershell
.\scripts\build.ps1
```

The outputs are `bin\CFL_File_Splitter.exe` and `bin\CFL_Splitter_CLI.exe`.
The build script runs the Go tests before compiling. To also run the Python
interoperability checks, install Python 3.9+ and run:

```powershell
.\scripts\test.ps1 -Streaming
```

Do not change organisation-wide script-execution settings or disable security
controls to run an unsigned script. Review it and use your approved process;
the equivalent direct commands are in [BUILD.md](docs/BUILD.md).

Linux/macOS developers can run `bash scripts/test.sh` and
`bash scripts/build.sh`. The latter builds a native CLI and cross-compiles the
Windows x64 GUI and CLI. The desktop GUI is Windows-only.

## CLI examples

```powershell
# The output parent folder must already exist.
# Auto is the new CLI default; select it explicitly for clarity.
.\bin\CFL_Splitter_CLI.exe split --source "D:\Work\export.txt" --output "D:\Uploads" --encoding auto --max-bytes 20000000
.\bin\CFL_Splitter_CLI.exe split --source "D:\Work\report.pdf" --output "D:\Uploads" --encoding base64 --max-bytes 20000000
.\bin\CFL_Splitter_CLI.exe inspect --parts "D:\Uploads\CFL_report_set"
.\bin\CFL_Splitter_CLI.exe verify --parts "D:\Uploads\CFL_report_set"
.\bin\CFL_Splitter_CLI.exe join --parts "D:\Uploads\CFL_report_set" --output "D:\Work\NEW_report.pdf"
```

`inspect` checks the inventory/headers; **use `verify` for payload hashing**.
Use `--quiet` to suppress progress messages without suppressing results/errors.
`Ctrl+C` requests cancellation. Keep one split set per directory.

## Using the pieces with Claude

Start with **one** included demo set: `examples/Encoded_TXT/Parts` or
`examples/Readable_UTF8/Parts`. Attach its part TXT files and its two helper TXT
files, `CLAUDE_JOINER.txt` and `CLAUDE_INSTRUCTIONS.txt`. Do not attach both sets
or the Windows program. See [CLAUDE_USAGE.md](docs/CLAUDE_USAGE.md).

The reconstructed file is not automatically safe or fully processable. A
code-enabled environment must have the **original attachment bytes**, enough
resources, and appropriate format support. For large forensic images, a local
workflow or meaningful smaller exports is usually more practical than a chat
upload. This project does not promise acceptance by Claude.

## Create your GitHub repository

This prepared source package is **not evidence of a remote GitHub repository**.
To create one with the official GitHub CLI, install Git and `gh`, authenticate,
review the files, and run:

```powershell
gh auth login --hostname github.com --git-protocol https --web
.\scripts\publish-github.ps1
```

The script defaults to **a private `cfl-file-splitter` repository** in the
signed-in account, displays the target and staged file inventory, and asks for
`CREATE` before the remote write. It does not overwrite an existing repository,
force-push, or upload client evidence. An organisation can be specified with
`-Repository "YOUR-ORG/cfl-file-splitter"`. See [GITHUB_SETUP.md](docs/GITHUB_SETUP.md).

## Automated builds and draft releases

`.github/workflows/ci.yml` defines Linux/Windows tests, Python compatibility,
Linux race detection, the streaming test and a Windows ZIP artifact. Action
revisions are SHA-pinned. These checks run only after a real GitHub push and
subject to your account's Actions permissions and available runners.

Pushing a version tag matching `VERSION` triggers `release.yml`, which tests and
builds again and creates a **draft release** with the ZIP and its SHA-256. The
owner reviews the draft and Windows smoke tests before publication. No public
release or signing certificate is created automatically by the setup script.

## Repository layout

```text
cmd/cflsplit/               Native Windows desktop UI
cmd/cflcli/                 CLI entry point
internal/core/             Streaming engine and Go tests
internal/core/assets/      Embedded CLAUDE_JOINER.txt
tools/claude_joiner.py     Standalone identical Python helper
tests/                    Cross-language, streaming and repository checks
scripts/                  Build, test, package and GitHub-publish scripts
examples/                 Byte-exact fictional demo sets
docs/                     Usage, build, format, validation and release docs
.github/                  CI, draft release, Dependabot and issue templates
```

The `tools` copy and embedded helper are checked for byte equality. No `go.sum`
is required while the module has no external dependencies. Generated binaries
belong in release assets, not in source history.

## Privacy and limitations

The application itself has no automatic network transport, updater or telemetry.
The **publishing script** deliberately uses GitHub, and **CI** downloads build
tools and uploads source/build artifacts to GitHub. Only source and fictional
demo data belong in this repository. Ignore rules are guardrails, not a data-loss
prevention system. Never commit real evidence, credentials or customer records.

Only a selected file's ordinary content stream is preserved; this is not a
forensic acquisition or filesystem-metadata preservation tool. Original access
times may change when read. Use stable working copies, retain originals, and
follow approved evidence-handling and information-disclosure processes.

## Licensing

No public open-source licence has been selected. Keep the repository private
until the owner has chosen one. [LICENSE](LICENSE) records that decision as
pending rather than silently granting public reuse rights. See also
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
