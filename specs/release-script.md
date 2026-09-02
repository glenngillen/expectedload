# Release Script

## Overview
A single script that generates the release files — per-platform archives plus checksums — from a clean checkout. Exact release *mechanics* (CI pipelines, GitHub Releases, plugin registry/manifest publishing) are deliberately deferred to a later stage; this script is the building block they will call.

## Requirements
- `scripts/release.sh` (invoked as `make release`) that, in one run:
  1. runs `go test ./...` and aborts on failure,
  2. builds the full platform matrix from `cross-platform-builds.md` into `dist/`,
  3. packages each platform as `infracost-parser-plugin-expectedload_<version>_<os>_<arch>.tar.gz` (`.zip` for windows), containing the exactly-named binary (`infracost-parser-plugin-expectedload`, `.exe` on windows) plus LICENSE and README,
  4. writes a SHA-256 `checksums.txt` covering all archives.
- Version resolution: `VERSION` env var if set, else the current git tag (`git describe --tags --exact-match`), else `<last-tag or v0.0.0>-dev+<short-sha>`. The resolved version is injected into the binary via `-ldflags`.
- The script is plain POSIX shell + go + tar/zip — runnable locally on macOS and Linux with no extra tooling (no GoReleaser dependency for now).
- `dist/` is gitignored.
- Idempotent: re-running cleans and regenerates `dist/`.

## Acceptance Criteria
- [ ] `make release` on a clean checkout produces six archives (one per platform in the matrix) and `checksums.txt` in `dist/`.
- [ ] Extracting any archive yields a binary with the exact SDK-mandated filename that runs and reports the injected version.
- [ ] `shasum -a 256 -c checksums.txt` passes inside `dist/`.
- [ ] A failing test aborts the script with a non-zero exit before any artifact is written.

## Edge Cases
- Untagged/dirty working tree: version falls back to the dev pseudo-version rather than failing (useful for local testing), but the script prints a clear warning.
- Missing `zip` on Linux hosts: fall back to `tar.gz` for windows too, with a warning (or require zip — decide during implementation, just don't fail silently).

## Dependencies
- Topic: `cross-platform-builds.md` (matrix, naming, ldflags).
- Later stage (out of scope here): wiring this script into CI and choosing the publish destination (GitHub Releases and/or an Infracost plugin download manifest).
