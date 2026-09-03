// Package scan implements the plugin-facing logic that is independent of the
// gRPC transport: plugin metadata, expected-load detection, and walking source
// trees to extract declarations via the root expectedload library.
//
// Keeping this package free of go-plugin/grpc/proto imports means the future
// proto version bump only touches the cmd/ package, and everything here is
// unit-testable without spawning a plugin process.
package scan

import (
	"path/filepath"
	"strings"

	"github.com/infracost/expectedload"
)

// Plugin metadata returned by GetPluginInfo and GetParserConfig. See
// specs/expected-load-detection.md for the rationale behind each value.
const (
	// CanonicalName is the plugin's identity in registry/namespace/name form.
	// The repo lives in the infracost org, so it uses the reserved official
	// namespace.
	CanonicalName = "glenngillen/expectedload"
	// DisplayName is shown in CLI output.
	DisplayName = "Expected Load"
	// Priority uses the SDK-recommended default. Expected-load identifies
	// individual files, so it need not preempt directory-oriented parsers.
	Priority uint32 = 0
	// SupportsDirectories is true because declarations are scattered across a
	// source tree rather than concentrated in one file.
	SupportsDirectories = true
	// ProjectType identifies the detected format in Detect responses.
	ProjectType = "expectedload"
)

// syntaxByExt maps a file extension to the library syntax frontend that parses
// comments in that language.
var syntaxByExt = map[string]expectedload.Syntax{
	".tf":   expectedload.Terraform,
	".ts":   expectedload.JSDoc,
	".tsx":  expectedload.JSDoc,
	".js":   expectedload.JSDoc,
	".jsx":  expectedload.JSDoc,
	".py":   expectedload.Python,
	".go":   expectedload.GoDirective,
	".java": expectedload.Javadoc,
	".kt":   expectedload.Javadoc,
	".rs":   expectedload.Rustdoc,
}

// FileExtensions returns the extensions the scanner can handle.
func FileExtensions() []string {
	return []string{".tf", ".ts", ".tsx", ".js", ".jsx", ".py", ".go", ".java", ".kt", ".rs"}
}

// SyntaxForPath returns the syntax frontend for a file path's extension and
// whether the extension is supported at all.
func SyntaxForPath(path string) (expectedload.Syntax, bool) {
	s, ok := syntaxByExt[strings.ToLower(filepath.Ext(path))]
	return s, ok
}
