package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/infracost/expectedload"
)

func TestScanMixedFixtureTree(t *testing.T) {
	res := Scan(fixtures(t))

	// One declaration per site across all fixtures:
	// terraform/main.tf: 2, typescript/app.ts: 1, python/service.py: 2,
	// golang/worker.go: 2, java/Handler.java: 1, java/Service.kt: 1,
	// rust/lib.rs: 1, errors/bad.py: 1. The node_modules decoy is skipped.
	wantByFile := map[string]int{
		"terraform/main.tf": 2,
		"typescript/app.ts": 1,
		"python/service.py": 2,
		"golang/worker.go":  2,
		"java/Handler.java": 1,
		"java/Service.kt":   1,
		"rust/lib.rs":       1,
		"errors/bad.py":     1,
	}
	got := map[string]int{}
	for _, d := range res.Declarations {
		got[d.File]++
		if d.Line <= 0 {
			t.Errorf("%s: declaration has no line number", d.File)
		}
		if d.Load == nil {
			t.Errorf("%s: nil load", d.File)
		}
	}
	for file, want := range wantByFile {
		if got[file] != want {
			t.Errorf("%s: got %d declarations, want %d", file, got[file], want)
		}
	}
	for file := range got {
		if _, ok := wantByFile[file]; !ok {
			t.Errorf("unexpected declarations in %s (vendor dir not skipped?)", file)
		}
	}
}

func TestScanDeclarationContent(t *testing.T) {
	res := Scan(fixtures(t, "terraform"))
	if len(res.Declarations) != 2 {
		t.Fatalf("got %d declarations, want 2", len(res.Declarations))
	}
	first := res.Declarations[0]
	if first.File != "main.tf" || first.Line != 1 {
		t.Errorf("first declaration at %s:%d, want main.tf:1", first.File, first.Line)
	}
	if v, ok := first.Load.Get("monthly_requests"); !ok || v != 5_000_000 {
		t.Errorf("monthly_requests = %d,%v, want 5000000,true", v, ok)
	}
	if first.Load.Confidence != "high" || first.Load.Source != "observed" {
		t.Errorf("meta fields = %q/%q, want high/observed", first.Load.Confidence, first.Load.Source)
	}
	second := res.Declarations[1]
	if v, ok := second.Load.Get("storage_gb"); !ok || v != 250 {
		t.Errorf("storage_gb = %d,%v, want 250,true", v, ok)
	}
	if len(res.Diagnostics) != 0 {
		t.Errorf("clean fixture produced diagnostics: %+v", res.Diagnostics)
	}
}

func TestScanInlineGoDirective(t *testing.T) {
	res := Scan(fixtures(t, "golang"))
	if len(res.Declarations) != 2 {
		t.Fatalf("got %d declarations, want 2", len(res.Declarations))
	}
	inline := res.Declarations[0]
	if inline.Line != 3 {
		t.Errorf("inline directive line = %d, want 3", inline.Line)
	}
	if v, _ := inline.Load.Get("requests_per_active_minute"); v != 3 {
		t.Errorf("requests_per_active_minute = %d, want 3", v)
	}
}

func TestScanPythonDocstring(t *testing.T) {
	res := Scan(fixtures(t, "python"))
	if len(res.Declarations) != 2 {
		t.Fatalf("got %d declarations, want 2", len(res.Declarations))
	}
	doc := res.Declarations[1]
	if v, _ := doc.Load.Get("avg_input_tokens"); v != 6_000 {
		t.Errorf("docstring avg_input_tokens = %d, want 6000", v)
	}
	if doc.Line != 12 {
		t.Errorf("docstring marker line = %d, want 12", doc.Line)
	}
}

func TestScanErrorFixtureDiagnostics(t *testing.T) {
	res := Scan(fixtures(t, "errors"))
	if len(res.Declarations) != 1 {
		t.Fatalf("got %d declarations, want 1 (partial results)", len(res.Declarations))
	}

	var errs, warns int
	for _, d := range res.Diagnostics {
		if d.File != "bad.py" || d.Line != 1 {
			t.Errorf("diagnostic located at %s:%d, want bad.py:1", d.File, d.Line)
		}
		switch d.Severity {
		case expectedload.Error:
			errs++
		case expectedload.Warning:
			warns++
		}
	}
	// monthly_calls: lots → error; monthy_calls (did-you-mean) and
	// confidence: absolutely → warnings.
	if errs != 1 || warns != 2 {
		t.Errorf("got %d errors, %d warnings; want 1, 2 (diags: %+v)", errs, warns, res.Diagnostics)
	}
}

func TestScanUnreadableFileDiagnostic(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; permission bits are not enforced")
	}
	tmp := t.TempDir()
	locked := filepath.Join(tmp, "locked.tf")
	if err := os.WriteFile(locked, []byte("# expected-load: storage_gb=1\n"), 0o000); err != nil {
		t.Fatal(err)
	}

	res := Scan(tmp)
	if len(res.Declarations) != 0 {
		t.Errorf("unexpected declarations from unreadable file")
	}
	if len(res.Diagnostics) != 1 || !res.Diagnostics[0].ReadFailure {
		t.Fatalf("want one ReadFailure diagnostic, got %+v", res.Diagnostics)
	}
}

func TestScanSingleFile(t *testing.T) {
	res := Scan(fixtures(t, "rust", "lib.rs"))
	if len(res.Declarations) != 1 {
		t.Fatalf("got %d declarations, want 1", len(res.Declarations))
	}
	if res.Declarations[0].File != "lib.rs" {
		t.Errorf("file = %q, want lib.rs", res.Declarations[0].File)
	}
	if res.Declarations[0].Line != 3 {
		t.Errorf("line = %d, want 3", res.Declarations[0].Line)
	}
}

func TestSyntaxForPathDispatch(t *testing.T) {
	cases := map[string]expectedload.Syntax{
		"a/b.tf": expectedload.Terraform,
		"a.ts":   expectedload.JSDoc,
		"a.tsx":  expectedload.JSDoc,
		"a.js":   expectedload.JSDoc,
		"a.jsx":  expectedload.JSDoc,
		"a.py":   expectedload.Python,
		"a.go":   expectedload.GoDirective,
		"a.java": expectedload.Javadoc,
		"a.kt":   expectedload.Javadoc,
		"a.rs":   expectedload.Rustdoc,
		"A.RS":   expectedload.Rustdoc, // extension matching is case-insensitive
	}
	for path, want := range cases {
		got, ok := SyntaxForPath(path)
		if !ok || got != want {
			t.Errorf("SyntaxForPath(%q) = %v,%v, want %v,true", path, got, ok, want)
		}
	}
	if _, ok := SyntaxForPath("a.yaml"); ok {
		t.Error("SyntaxForPath claimed .yaml")
	}
	if _, ok := SyntaxForPath("noext"); ok {
		t.Error("SyntaxForPath claimed extensionless path")
	}
}
