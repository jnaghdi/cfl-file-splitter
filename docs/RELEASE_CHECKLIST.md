# Owner release checklist

- [ ] Retain MIT copyright/permission notices and Go runtime notices; confirm the private/public distribution decision.
- [ ] Review changes; exclude real evidence, credentials and customer details.
- [ ] Confirm VERSION matches core.Version and the proposed v-prefixed tag.
- [ ] Run repository, Go, Python cross-language, Auto-mode and streaming tests.
- [ ] On Windows, test the supplied UTF-16 example in Auto; then select strict
      readable mode and confirm the offered encoded retry works without changing
      the original. Confirm the GUI reports v1.0.2 and the actual output encoding.
- [ ] Confirm the complete new name fits the header at supported DPI/window sizes; test About & licence.
- [ ] Review the actual GitHub Linux/Windows CI results, not only local reports.
- [ ] Inspect the Windows package, source inventory, build information and hashes.
- [ ] Smoke-test the GUI on actual Windows x64: source/folder selection, drag/drop,
      Unicode/long paths, each mode, cancellation, error dialogs, Verify, Rejoin,
      no-overwrite, progress updates and repeated operations.
- [ ] Check NTFS and any intended network/FAT/exFAT targets using synthetic data.
- [ ] Test Claude uploads with fictional data in the intended account/workflow;
      verify raw attachment access and the reconstructed file hash.
- [ ] Document tests not performed and unresolved limitations.
- [ ] Review Go/Actions/toolchain updates; rebuild with a maintained toolchain.
- [ ] Decide whether Authenticode signing is required and use an approved signing
      process; never commit certificates or private keys.
- [ ] Push a tag matching VERSION only after approval; inspect the draft release.
- [ ] Publish the release only when the owner approves its notes and validation.

The supplied scripts do not tick these boxes or assert production certification.
