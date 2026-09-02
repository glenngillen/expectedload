# Plugin Validation

## Overview
The plugin must pass the Infracost CLI's built-in conformance suite (`infracost plugin validate`) and ship test fixtures that exercise detection and parsing end to end.

## Requirements
- `infracost plugin validate ./infracost-parser-plugin-expectedload` passes all checks:
  1. Connectivity — binary starts, handshakes, responds to gRPC.
  2. Describe — valid metadata (non-empty name, valid priority, extensions).
  3. Detect — returns `detected: false` gracefully for empty/nonexistent/unknown paths.
  4. Initialize — accepts the call without error.
  5. With `--fixture`: Detect claims the fixture and Parse returns a valid response.
- Add a `testdata/fixtures/` directory containing at least one fixture per supported syntax (Terraform, TS/JS, Python, Go, Java/Kotlin, Rust), each with a well-formed expected-load block, plus one fixture with deliberate errors to exercise diagnostics.
- Provide a `make validate` target that builds the binary and runs `infracost plugin validate` against it with the fixtures (mirrors the SDK example's Makefile).
- Note: `infracost plugin validate` is not yet in the released CLI (it exists in the in-flight CLI work implementing the new SDK). Until it ships, `make validate` must fail with an actionable "your infracost CLI does not support plugin validate" message, and the fixtures are still exercised by the plugin's own Go tests. CI wiring is deferred with the rest of the release mechanics (see `release-script.md`).

## Acceptance Criteria
- [ ] `make validate` passes locally against a current Infracost CLI build.
- [ ] `infracost plugin validate ... --fixture testdata/fixtures/<each>` passes for every syntax fixture.
- [ ] The error fixture yields diagnostics in the Parse response without failing validation.
- [ ] Go tests exercise every fixture directly (independent of the CLI), so validation coverage exists even before `plugin validate` ships.

## Edge Cases
- A dev machine whose `infracost` binary lacks `plugin validate`: the target must fail with an actionable message, never silently pass.
- Windows: validation of the `.exe` binary is a manual step for now (cross-compile only).

## Dependencies
- Infracost CLI with `plugin validate` support (in-flight; validate locally against that build when available).
- Topics: `parser-plugin-interface.md`, `expected-load-detection.md`, `parse-output-mapping.md`.

## Resolved Questions
- **Where `plugin validate` lives**: verified present on the CLI repo's `feature/plugin-architecture-refactor` branch (`internal/cmds/plugin.go`: `infracost plugin validate <plugin-binary>` with a repeatable `--fixture <path>` flag), and absent from `main`/released builds. Local validation is done against a build of that branch; no released version ships it yet, so the actionable-failure behavior of `make validate` (above) stays in place until one does.
- **A full pass is achievable**: the validator hard-fails on a Describe error, which is why the plugin implements all five RPCs from day one (see `parser-plugin-interface.md`).
- **Building the validation CLI locally**: the CLI feature branch `replace`s `infracost/proto` with `../proto`, so building it requires the proto repo checked out on its matching `feature/plugin-architecture-refactor` branch alongside the CLI repo. Document this in the README next to `make validate`.
