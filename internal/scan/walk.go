package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/infracost/expectedload"
)

// Declaration is one parsed expected-load block located in a source file.
type Declaration struct {
	// File is the path relative to the scan root (or to repo_directory when
	// the caller rebases it), using forward slashes.
	File string
	// Line is the 1-based line number of the marker line.
	Line int
	// Load is the parsed declaration (never nil).
	Load *expectedload.ExpectedLoad
}

// Diagnostic is a library diagnostic annotated with the file (and, when known,
// the marker line) it came from. File-level failures (unreadable file) use
// Line 0 and Severity Error.
type Diagnostic struct {
	expectedload.Diagnostic
	File string
	Line int
	// ReadFailure marks filesystem-level failures, so the transport layer can
	// classify them separately from declaration parse problems.
	ReadFailure bool
}

// Result aggregates a scan: every declaration found plus every diagnostic.
// Partial results are the norm — one broken file never aborts a scan.
type Result struct {
	Declarations []Declaration
	Diagnostics  []Diagnostic
}

// skipDirs are well-known vendor/metadata directories that never hold
// first-party declarations.
var skipDirs = map[string]struct{}{
	"node_modules": {},
	"vendor":       {},
	".git":         {},
	".terraform":   {},
}

// Scan walks path (a directory or a single supported file) and parses every
// expected-load declaration found. It never returns an error: unreadable files
// become diagnostics.
func Scan(path string) Result {
	var res Result
	info, err := os.Stat(path)
	if err != nil {
		res.Diagnostics = append(res.Diagnostics, readFailure(filepath.Base(path), err))
		return res
	}

	if !info.IsDir() {
		scanFile(&res, path, filepath.Base(path))
		return res
	}

	root := path
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			rel := relPath(root, p)
			res.Diagnostics = append(res.Diagnostics, readFailure(rel, err))
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip && p != root {
				return fs.SkipDir
			}
			return nil
		}
		scanFile(&res, p, relPath(root, p))
		return nil
	})
	return res
}

// scanFile parses one file's comment blocks into res. rel is the path recorded
// on declarations and diagnostics.
func scanFile(res *Result, path, rel string) {
	syntax, ok := SyntaxForPath(path)
	if !ok {
		return
	}
	src, err := os.ReadFile(path)
	if err != nil {
		res.Diagnostics = append(res.Diagnostics, readFailure(rel, err))
		return
	}

	for _, block := range extractComments(syntax, string(src)) {
		load, diags := expectedload.ParseComment(syntax, strings.Join(block.lines, "\n"))
		if load == nil {
			continue // no marker in this comment — normal, not diagnosable
		}
		line := block.markerLine()
		res.Declarations = append(res.Declarations, Declaration{File: rel, Line: line, Load: load})
		for _, d := range diags {
			res.Diagnostics = append(res.Diagnostics, Diagnostic{Diagnostic: d, File: rel, Line: line})
		}
	}
}

func readFailure(rel string, err error) Diagnostic {
	return Diagnostic{
		Diagnostic: expectedload.Diagnostic{
			Severity: expectedload.Error,
			Message:  "cannot read " + rel + ": " + err.Error(),
		},
		File:        rel,
		ReadFailure: true,
	}
}

func relPath(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		rel = p
	}
	return filepath.ToSlash(rel)
}
