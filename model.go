// Package expectedload parses `expected-load` declarations out of source-code
// comments across multiple languages (Terraform/HCL, TypeScript/JavaScript,
// Python, Go, Java/Kotlin, Rust) into a single canonical model.
//
// It is the shared frontend referenced by ai-native-plan.md Phase 1: one parser,
// many syntax frontends. Both the IaC parser plugin and the AppCode analyzer
// plugin import it so the `expected-load` convention is identical everywhere.
//
// The package has no dependencies beyond the standard library so it can be
// vendored or re-published unbranded (see plan Phase 9).
package expectedload

// SpecURL is the canonical home of the Expected Load specification, where the
// declaration format and field vocabulary are documented. Diagnostics reference
// it so anyone who hits one can find the spec and learn how to populate the data.
const SpecURL = "https://expectedload.dev"

// SpecMarkdownURL is the raw Markdown of the specification, served on the same
// domain. Diagnostics point at this (rather than the HTML homepage) so an agent
// acting on one can fetch the authoritative format directly, with no scraping.
const SpecMarkdownURL = "https://expectedload.dev/spec.md"

// Example returns a minimal, copy-pasteable expected-load declaration in the
// comment grammar for the given language: the three core AI load fields, placed
// immediately above the call. A URL alone tells a reader (or an agent acting on
// a missing-expected-load diagnostic) that a declaration is needed but not how
// to write one in this language — this gives them the exact syntax. The full
// field set and meta fields live at SpecURL.
func Example(s Syntax) string {
	switch s {
	case Python:
		return "# expected-load:\n" +
			"#   monthly_calls: 100_000\n" +
			"#   avg_input_tokens: 1_200\n" +
			"#   avg_output_tokens: 500"
	case GoDirective:
		return "//expected-load:\n" +
			"//  monthly_calls: 100_000\n" +
			"//  avg_input_tokens: 1_200\n" +
			"//  avg_output_tokens: 500"
	case Rustdoc:
		return "/// expected-load:\n" +
			"///   monthly_calls: 100_000\n" +
			"///   avg_input_tokens: 1_200\n" +
			"///   avg_output_tokens: 500"
	case Terraform:
		return "# expected-load:\n" +
			"#   monthly_calls: 100_000\n" +
			"#   avg_input_tokens: 1_200\n" +
			"#   avg_output_tokens: 500"
	default: // JSDoc, Javadoc
		return "/**\n" +
			" * @expected-load\n" +
			" *   monthly_calls: 100_000\n" +
			" *   avg_input_tokens: 1_200\n" +
			" *   avg_output_tokens: 500\n" +
			" */"
	}
}

// Syntax selects which comment grammar a frontend should strip before the
// shared key/value parser runs.
type Syntax int

const (
	// Terraform parses `# expected load:` HCL comments.
	Terraform Syntax = iota
	// JSDoc parses `/** @expected-load ... */` TypeScript/JavaScript comments.
	JSDoc
	// Python parses `# expected-load:` comments and docstring `Expected load:` blocks.
	Python
	// GoDirective parses `//expected-load:` Go comments.
	GoDirective
	// Javadoc parses `/** @expected-load ... */` Java/Kotlin comments.
	Javadoc
	// Rustdoc parses `/// expected-load:` Rust comments.
	Rustdoc
)

// ExpectedLoad is the canonical, surface-agnostic model produced from any
// supported comment syntax. Numeric load fields (monthly_calls,
// avg_input_tokens, monthly_requests, ...) live in Fields keyed by their
// normalized snake_case name; the meta fields are promoted to typed fields.
type ExpectedLoad struct {
	// Version of the declaration. Defaults to 1 when not specified.
	Version int
	// Fields holds the normalized numeric load fields, e.g.
	// {"monthly_calls": 100000, "avg_input_tokens": 1200}.
	Fields map[string]int64
	// Confidence is one of "low", "medium", "high" (empty if unset).
	Confidence string
	// LastUpdated is an ISO-8601 date string (empty if unset).
	LastUpdated string
	// Source is one of "manual", "observed", "estimated" (empty if unset).
	Source string
}

// Get returns a load field value and whether it was present.
func (e *ExpectedLoad) Get(field string) (int64, bool) {
	if e == nil || e.Fields == nil {
		return 0, false
	}
	v, ok := e.Fields[field]
	return v, ok
}

// Severity classifies a Diagnostic.
type Severity int

const (
	// Warning indicates a malformed or suspicious declaration that was still
	// parsed as best as possible.
	Warning Severity = iota
	// Error indicates a value that could not be parsed at all.
	Error
)

// Diagnostic reports a problem found while parsing a declaration. Absence of an
// expected-load block is never a diagnostic — it returns (nil, nil).
type Diagnostic struct {
	Severity Severity
	// Field is the offending (raw, pre-normalization) field name, if any.
	Field string
	// Message is human-readable, e.g.
	// `unknown field "monthy_calls" — did you mean "monthly_calls"?`
	Message string
}

// confidenceValues and sourceValues are the accepted meta-field enumerations.
var (
	confidenceValues = map[string]struct{}{"low": {}, "medium": {}, "high": {}}
	sourceValues     = map[string]struct{}{"manual": {}, "observed": {}, "estimated": {}}
)

// knownFields is the union of the AI (plan §Data Model) and a representative
// set of Terraform (inline-usage-plan.md) load fields. Unknown fields are
// tolerated for forward-compatibility, but a close miss triggers a
// "did you mean" diagnostic.
var knownFields = []string{
	// AI module (v1) — pure load facts. Caching cost (prefix size, TTL, keys,
	// hit rate) is derived by the analyzer from the prompt structure + the
	// request pattern below, never declared as caching knowledge here.
	"monthly_calls",
	"avg_input_tokens",
	"avg_output_tokens",
	"avg_conversation_turns",
	"requests_per_active_minute", // request rate while the workload is active — captures the spacing/burstiness that monthly volume alone misses
	// Terraform (representative; full set in inline-usage-plan.md)
	"monthly_requests",
	"request_duration_ms",
	"storage_gb",
	"monthly_data_processed_gb",
}
