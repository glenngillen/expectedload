package expectedload

import "testing"

func TestParseComment_AllSyntaxes_BlockForm(t *testing.T) {
	tests := []struct {
		name    string
		syntax  Syntax
		comment string
	}{
		{
			name:   "terraform",
			syntax: Terraform,
			comment: "# expected load:\n" +
				"#   monthly_calls: 100_000\n" +
				"#   avg_input_tokens: 1_200",
		},
		{
			name:   "jsdoc",
			syntax: JSDoc,
			comment: "/**\n" +
				" * @expected-load\n" +
				" *   monthly-calls: 100_000\n" +
				" *   avg-input-tokens: 1_200\n" +
				" */",
		},
		{
			name:   "python comment",
			syntax: Python,
			comment: "# expected-load:\n" +
				"#   monthly_calls: 100_000\n" +
				"#   avg_input_tokens: 1_200",
		},
		{
			name:   "python docstring",
			syntax: Python,
			comment: "Summarises an article using Claude.\n\n" +
				"Expected load:\n" +
				"    monthly_calls: 100_000\n" +
				"    avg_input_tokens: 1_200",
		},
		{
			name:   "go multi-line",
			syntax: GoDirective,
			comment: "// expected-load:\n" +
				"//   monthly-calls: 100_000\n" +
				"//   avg-input-tokens: 1_200",
		},
		{
			name:   "javadoc camelCase",
			syntax: Javadoc,
			comment: "/**\n" +
				" * @expected-load\n" +
				" *   monthlyCalls = 100_000\n" +
				" *   avgInputTokens = 1_200\n" +
				" */",
		},
		{
			name:   "rustdoc",
			syntax: Rustdoc,
			comment: "/// expected-load:\n" +
				"///   monthly-calls: 100_000\n" +
				"///   avg-input-tokens: 1_200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el, diags := ParseComment(tt.syntax, tt.comment)
			if el == nil {
				t.Fatalf("expected a declaration, got nil")
			}
			if len(diags) != 0 {
				t.Fatalf("unexpected diagnostics: %+v", diags)
			}
			if got, _ := el.Get("monthly_calls"); got != 100000 {
				t.Errorf("monthly_calls = %d, want 100000", got)
			}
			if got, _ := el.Get("avg_input_tokens"); got != 1200 {
				t.Errorf("avg_input_tokens = %d, want 1200", got)
			}
			if el.Version != 1 {
				t.Errorf("Version = %d, want default 1", el.Version)
			}
		})
	}
}

func TestParseComment_InlineForms(t *testing.T) {
	tests := []struct {
		name    string
		syntax  Syntax
		comment string
	}{
		{"jsdoc inline", JSDoc, `/** @expected-load monthly-calls=100_000 avg-input-tokens=1_200 */`},
		{"python single-line", Python, `# expected-load: monthly_calls=100_000 avg_input_tokens=1_200`},
		{"go directive", GoDirective, `//expected-load: monthly-calls=100_000 avg-input-tokens=1_200`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el, diags := ParseComment(tt.syntax, tt.comment)
			if el == nil {
				t.Fatalf("expected a declaration, got nil")
			}
			if len(diags) != 0 {
				t.Fatalf("unexpected diagnostics: %+v", diags)
			}
			if got, _ := el.Get("monthly_calls"); got != 100000 {
				t.Errorf("monthly_calls = %d, want 100000", got)
			}
			if got, _ := el.Get("avg_input_tokens"); got != 1200 {
				t.Errorf("avg_input_tokens = %d, want 1200", got)
			}
		})
	}
}

func TestParseComment_MetaFields(t *testing.T) {
	el, diags := ParseComment(JSDoc, "/**\n"+
		" * @expected-load\n"+
		" *   monthly-calls: 142_000\n"+
		" *   confidence: high\n"+
		" *   source: observed\n"+
		" *   last-updated: 2026-05-26\n"+
		" */")
	if el == nil {
		t.Fatal("expected a declaration")
	}
	if len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diags)
	}
	if el.Confidence != "high" {
		t.Errorf("Confidence = %q, want high", el.Confidence)
	}
	if el.Source != "observed" {
		t.Errorf("Source = %q, want observed", el.Source)
	}
	if el.LastUpdated != "2026-05-26" {
		t.Errorf("LastUpdated = %q, want 2026-05-26", el.LastUpdated)
	}
}

func TestParseComment_AbsenceIsNotAnError(t *testing.T) {
	el, diags := ParseComment(Python, "# just a normal comment\n# nothing to see here")
	if el != nil {
		t.Errorf("expected nil for comment without marker, got %+v", el)
	}
	if diags != nil {
		t.Errorf("expected no diagnostics, got %+v", diags)
	}
}

func TestParseComment_UnknownFieldSuggestion(t *testing.T) {
	el, diags := ParseComment(Python, "# expected-load:\n#   monthy_calls: 100_000")
	if el == nil {
		t.Fatal("expected a declaration")
	}
	if len(diags) != 1 {
		t.Fatalf("want 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	if diags[0].Severity != Warning {
		t.Errorf("severity = %v, want Warning", diags[0].Severity)
	}
	want := `unknown field "monthy_calls" — did you mean "monthly_calls"?`
	if diags[0].Message != want {
		t.Errorf("message = %q, want %q", diags[0].Message, want)
	}
	// The value is still captured (forward-compatible).
	if got, _ := el.Get("monthy_calls"); got != 100000 {
		t.Errorf("monthy_calls = %d, want 100000 (captured despite warning)", got)
	}
}

func TestParseComment_MalformedValue(t *testing.T) {
	el, diags := ParseComment(Python, "# expected-load:\n#   monthly_calls: lots")
	if el == nil {
		t.Fatal("expected a declaration")
	}
	if len(diags) != 1 || diags[0].Severity != Error {
		t.Fatalf("want 1 error diagnostic, got %+v", diags)
	}
	if _, ok := el.Get("monthly_calls"); ok {
		t.Errorf("malformed value should not be stored")
	}
}

func TestNormalizeKey(t *testing.T) {
	cases := map[string]string{
		"monthly-calls":   "monthly_calls",
		"monthly_calls":   "monthly_calls",
		"monthlyCalls":    "monthly_calls",
		"Monthly Calls":   "monthly_calls",
		"avgInputTokens":  "avg_input_tokens",
		"request.duration": "request_duration",
	}
	for in, want := range cases {
		if got := normalizeKey(in); got != want {
			t.Errorf("normalizeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseLoadValue(t *testing.T) {
	cases := map[string]int64{
		"5_000_000": 5000000,
		"1,200":     1200,
		"  730 ":    730,
		"0":         0,
	}
	for in, want := range cases {
		got, ok := parseLoadValue(in)
		if !ok || got != want {
			t.Errorf("parseLoadValue(%q) = (%d, %v), want (%d, true)", in, got, ok, want)
		}
	}
	if _, ok := parseLoadValue("abc"); ok {
		t.Errorf("parseLoadValue(abc) should fail")
	}
}
