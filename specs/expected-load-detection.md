# Expected-Load Detection

## Overview
How the plugin answers the CLI's `Detect` RPC: deciding whether a given file or directory contains `expected-load` declarations the plugin can parse, quickly and without false claims that would shadow other parsers.

## Requirements
- `Describe` metadata that drives detection:
  - `file_extensions`: the union of the syntaxes the library supports — `.tf`, `.ts`, `.tsx`, `.js`, `.jsx`, `.py`, `.go`, `.java`, `.kt`, `.rs`.
  - `priority`: 40+ (weak/generic signal band per the SDK spec — these extensions are all claimed by other tools; expected-load must never outrank the Terraform plugin on `.tf`). Proposed: 45.
  - `supports_directories`: true (declarations are scattered across a source tree).
- `Detect` contract:
  - Empty path → `detected: false`, no error.
  - Unsupported extension → `detected: false` immediately, without reading the file.
  - Supported extension → content-sniff for the marker (the library's `markerRe`: `@?expected[-_ ]load` with optional colon). Found → `detected: true`, `confidence: MEDIUM`, `project_type: "expectedload"`. Not found → `detected: false`.
  - If `content_provided` is true, sniff the provided bytes instead of reading from disk (LSP virtual documents).
  - Directory path → scan only top-level entries (no deep recursion) for at least one supported file containing the marker.
- Detection must complete in under 100 ms per path: bound reads (e.g. first 256 KB of a file), no network calls, never error on unknown paths.

## Acceptance Criteria
- [ ] `Detect` returns `detected: false` (not an error) for: empty path, nonexistent path, unsupported extension, supported extension with no marker.
- [ ] `Detect` returns `detected: true` with MEDIUM confidence for a fixture file per supported syntax containing an expected-load block.
- [ ] `Detect` honors `content_provided` and never touches disk in that case.
- [ ] Directory detection finds a marker in a top-level file but does not recurse into subdirectories.
- [ ] Unit tests cover each supported extension plus the negative cases; measured detection time on a large fixture stays well under 100 ms.

## Edge Cases
- Binary or non-UTF-8 file with a matching extension: sniff must not crash; return `detected: false`.
- Very large files: read a bounded prefix only; a marker beyond the bound is acceptable to miss at detect time (documented trade-off).
- The word "expected load" in prose (e.g. a README-style comment) matches the marker regex; MEDIUM confidence and the parse-stage tolerance (a marker with zero fields is valid) make this harmless.
- Case variations (`Expected-Load`, `EXPECTED_LOAD`) are matched — the regex is case-insensitive.

## Dependencies
- The library's `markerRe` in `parse.go` (reuse, don't duplicate).
- Topic: `parser-plugin-interface.md`.

## Resolved Questions
- **Priority = 45.** Verified against the CLI's `feature/plugin-architecture-refactor` routing (`tryPerIaCPlugins` in `pkg/plugins/parser/parse.go`): plugins are simply tried in ascending priority order and the first `detected: true` wins, so any value ≥ 40 keeps expected-load behind Terraform (10), ARM (25), and CloudFormation (30) on every shared extension. 45 leaves room on both sides of the weak-signal band.
- Note: `Describe`/`Detect` are implemented as real RPCs from day one (the CLI skips plugins without them — see `parser-plugin-interface.md` for the proto pseudo-version pin that makes this possible). The metadata and detection predicate still live in plain functions the handlers expose, so the later proto tag bump is mechanical.
