package expectedload

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRootPackageIsStdlibOnly enforces the shared-library contract: the root
// package must stay importable with zero third-party dependencies. Plugin
// dependencies (go-plugin, grpc, proto) may only be pulled in by cmd/ and
// internal/ packages.
func TestRootPackageIsStdlibOnly(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		f, err := parser.ParseFile(fset, e.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		for _, imp := range f.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			// Stdlib import paths have no dot in their first segment.
			first, _, _ := strings.Cut(path, "/")
			if strings.Contains(first, ".") {
				t.Errorf("%s imports non-stdlib package %q — the root package must stay dependency-free", e.Name(), path)
			}
		}
	}
}
