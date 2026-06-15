package expectedload

import "strings"

// stripJSDoc removes JSDoc block-comment framing from a line:
//
//	/**
//	 * @expected-load
//	 *   monthly-calls: 100_000
//	 */
//
// It also tolerates `// @expected-load:` single-line comments (the documented
// fallback when JSDoc-on-statement doesn't render on hover).
func stripJSDoc(line string) string {
	t := strings.TrimLeft(line, " \t")
	t = strings.TrimPrefix(t, "/**")
	t = strings.TrimPrefix(t, "/*")
	t = strings.TrimPrefix(t, "//")
	t = strings.TrimSpace(t)
	t = strings.TrimSuffix(t, "*/")
	t = strings.TrimLeft(t, " \t")
	// A leading `*` is the JSDoc continuation marker.
	t = strings.TrimPrefix(t, "*")
	return t
}
