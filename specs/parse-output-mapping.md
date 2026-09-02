# Parse Output Mapping

## Overview
How `Parse` and `ParseToTree` turn the library's canonical `ExpectedLoad` model and `Diagnostic`s into the SDK's gRPC response types, so the CLI and downstream provider plugins can consume expected-load declarations.

## Requirements
- `Parse` walks the working directory (or the single detected file), selects the library `Syntax` frontend from each file's extension (`.tf`→Terraform, `.ts/.tsx/.js/.jsx`→JSDoc, `.py`→Python, `.go`→GoDirective, `.java/.kt`→Javadoc, `.rs`→Rustdoc), extracts comment blocks, and calls `ParseComment`.
- Every parsed declaration is emitted with:
  - its source location (file path relative to `repo_directory`, plus line number of the marker),
  - the normalized `Fields` map, and the meta fields (`version`, `confidence`, `last_updated`, `source`).
- Library `Diagnostic`s map to SDK `Diagnostic`s: library `Warning` → SDK warning severity, library `Error` → SDK error severity; the message and offending field are preserved, and each is annotated with the file/line it came from.
- Recoverable failures (one unreadable/unparsable file among many) produce diagnostics and partial results — the RPC itself succeeds. The RPC errors only on nil request/target.
- `ParseToTree` is the **primary output path**: it reuses the same walk/parse logic and converts declarations into the provider-agnostic `tree.Tree` structure — one node per declaration site carrying its load fields as attributes/usage values, so provider plugins can apply them to cost components. This path needs no format-specific proto messages at all.
- **No changes to `infracost/proto` are required or planned.** `Parse` works within the existing `ParseResponseResult` oneof variants in the pinned proto (Terraform, CloudFormation): return the variant matching the requested target, populated with expected-load data in the variant's extension/metadata fields where they fit, or an empty result plus diagnostics when they don't. (The SDK's own example plugin takes exactly this reuse-a-variant approach.) If a dedicated variant ever ships upstream, adopting it is a follow-up, not a prerequisite.

## Acceptance Criteria
- [ ] A fixture project mixing all six syntaxes parses into one declaration per site with correct fields, meta fields, and file/line locations.
- [ ] Malformed declarations (bad integer, misspelled field, invalid confidence/source enum) surface as diagnostics with the right severity while the rest of the project still parses (partial results).
- [ ] A file that cannot be read produces a diagnostic, not an RPC error.
- [ ] `ParseToTree` returns a tree the CLI accepts (verified via `infracost plugin validate --fixture`).
- [ ] Unit tests cover the extension→Syntax dispatch and the Diagnostic severity mapping.

## Edge Cases
- Multiple expected-load blocks in one file: each is a separate declaration with its own location.
- A marker with zero key/value pairs: valid per the library (non-nil model, empty fields); emit it — the analyzer side decides what an empty declaration means.
- Files matching an extension but in an unsupported dialect (e.g. `.ts` inside `node_modules`): skip well-known vendor directories (`node_modules`, `vendor`, `.git`, `.terraform`) during the walk.
- Duplicate keys within one block: the library's last-write-wins behavior is preserved; no extra diagnostic required for v1.

## Dependencies
- Library API: `ParseComment`, `ExpectedLoad`, `Diagnostic` (this repo).
- `infracost/proto` message definitions at the pin documented in `parser-plugin-interface.md` (no proto changes).
- Topics: `parser-plugin-interface.md`, `expected-load-detection.md`.

## Resolved Questions
- **Tree mapping reference confirmed.** `infracost/tree/tree.proto` and its Go bindings (`gen/go/infracost/tree/tree.pb.go`) are present both in tagged proto v1.167.0 and at the pinned feature-branch commit `689e4e6` (see `parser-plugin-interface.md`), with identical structure. Shape: `Tree` → `providers` map → `Service` → `Resource[]`; each `Resource` carries `id`, `type`, an optional `Definition` (source range, resource_type, address), and an `attributes` `ValueObject`. Mapping: one `Resource` per declaration site, load fields and meta fields carried in `attributes`, source file/line in `Definition.source`. The proto file itself warns not to hand-assemble the wire format — use the `ToProto()`/`FromProto()` tree helpers from the Go bindings; exact helper usage is an implementation detail for planning.
- **`ParseResponseResult` variant**: the oneof contains only `terraform.ModuleResult` and `cloudformation.Result` (verified identical in v1.167.0 and at the pinned commit `689e4e6`). `Parse` returns the variant matching the requested target's oneof case, or empty-result-plus-diagnostics when the target doesn't fit — as already specified above. `ParseToTree` remains the primary, variant-free path.
