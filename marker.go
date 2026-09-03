package expectedload

import "regexp"

// MarkerRegexp returns the compiled regular expression that recognizes an
// expected-load marker (`@expected-load`, `expected_load`, `Expected Load:`,
// ...) within a single line. It is exported so tooling built on this library
// (e.g. the Infracost parser plugin's Detect sniffer) matches exactly the
// markers ParseComment accepts, rather than maintaining a duplicate pattern.
//
// The pattern is line-oriented: apply it to individual lines, not to a whole
// multi-line document.
func MarkerRegexp() *regexp.Regexp {
	return markerRe
}
