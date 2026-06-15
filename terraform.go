package expectedload

import "strings"

// stripTerraform removes the leading `#` from an HCL comment line.
//
//	# expected load:
//	#   monthly_requests: 5_000_000
func stripTerraform(line string) string {
	t := strings.TrimLeft(line, " \t")
	t = strings.TrimPrefix(t, "#")
	return t
}
