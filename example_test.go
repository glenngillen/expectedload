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
