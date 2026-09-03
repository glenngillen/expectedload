# Shared Library Compatibility

## Overview
The repo's existing role — the shared, stdlib-only `expectedload` comment-parsing library imported by other plugins (the IaC parser plugin and the AppCode analyzer) — must survive the addition of the plugin binary unchanged.

## Requirements
- The root package `github.com/infracost/expectedload` keeps its current public API: `Syntax` constants, `ParseComment`, `ExpectedLoad`, `Diagnostic`, `Severity`. No breaking changes.
- The root package stays dependency-free (standard library only) so it remains vendorable/re-publishable. All new dependencies (go-plugin, grpc, proto) are pulled in only by the plugin's `cmd/` package — acceptable in a single module because library consumers importing only the root package won't compile the plugin deps into their builds, but the go.mod grows; if a consumer objects, the fallback is a nested module under `cmd/`.
- Existing tests (`parse_test.go`, `example_test.go`) continue to pass unmodified.
- Documentation (README) explains the dual nature: importable library + installable plugin binary.

## Acceptance Criteria
- [ ] `go test ./...` passes with zero changes to existing library files and tests.
- [ ] A consumer importing `github.com/infracost/expectedload` alone builds without linking go-plugin/grpc code.
- [ ] The root package has no non-stdlib imports (enforced by a small test or CI grep).
- [ ] README documents both usages.

## Edge Cases
- The module must never gain `replace` directives — they leak to library consumers and break `go get`. All plugin dependencies use published tagged releases, so a single module suffices.

## Dependencies
- Topics: `parser-plugin-interface.md`.
- Downstream consumers named in the engineering plan (IaC parser plugin, AppCode analyzer plugin).
