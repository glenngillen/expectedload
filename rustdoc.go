package expectedload

import "strings"

// stripRustdoc removes the leading `///` (outer) or `//!` (inner) doc-comment
// marker from a Rust line, falling back to a plain `//` comment.
//
//	/// expected-load:
//	///   monthly-calls: 5_000_000
func stripRustdoc(line string) string {
	t := strings.TrimLeft(line, " \t")
	switch {
	case strings.HasPrefix(t, "///"):
		return strings.TrimPrefix(t, "///")
	case strings.HasPrefix(t, "//!"):
		return strings.TrimPrefix(t, "//!")
	default:
		return strings.TrimPrefix(t, "//")
	}
}
