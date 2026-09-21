# Fictional UTF-16 regression example

`UTF16LE_example.txt` is harmless English/Persian text saved as UTF-16LE with a
BOM. It intentionally contains NUL bytes. Open it in the splitter and leave
**Auto (recommended)** selected. The selected output encoding should be `base64`.

This is source data for a test, not a CFLSPLIT part. Do not upload it together
with unrelated demo sets. The application must preserve every original byte,
not strip zero bytes or re-save the source as UTF-8. Rejoin into a NEW filename
and compare the source/reconstructed SHA-256 values.

The automated engine reproduction is recorded in `docs/VALIDATION.md`. The
Windows GUI still requires an interactive smoke test in that environment.
