# CFLSPLIT/1 format

A part is a stream of bytes:

```
CFLSPLIT/1\n
{one UTF-8 JSON metadata object}\n
\n
[payload to end of file]
```

The literal magic is the ten ASCII bytes `CFLSPLIT/1` followed by LF. The whole
header must fit within 8192 bytes. There is no BOM or trailing padding in the
header, and no unconditional newline after the payload. Do not rewrite line
endings or edit the pieces. The original file's BOM/line endings, when present,
are payload bytes and are not normalised.

## Required metadata

| Field | Meaning |
|---|---|
| `format` | Literal `CFLSPLIT` |
| `version` | Integer 1 |
| `set_id` | Random 128-bit identifier represented by 32 hexadecimal characters |
| `original_name` | Original basename only; never use it as a trusted output path |
| `original_size` | Total original byte count |
| `original_sha256` | Lowercase SHA-256 of the complete original content |
| `original_modified_utc` | Recorded last-write time, informational only |
| `created_utc` | Split-set creation time |
| `encoding` | `binary`, `utf8` or `base64` |
| `part_index` | 1-based index |
| `total_parts` | Number of pieces, at least 1 |
| `offset` | 0-based original, DECODED-byte offset |
| `payload_size` | DECODED-byte count |
| `payload_sha256` | SHA-256 of the DECODED original bytes in this part |

All global fields must agree across the set. Byte ranges must cover the whole
original without gaps or overlaps. There must be exactly one part for every
index. An empty original uses one empty payload. Decoders reject incomplete,
mixed or duplicate sets rather than guessing or silently discarding pieces.

Binary/UTF-8 payloads are literal bytes. A UTF-8 piece is validated by the
splitter and never cuts a UTF-8 code point; it can cut a very long logical line.
Base64 uses the standard RFC 4648 alphabet, canonical `=` padding, no line
wrapping and no whitespace. Its transport length is `4 * ceil(payload_size/3)`.

Reconstruct by sorting METADATA indices, checking offsets, decoding each payload
when necessary and concatenating the decoded bytes. Check both per-part and
whole-file hashes. Do not concatenate the complete part files: their headers
are not part of the original. The filename extension is not part of the format.

The manifest is an optional inventory; it is not required to join. Header fields
are identification/integrity metadata, NOT signed provenance. An independent
trusted original hash is needed to distinguish a substituted consistent set.
No encryption, digital signature, compression, parity repair, original-data
execution, automatic archive extraction or automatic network transport is part
of version 1. This is a custom format, not compatible with ordinary `.001`
concatenation, 7-Zip multi-volume archives or forensic E01 segmentation.

## Auto mode (application v1.0.1)

`auto` is a source-selection policy, not a new on-disk encoding. Full-source
NUL-free UTF-8 validation selects `utf8`; text-compatibility failures restart
planning as `base64`. Headers contain only the selected actual encoding. No
metadata field or payload representation changes; v1.0.0 joiners remain
compatible. Explicit strict `utf8` continues to reject incompatible sources.
