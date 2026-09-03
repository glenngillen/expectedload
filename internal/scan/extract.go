package scan

import (
	"strings"

	"github.com/glenngillen/expectedload"
)

// commentBlock is one contiguous comment (a run of line comments, one block
// comment, or a Python docstring) with the 1-based line number where it starts.
type commentBlock struct {
	startLine int
	lines     []string
}

// commentShape describes how comments look in a syntax family, for lifting
// candidate blocks out of a source file. This is deliberately not a lexer:
// a comment-looking token inside a string literal may produce a spurious
// block, which is harmless — ParseComment ignores blocks without a marker.
type commentShape struct {
	linePrefixes []string    // e.g. "//", "#" — comment must start the line (after indentation)
	blocks       [][2]string // block delimiters, e.g. {"/*", "*/"} or {`"""`, `"""`}
}

// shapeFor returns the comment shape for a library syntax.
func shapeFor(syntax expectedload.Syntax) commentShape {
	switch syntax {
	case expectedload.Terraform:
		// The library's Terraform stripper handles `#` comments only.
		return commentShape{linePrefixes: []string{"#"}}
	case expectedload.Python:
		return commentShape{
			linePrefixes: []string{"#"},
			blocks:       [][2]string{{`"""`, `"""`}, {"'''", "'''"}},
		}
	case expectedload.JSDoc, expectedload.Javadoc:
		return commentShape{
			linePrefixes: []string{"//"},
			blocks:       [][2]string{{"/*", "*/"}},
		}
	case expectedload.GoDirective, expectedload.Rustdoc:
		// `///`, `//!`, and `//expected-load:` all share the `//` prefix.
		return commentShape{
			linePrefixes: []string{"//"},
			blocks:       [][2]string{{"/*", "*/"}},
		}
	default:
		return commentShape{}
	}
}

// extractComments lifts every comment block out of src. Line numbers are
// 1-based.
func extractComments(syntax expectedload.Syntax, src string) []commentBlock {
	shape := shapeFor(syntax)
	lines := strings.Split(src, "\n")

	var out []commentBlock
	var cur *commentBlock // open line-comment run
	closeRun := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimLeft(lines[i], " \t")

		// Block comments (and docstrings) take precedence over line prefixes:
		// `/*` also matches no line prefix, but `"""` needs the check first.
		if start, end, ok := blockStart(shape, trimmed); ok {
			closeRun()
			blk := commentBlock{startLine: i + 1, lines: []string{lines[i]}}
			// Same-line close: content after the opening delimiter must
			// contain the closing one (e.g. `""" one-liner """`).
			rest := trimmed[len(start):]
			if strings.Contains(rest, end) {
				out = append(out, blk)
				continue
			}
			for i++; i < len(lines); i++ {
				blk.lines = append(blk.lines, lines[i])
				if strings.Contains(lines[i], end) {
					break
				}
			}
			out = append(out, blk)
			continue
		}

		if hasLinePrefix(shape, trimmed) {
			if cur == nil {
				cur = &commentBlock{startLine: i + 1}
			}
			cur.lines = append(cur.lines, lines[i])
			continue
		}
		closeRun()
	}
	closeRun()
	return out
}

func blockStart(shape commentShape, trimmed string) (start, end string, ok bool) {
	for _, b := range shape.blocks {
		if strings.HasPrefix(trimmed, b[0]) {
			return b[0], b[1], true
		}
	}
	return "", "", false
}

func hasLinePrefix(shape commentShape, trimmed string) bool {
	for _, p := range shape.linePrefixes {
		if strings.HasPrefix(trimmed, p) {
			return true
		}
	}
	return false
}

// markerLine returns the 1-based line number (within the file) of the first
// marker line in the block, or the block's start line if none is found —
// callers only ask after ParseComment confirmed a marker exists.
func (b commentBlock) markerLine() int {
	marker := expectedload.MarkerRegexp()
	for i, line := range b.lines {
		if marker.MatchString(line) {
			return b.startLine + i
		}
	}
	return b.startLine
}
