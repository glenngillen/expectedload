# expectedload

Parses `expected-load` declarations out of source-code comments so cost tooling
can reason about how heavily a piece of infrastructure or AI code will be used.
One canonical model, many comment syntaxes: Terraform/HCL, TypeScript/JavaScript
(JSDoc), Python, Go, Java/Kotlin (Javadoc/KDoc), and Rust.

```hcl
# expected load:
#   monthly_requests: 5_000_000
#   request_duration_ms: 120
#   confidence: high
resource "aws_lambda_function" "api" { ... }
```

```ts
/**
 * @expected-load
 *   monthly_calls: 100_000
 *   avg_input_tokens: 1_200
 */
export async function summarizeTicket(body: string) { ... }
```

This repo is two things:

1. **A shared Go library** — the root package `github.com/infracost/expectedload`,
   dependency-free (standard library only), imported by tools that need to read
   the same declaration convention.
2. **An Infracost parser plugin** — `cmd/infracost-parser-plugin-expectedload`,
   a standalone binary the [Infracost](https://infracost.io) CLI spawns over
   gRPC (HashiCorp go-plugin) to extract declarations from a project tree.

## Using the library

```go
import "github.com/infracost/expectedload"

load, diags := expectedload.ParseComment(expectedload.JSDoc, comment)
if load != nil {
    calls, _ := load.Get("monthly_calls")
    ...
}
```

`ParseComment` returns `(nil, nil)` when the comment has no expected-load
marker; a non-nil `*ExpectedLoad` (plus any diagnostics) otherwise. The root
package never gains non-stdlib dependencies — that contract is enforced by a
test — so importing it never drags in the plugin's gRPC stack.

## Using the plugin

Build and install:

```sh
make build
# or, without make (any platform):
CGO_ENABLED=0 go build -o infracost-parser-plugin-expectedload ./cmd/infracost-parser-plugin-expectedload
```

Place the binary in the Infracost CLI's plugin directory; the CLI discovers it
by its filename. The plugin implements the current parser-plugin SDK
(`Describe`, `Detect`, `Initialize`, `Parse`, `ParseToTree`):

- **Describe** — canonical name `plugins.infracost.io/infracost/expectedload`,
  priority 45 (weak-signal band: it never outranks the Terraform/ARM/
  CloudFormation parsers on shared extensions), extensions
  `.tf .ts .tsx .js .jsx .py .go .java .kt .rs`, directory support.
- **Detect** — content-sniffs (bounded to the first 256 KB) for an
  expected-load marker; directories are checked one level deep only.
- **Parse / ParseToTree** — walks the project (skipping `node_modules`,
  `vendor`, `.git`, `.terraform`), extracts comment blocks per syntax, and
  emits one declaration per site with repo-relative file/line locations.
  Malformed declarations become warning/error diagnostics without failing the
  parse. `ParseToTree` is the primary output: one tree resource per
  declaration, load fields as attributes.

The binary only speaks the new parser-plugin protocol; it is not loadable by
CLI releases that predate the plugin-architecture work.

## Building and releasing

```sh
make build       # host-platform binary (version stamped from git)
make build-all   # linux/darwin/windows × amd64/arm64 into dist/
make test        # go test ./...
make release     # test + build-all + per-platform archives + checksums.txt
make clean
```

`make release` (or `scripts/release.sh`) produces
`dist/infracost-parser-plugin-expectedload_<version>_<os>_<arch>.tar.gz`
(`.zip` for Windows) — each containing the exactly-named binary, LICENSE, and
this README — plus a SHA-256 `checksums.txt`. The version comes from `$VERSION`
if set, else the exact git tag, else `<last-tag>-dev+<sha>`. Publishing the
artifacts (CI, GitHub Releases, registry manifests) is deliberately out of
scope of the script.

## Validating against the Infracost CLI

```sh
make validate
```

This runs `infracost plugin validate` against the built binary with the
fixtures in `testdata/fixtures/`. The `plugin validate` subcommand is not in
released CLI versions yet — it lives on the CLI's
`feature/plugin-architecture-refactor` branch. To build that CLI locally, check
out the `infracost/proto` repo on the same branch as a sibling directory named
`proto` (the CLI branch `replace`s the proto module with `../proto`), then
`go build` the CLI. Until you have such a build, `make validate` fails with an
actionable message; the same fixtures are covered by this repo's own Go tests
(`go test ./...`), which exercise detection and parsing for every syntax
without needing the CLI.

## Declaration reference

A declaration is a comment containing the marker (`expected-load`,
`expected load`, `@expected-load`, case-insensitive, optional colon) followed
by `key: value` or `key = value` pairs — inline on the marker line or as an
indented block below it. Keys normalize to snake_case (`monthlyCalls`,
`monthly-calls`, and `monthly_calls` are the same field). Values are integers
(`_` and `,` separators tolerated). Meta fields: `version` (int, default 1),
`confidence` (`low|medium|high`), `source` (`manual|observed|estimated`),
`last_updated` (ISO-8601 date). Unknown fields are kept for forward
compatibility; a near-miss of a known field produces a "did you mean"
warning diagnostic.
