package expectedload

import "testing"

// Every Example() must be a declaration the parser accepts cleanly and that
// yields the three core AI load fields — otherwise a tool pasting it from a
// diagnostic would emit something the parser rejects.
func TestExampleRoundTrips(t *testing.T) {
	for _, s := range []Syntax{Terraform, JSDoc, Python, GoDirective, Javadoc, Rustdoc} {
		el, diags := ParseComment(s, Example(s))
		if el == nil {
			t.Errorf("syntax %d: Example did not parse to a declaration", s)
			continue
		}
		for _, d := range diags {
			if d.Severity == Error {
				t.Errorf("syntax %d: Example produced error diagnostic: %s", s, d.Message)
			}
		}
		for _, f := range []string{"monthly_calls", "avg_input_tokens", "avg_output_tokens"} {
			if _, ok := el.Fields[f]; !ok {
				t.Errorf("syntax %d: Example missing field %q (got %v)", s, f, el.Fields)
			}
		}
	}
}

// The optional `model` pin is stored verbatim (model ids carry dots/colons) and
// is not mistaken for a numeric load field.
func TestModelFieldParses(t *testing.T) {
	block := "/**\n * @expected-load\n *   monthly_calls: 100\n *   model: us.anthropic.claude-sonnet-4-6\n */"
	el, diags := ParseComment(JSDoc, block)
	if el == nil {
		t.Fatal("block: did not parse")
	}
	for _, d := range diags {
		if d.Severity == Error {
			t.Errorf("block: unexpected error diagnostic: %s", d.Message)
		}
	}
	if el.Model != "us.anthropic.claude-sonnet-4-6" {
		t.Errorf("block: model = %q", el.Model)
	}
	if _, ok := el.Fields["model"]; ok {
		t.Error("block: model leaked into numeric Fields")
	}

	// Inline form with a colon inside the id (`=` separator wins over the id's `:`).
	inline := "// @expected-load model=anthropic.claude-opus-4-8-v1:0 monthly_calls=5"
	el2, _ := ParseComment(JSDoc, inline)
	if el2 == nil || el2.Model != "anthropic.claude-opus-4-8-v1:0" {
		t.Errorf("inline: model = %q (el=%v)", func() string { if el2 != nil { return el2.Model }; return "" }(), el2)
	}
}
