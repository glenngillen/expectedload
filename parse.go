package expectedload

import (
	"regexp"
	"strings"
)

// markerRe matches the expected-load marker in any spelling: `@expected-load`,
// `expected-load`, `expected_load`, `expected load`, with an optional trailing
// colon, capturing any inline remainder after it.
var markerRe = regexp.MustCompile(`(?i)@?expected[-_ ]load\s*:?\s*(.*)$`)

// pairRe matches a single `key: value` or `key = value` declaration line.
var pairRe = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_. -]*?)\s*[:=]\s*(.+?)\s*$`)

// ParseComment parses an expected-load declaration out of a raw comment block
// in the given syntax. The comment should be the full comment text (with its
// native markers, e.g. leading `#`, `//`, `*`, `///`).
//
// It returns (nil, nil) when the comment contains no expected-load marker —
// absence is normal, not an error. When a marker is present it always returns a
// non-nil *ExpectedLoad (possibly with zero fields) plus any diagnostics for
// malformed entries.
func ParseComment(syntax Syntax, comment string) (*ExpectedLoad, []Diagnostic) {
	return parseContentLines(contentLines(syntax, comment))
}

// contentLines strips the per-syntax comment markers, returning the inner text
// lines for the shared key/value parser.
func contentLines(syntax Syntax, comment string) []string {
	raw := strings.Split(comment, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		var stripped string
		switch syntax {
		case Terraform:
			stripped = stripTerraform(line)
		case JSDoc:
			stripped = stripJSDoc(line)
		case Python:
			stripped = stripPython(line)
		case GoDirective:
			stripped = stripGoDirective(line)
		case Javadoc:
			stripped = stripJavadoc(line)
		case Rustdoc:
			stripped = stripRustdoc(line)
		default:
			stripped = line
		}
		out = append(out, stripped)
	}
	return out
}

// orderedPair preserves declaration order so diagnostics read top-to-bottom.
type orderedPair struct {
	key string
	val string
}

// parseContentLines finds the marker and collects key/value pairs from the
// marker's inline remainder and any subsequent indented declaration lines.
func parseContentLines(lines []string) (*ExpectedLoad, []Diagnostic) {
	markerIdx := -1
	var inline string
	for i, line := range lines {
		if m := markerRe.FindStringSubmatch(line); m != nil {
			markerIdx = i
			inline = strings.TrimSpace(m[1])
			break
		}
	}
	if markerIdx == -1 {
		return nil, nil
	}

	var pairs []orderedPair
	// Inline form: `expected-load monthly-calls=100_000 avg-input-tokens=1_200`
	pairs = append(pairs, inlinePairs(inline)...)

	// Block form: subsequent indented `key: value` lines, until a blank line or
	// a line that isn't a declaration.
	for _, line := range lines[markerIdx+1:] {
		if strings.TrimSpace(line) == "" {
			if len(pairs) > 0 {
				break
			}
			continue
		}
		m := pairRe.FindStringSubmatch(line)
		if m == nil {
			break
		}
		pairs = append(pairs, orderedPair{key: m[1], val: m[2]})
	}

	return build(pairs)
}

// inlinePairs splits an inline remainder like `monthly-calls=100_000 turns=1`
// into key/value pairs. Supports `=` and `:` separators.
func inlinePairs(s string) []orderedPair {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var pairs []orderedPair
	for _, tok := range strings.Fields(s) {
		sep := strings.IndexAny(tok, "=:")
		if sep <= 0 {
			continue
		}
		pairs = append(pairs, orderedPair{
			key: tok[:sep],
			val: strings.Trim(tok[sep+1:], " \t,"),
		})
	}
	return pairs
}

// build turns ordered raw pairs into the canonical model plus diagnostics.
func build(pairs []orderedPair) (*ExpectedLoad, []Diagnostic) {
	el := &ExpectedLoad{Version: 1, Fields: map[string]int64{}}
	var diags []Diagnostic

	for _, p := range pairs {
		key := normalizeKey(p.key)
		val := strings.TrimSpace(p.val)

		switch key {
		case "version":
			if n, ok := parseLoadValue(val); ok {
				el.Version = int(n)
			} else {
				diags = append(diags, Diagnostic{Severity: Error, Field: p.key,
					Message: `field "version" must be an integer, got "` + val + `"`})
			}
		case "confidence":
			lc := strings.ToLower(val)
			if _, ok := confidenceValues[lc]; ok {
				el.Confidence = lc
			} else {
				diags = append(diags, Diagnostic{Severity: Warning, Field: p.key,
					Message: `confidence must be one of low, medium, high — got "` + val + `"`})
			}
		case "source":
			lc := strings.ToLower(val)
			if _, ok := sourceValues[lc]; ok {
				el.Source = lc
			} else {
				diags = append(diags, Diagnostic{Severity: Warning, Field: p.key,
					Message: `source must be one of manual, observed, estimated — got "` + val + `"`})
			}
		case "last_updated":
			el.LastUpdated = val
		case "model":
			// An optional pin for the model identifier when a consumer can't infer
			// it from code (e.g. a dependency-injected client). Stored verbatim —
			// model ids carry dots/colons/slashes that must not be normalized.
			el.Model = val
		default:
			n, ok := parseLoadValue(val)
			if !ok {
				diags = append(diags, Diagnostic{Severity: Error, Field: p.key,
					Message: `field "` + key + `" must be an integer, got "` + val + `"`})
				continue
			}
			el.Fields[key] = n
			if !isKnownField(key) {
				if sug := closestField(key); sug != "" {
					diags = append(diags, Diagnostic{Severity: Warning, Field: p.key,
						Message: `unknown field "` + key + `" — did you mean "` + sug + `"?`})
				}
			}
		}
	}

	// Point every diagnostic at the raw spec Markdown so authors (and tools/agents
	// acting on them) can fetch the format and field rules directly.
	for i := range diags {
		diags[i].Message += " (see " + SpecMarkdownURL + ")"
	}

	return el, diags
}

func isKnownField(key string) bool {
	for _, k := range knownFields {
		if k == key {
			return true
		}
	}
	return false
}
