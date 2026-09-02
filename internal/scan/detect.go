package scan

import (
	"bufio"
	"bytes"
	"io"
	"os"

	"github.com/infracost/expectedload"
)

// sniffLimit bounds how much of a file Detect reads. A marker beyond the first
// 256 KB is missed at detect time — a documented trade-off to keep detection
// under the SDK's 100 ms budget on arbitrarily large files.
const sniffLimit = 256 * 1024

// DetectContent reports whether the given file content contains an
// expected-load marker. Only the first sniffLimit bytes are examined. The
// content may be binary or non-UTF-8; the sniff never errors.
func DetectContent(content []byte) bool {
	if len(content) > sniffLimit {
		content = content[:sniffLimit]
	}
	return sniffReader(bytes.NewReader(content))
}

// DetectPath reports whether path is a supported file (or a directory with a
// supported top-level file) containing an expected-load marker. It never
// returns an error: unknown, unreadable, and nonexistent paths are simply not
// detected.
func DetectPath(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return detectDir(path)
	}
	return detectFile(path)
}

// detectFile sniffs a single file, provided its extension is supported.
func detectFile(path string) bool {
	if _, ok := SyntaxForPath(path); !ok {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	return sniffReader(io.LimitReader(f, sniffLimit))
}

// detectDir scans only the directory's top-level entries (no recursion, per
// the SDK detection contract) for one supported file bearing a marker.
func detectDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if detectFile(dir + string(os.PathSeparator) + e.Name()) {
			return true
		}
	}
	return false
}

// sniffReader scans r line by line for the marker. The marker regexp is
// line-oriented ($ is end-of-text without (?m)), so matching the whole buffer
// at once would only find markers on the final line.
func sniffReader(r io.Reader) bool {
	marker := expectedload.MarkerRegexp()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), sniffLimit)
	for sc.Scan() {
		if marker.Match(sc.Bytes()) {
			return true
		}
	}
	// Scanner errors (e.g. a "line" longer than the buffer in a binary file)
	// mean "not detected", never a crash.
	_ = sc.Err()
	return false
}
