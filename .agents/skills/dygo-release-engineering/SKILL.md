---
name: dygo-release-engineering
description: Prepare, validate, package, or review a dygo framework release and its upgrade path. Use for versioning, tags, binaries, Studio and Core bundling, installers, archives, checksums, and release smoke checks.
---

# dygo Release Engineering

Release a reproducible framework artifact from a clean, reviewed state.

## Source Contract

Read `docs/releasing.md`, `docs/installation.md`, the release scripts, and `.github/workflows/release.yml` before changing or running the release process.

## Rules

- Release from clean `main` that matches its remote.
- Use semantic version tags in the documented form.
- Keep tags annotated and make sure the commit belongs to `main`.
- Build and validate Studio, bundled Core metadata, Go binaries, installers, archives, and checksums through the repository release path.
- Keep generated-project upgrades compatible with the released framework contract.
- Verify that the installed binary reports the requested version.
- Keep binary replacement atomic and checksum verification mandatory.
- Do not use a moving source branch as a versioned installer.
- Do not claim a release exists until the tag, workflow, artifacts, and remote state prove it.

Release work is a major finalization task. Run the complete documented release checks. Do not push a tag or publish a release without explicit authorization.
