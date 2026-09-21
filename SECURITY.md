# Security policy and boundaries

## Reporting

Do not post client files, credentials, case names or sensitive filenames in an
issue. Contact the repository owner through an agreed private channel. If GitHub
private vulnerability reporting is enabled for the repository, use it. No
security mailbox, response-time guarantee or reporting service is configured by
this package; the owner must establish these before a public launch.

## Trust model

Treat part files and reconstructed content as untrusted data. Headers must not
be trusted as output paths. Neither the Go engine nor Python helper automatically
executes or extracts recovered data. Valid hashes show agreement with the hashes
supplied; they do not authenticate the sender. An attacker able to replace the
entire set can supply a new internally consistent set.

There is no encryption, digital signature, malware scanner, repair parity,
forensic acquisition or chain-of-custody system. This is not a substitute for
format-specific validation or controlled evidence-handling processes. Limit disk
space, CPU time and file counts when accepting untrusted data.

## Repository hygiene

Keep real evidence and client exports outside the checkout. The demo fixtures
are fictional. `.gitignore` cannot prevent all accidental disclosures, including
arbitrary TXT/PDF files; inspect staged content before every push. Never commit
GitHub tokens, signing certificates, passwords or API keys.

The application is local, but publishing and Actions workflows use the network.
GitHub receives committed source and any uploaded CI artifacts. The publisher
creates a private repository by default; it does not grant an open-source licence.

## Build and release

Review source before executing unsigned scripts or binaries. Use a maintained Go
toolchain. Actions are pinned to full commit SHAs and Dependabot is configured to
propose updates; review those updates rather than merging blindly. Pull request
checks do not receive a write token from these workflows. The release workflow
runs only on version-tag pushes and creates a draft for owner review.

No independent security audit, Windows GUI runtime qualification, penetration
test or Authenticode signing has been performed. A SHA-256 checksum file is an
integrity inventory, not publisher authentication. Use your normal IT approval
process; do not disable security controls to run this utility.
