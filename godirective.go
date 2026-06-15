package expectedload

import "strings"

// stripGoDirective removes the leading `//` from a Go comment line. It handles
// both the directive form (`//expected-load: ...`, no space) and the
// conventional multi-line form (`// expected-load:` then `//   key: value`).
func stripGoDirective(line string) string {
	t := strings.TrimLeft(line, " \t")
	return strings.TrimPrefix(t, "//")
}
