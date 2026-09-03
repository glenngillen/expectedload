package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixtures returns the path to the repo's shared fixture tree.
func fixtures(t *testing.T, parts ...string) string {
	t.Helper()
	return filepath.Join(append([]string{"..", "..", "testdata", "fixtures"}, parts...)...)
}

func TestDetectPathPerSyntaxFixture(t *testing.T) {
	cases := map[string]string{
		"terraform":  "main.tf",
		"typescript": "app.ts",
		"python":     "service.py",
		"golang":     "worker.go",
		"java":       "Handler.java",
		"kotlin":     filepath.Join("..", "java", "Service.kt"),
		"rust":       "lib.rs",
	}
	for name, file := range cases {
		dir := name
		if name == "kotlin" {
			dir = "java"
			file = "Service.kt"
		}
		if !DetectPath(fixtures(t, dir, file)) {
			t.Errorf("%s: expected fixture %s to be detected", name, file)
		}
	}
}

func TestDetectPathNegatives(t *testing.T) {
	tmp := t.TempDir()
	noMarker := filepath.Join(tmp, "plain.go")
	if err := os.WriteFile(noMarker, []byte("package plain\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	unsupported := filepath.Join(tmp, "notes.txt")
	if err := os.WriteFile(unsupported, []byte("expected-load: monthly_calls=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for name, path := range map[string]string{
		"empty path":            "",
		"nonexistent path":      filepath.Join(tmp, "missing.tf"),
		"unsupported extension": unsupported,
		"no marker":             noMarker,
	} {
		if DetectPath(path) {
			t.Errorf("%s: DetectPath(%q) = true, want false", name, path)
		}
	}
}

func TestDetectPathBinaryFile(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "blob.go")
	junk := make([]byte, 512*1024)
	for i := range junk {
		junk[i] = byte(i % 251)
	}
	if err := os.WriteFile(bin, junk, 0o644); err != nil {
		t.Fatal(err)
	}
	if DetectPath(bin) {
		t.Error("binary file detected as expected-load")
	}
}

func TestDetectContent(t *testing.T) {
	if !DetectContent([]byte("// expected-load:\n//   monthly_calls: 1\n")) {
		t.Error("marker in provided content not detected")
	}
	if DetectContent([]byte("// nothing to see\n")) {
		t.Error("content without marker detected")
	}
	// A marker past the sniff bound is missed by design.
	big := strings.Repeat("x\n", sniffLimit/2)
	if DetectContent([]byte(big + "// expected-load: monthly_calls=1\n")) {
		t.Error("marker beyond sniff limit should not be detected")
	}
}

func TestDetectDirectoryTopLevelOnly(t *testing.T) {
	// The terraform fixture dir has a marker in a top-level file.
	if !DetectPath(fixtures(t, "terraform")) {
		t.Error("directory with top-level marker file not detected")
	}

	// A directory whose only marker sits in a subdirectory is not detected.
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	err := os.WriteFile(filepath.Join(sub, "deep.tf"), []byte("# expected-load: storage_gb=1\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if DetectPath(tmp) {
		t.Error("directory detection recursed into subdirectories")
	}
}

func TestDetectSpeedOnLargeFixture(t *testing.T) {
	tmp := t.TempDir()
	big := filepath.Join(tmp, "big.py")
	var sb strings.Builder
	for range 200_000 {
		sb.WriteString("x = 1  # a comment line to pad the file out\n")
	}
	if err := os.WriteFile(big, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	DetectPath(big)
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Errorf("detection took %v, budget is 100ms", d)
	}
}
