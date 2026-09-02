# Cross-Platform Builds

## Overview
Reproducible build tooling that produces the plugin binary for all platforms the Infracost CLI supports, with the SDK-mandated naming.

## Requirements
- Build matrix mirrors the Infracost CLI's own release matrix (verified against the CLI repo's `.github/workflows/release.yml`): `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`.
- Output naming follows the SDK convention: `infracost-parser-plugin-expectedload` (with `.exe` appended on Windows). Per-platform artifacts are disambiguated by directory or archive name (e.g. `dist/infracost-parser-plugin-expectedload_<version>_<os>_<arch>.tar.gz`), never by renaming the binary inside — the CLI discovers plugins by exact filename.
- `CGO_ENABLED=0` static builds (the library is stdlib-only; the plugin adds only pure-Go deps) so binaries run without a toolchain.
- Version metadata injected at build time via `-ldflags "-X main.version=..."` from the git tag; `-s -w` to keep binaries small (SDK guideline: < 50 MB).
- A `Makefile` with targets: `build` (host platform), `build-all` (full matrix into `dist/`), `test`, `validate` (see `plugin-validation.md`), `release` (see `release-script.md`), `clean`.
- Debug builds use the `-debug` suffix (`infracost-parser-plugin-expectedload-debug`) so the CLI ignores them, per the SDK spec.
- Archiving and checksums are handled by the release script (`release-script.md`); `build-all` only produces the raw per-platform binaries.

## Acceptance Criteria
- [ ] `make build` produces a working binary on the host platform.
- [ ] `make build-all` produces all six platform binaries in one invocation on any host OS.
- [ ] The Windows artifact contains `infracost-parser-plugin-expectedload.exe`.
- [ ] `--version` (or equivalent) on the built binary reports the injected version.
- [ ] Each binary is under 50 MB.

## Edge Cases
- Building on a host without `make` (Windows contributors): document the equivalent raw `go build` command in the README.
- Dirty working tree: version falls back to `<last-tag>-dev+<short-sha>` rather than failing.

## Dependencies
- Go toolchain matching `go.mod` (currently `go 1.25`).
- Topics: `parser-plugin-interface.md` (binary name), `release-script.md` (packages the `build-all` outputs).

## Resolved Questions
- Platform matrix: matches the CLI's published release matrix exactly — six platforms including `windows/arm64`; no 32-bit `linux/arm` (the CLI does not ship it).
