# Implementation Plan — expectedload generic parser plugin

Implement an Infracost parser plugin around the existing stdlib-only
`expectedload` library. The protocol authority is the current
`../parser-plugin-sdk/parser/SPEC.md`, verified against the CLI's
`pkg/plugins/manager.go` and the `cli-5fb` validation implementation.

## 1. Shared plugin contract

- [x] Use protocol version 1 and the shared handshake:
  `INFRACOST_PLUGIN=de8c7e96-497c-4168-80c4-fc875c8ce764`.
- [x] Serve the single go-plugin dispense key `plugin`.
- [x] Register both `PluginService` and `ParserService` on one gRPC server.
- [x] Implement `GetPluginInfo`, returning name `infracost/expectedload`, the
  build version, metadata, and type `PARSER`.
- [x] Implement `GetParserConfig`, returning project type `expectedload` and
  the SDK-recommended default identification priority `0`.
- [x] Use tagged `github.com/infracost/proto v1.160.0`; do not use the abandoned
  parser-only proto branch, a pseudo-version, or a `replace` directive.

## 2. Identification and parsing

- [x] Implement non-recursive `IdentifyProjects` for each directory offered by
  the CLI.
- [x] Treat declarations as file-oriented projects: return marker-bearing file
  names in `files`, never `directory: true`. This prevents expected-load from
  taking ownership of a Terraform or other directory-oriented project.
- [x] Detect supported files by extension and a bounded 256 KB marker scan;
  missing and unreadable paths produce an empty response.
- [x] Implement the generic `Parse(ParseRequest.path)` RPC and return a
  provider-agnostic `tree.Tree` plus diagnostics.
- [x] Preserve the recursive source walker, language-specific comment
  extraction, vendor-directory exclusions, partial results, and source ranges.
- [x] Do not implement the abandoned `Describe`, `Detect`, `Initialize`, or
  `ParseToTree` RPCs, nor Terraform/CloudFormation result variants.

## 3. Library compatibility

- [x] Preserve the public root-package API and keep its imports stdlib-only.
- [x] Keep plugin dependencies isolated to `cmd/` at compile time and enforce
  root-package dependency hygiene with `depsguard_test.go`.
- [x] Reuse the exported marker regexp so identification and parsing agree.

## 4. Build, release, and validation

- [x] Keep host and six-platform static builds, version injection, archives,
  checksums, and clean/test targets.
- [x] Validate the binary with `infracost plugin validate <binary>`; the current
  validator checks the handshake, `GetPluginInfo`, `GetParserConfig`, and name
  collisions. Fixture behavior remains covered by Go integration tests because
  the validator has no `--fixture` option.
- [x] Verify against `cli-5fb`: CLI build, all CLI tests, plugin-package tests,
  and an actual validator handshake have passed.

## 5. Remaining release work

- [x] Decide the publication destination and add CI/release automation.
  Destination: GitHub Releases on this repository (community parser plugins are
  distributed as standalone binaries dropped into the CLI plugin directory; the
  SDK requires no registry registration). Added `.github/workflows/test.yml`
  (vet + test on PRs and pushes to `main`) and `.github/workflows/release.yml`
  (on a `v*.*.*` tag or manual dispatch, runs `make release` and uploads the
  six archives + `checksums.txt` to a draft GitHub Release).
- [~] Add the plugin to the registry manifest when it is ready for distribution.
  Not actionable: the current parser SDK (`../parser-plugin-sdk/parser/SPEC.md`)
  states "No registration with Infracost is required — a plugin only needs to be
  discoverable in the plugin directory," and defines no registry manifest format.
  Deferred until such a manifest exists.

The former plan's parser-specific cookie, `parser` dispense key, five-RPC API,
priority 45, directory claim, proto pseudo-version, and deferred validation
steps described an unmerged experimental interface and are intentionally not
part of this implementation.
