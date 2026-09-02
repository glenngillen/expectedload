# Parser Plugin Interface Conformance

## Overview
Wrap the existing `expectedload` comment-parsing library in an Infracost **parser plugin** binary that conforms to the current plugin SDK (documented in `../parser-plugin-sdk/parser/SPEC.md`). The plugin is a standalone binary that the Infracost CLI spawns and talks to over gRPC via the HashiCorp go-plugin framework.

## Requirements
- Add a `cmd/infracost-parser-plugin-expectedload/main.go` entrypoint; the existing library packages stay importable and unchanged (see `shared-library-compatibility.md`).
- Use the exact go-plugin handshake from the SDK spec:
  - `ProtocolVersion: 1`
  - `MagicCookieKey: "INFRACOST_PARSER_PLUGIN_MAGIC_COOKIE"`
  - `MagicCookieValue: "ac92b06c592f"`
  - Plugin map key `"parser"`
  - gRPC max send/recv message size of 64 MB
- The plugin struct embeds `plugin.NetRPCUnsupportedPlugin`, implements `plugin.GRPCPlugin`, registers a `ParserService` server in `GRPCServer`, and returns "not implemented" from `GRPCClient`.
- Implement **all five** `parser/api.ParserService` RPCs from the SDK spec: `Describe`, `Detect`, `Initialize`, `Parse`, `ParseToTree`. This is non-negotiable for working with the CLI: the new plugin manager (verified in the CLI's `feature/plugin-architecture-refactor` branch, `pkg/plugins/parser/manager.go`) calls `Describe` at load time and **skips any plugin that returns Unimplemented**, and `infracost plugin validate` hard-fails on a Describe error. `Initialize` may return an empty response but must not error.
- Dependencies: tagged releases of `github.com/hashicorp/go-plugin` and `google.golang.org/grpc`. For `github.com/infracost/proto`, no tagged release contains Describe/Detect yet (verified: absent from v1.167.0 and proto `main`; present only on the proto repo's `feature/plugin-architecture-refactor` branch, commit `689e4e6`, which carries generated Go bindings and tree.proto). Pin that commit as a Go **pseudo-version** (`v1.141.1-0.20260527052544-689e4e6554b4` — the branch is based on v1.141.0): resolvable via normal `go get`, reproducible, and crucially **no `replace` directives** (they would break library consumers). Bump to the real tagged release the moment one ships Describe/Detect. No changes to the proto repo itself.
- Keep plugin metadata (canonical name, display name, priority, extensions, directory support — see `expected-load-detection.md`) and the detection predicate in their own plain functions that the `Describe`/`Detect` handlers expose, so the proto version bump later is a mechanical change.
- Plugin metadata values (returned by Describe):
  - canonical name: `plugins.infracost.io/infracost/expectedload` (repo lives in the `infracost` org, so the reserved official namespace is correct)
  - display name: `Expected Load`
- Keep the plugin binary's `main` package thin: gRPC plumbing plus translation into/out of the library's `ExpectedLoad`/`Diagnostic` types.

## Acceptance Criteria
- [ ] `go build ./cmd/infracost-parser-plugin-expectedload` produces a binary named `infracost-parser-plugin-expectedload`.
- [ ] The binary handshakes with the CLI (verified via `infracost plugin validate` where available, see `plugin-validation.md`).
- [ ] All five RPCs respond without panics for empty inputs; `Parse`/`ParseToTree` error only when the request or target is nil.
- [ ] `go.mod` contains no `replace` directives; every dependency is a tagged version except `infracost/proto`, which is the documented pseudo-version until Describe/Detect ship in a tag.
- [ ] `go vet ./...` and `go test ./...` pass.

## Edge Cases
- Nil request or nil target in `Parse`/`ParseToTree`: return a gRPC error (per SPEC contract), never panic.
- Unknown target variant in the `oneof`: return a diagnostic-bearing empty result rather than a hard error where recoverable.
- Concurrent plugin instances: the library is stateless/pure; the service implementation must hold no shared mutable state.

## Dependencies
- `../parser-plugin-sdk/parser/SPEC.md` and `parser/example` as the reference for the target interface (note: the example's go.mod is stale — it pins proto v1.34.0 with a local `replace`; do not copy that).
- `github.com/infracost/proto` Go bindings at the pinned pseudo-version (see Requirements).
- Topics: `expected-load-detection.md`, `parse-output-mapping.md`.

## Resolved Questions
- **Namespace**: official (`plugins.infracost.io/infracost/expectedload`) — the module lives at `github.com/infracost/expectedload`, inside the reserved `infracost/` org namespace.
- **New SDK interface only; no legacy protocol.** Verified: the released CLI (`main`) speaks a legacy protocol (`INFRACOST_PLUGIN` magic cookie, dispense name `"plugin"`, `PluginService.GetPluginInfo` + a different `ParserService`), while the new SDK protocol (`INFRACOST_PARSER_PLUGIN_MAGIC_COOKIE`, dispense name `"parser"`, per-IaC discovery/routing via `tryPerIaCPlugins`) lives on the CLI's `feature/plugin-architecture-refactor` branch. The issue explicitly names the SDK docs as the current interface, so the plugin targets the new protocol exclusively. It will not be loadable by today's released CLI; dual-protocol support is out of scope unless requested later (the design keeps it possible — a single binary could branch on which magic-cookie env var is set).
- **Describe/Detect must ship now, not later.** Originally deferred as "not in a tagged proto", but verified CLI behavior overrules that: the new plugin manager skips plugins whose Describe returns Unimplemented, so a three-RPC plugin would work with *no* CLI at all. Resolution: implement all five RPCs against the proto feature-branch commit pinned as a pseudo-version (see Requirements); swap to the tagged release when it exists.
- **Proto pin is v1.141-based, not v1.167**: the proto feature branch forks from v1.141.0. Verified that everything this plugin consumes (the five-RPC `ParserService`, the same `ParseRequestTarget`/`ParseResponseResult` oneof variants, `tree.proto`, generated Go bindings) is present at commit `689e4e6`; the v1.142–v1.167 changes are unrelated to parser plugins.
