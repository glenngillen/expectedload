package expectedload

import "strings"

// stripPython removes the leading `#` from a Python comment line. Lines that
// are not `#` comments (e.g. the body of a docstring) are passed through
// unchanged, so the docstring `Expected load:` block form is handled by the
// shared marker parser without special-casing.
//
//	# expected-load:
//	#   monthly_calls: 100_000
//
//	"""...
//	Expected load:
//	    monthly_calls: 100_000
//	"""
func stripPython(line string) string {
	t := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(t, "#") {
		return strings.TrimPrefix(t, "#")
	}
	return line
}
