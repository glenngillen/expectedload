package expectedload

import (
	"strings"
)

// normalizeKey converts a field name from any supported casing convention into
// canonical snake_case: `monthly-calls`, `monthlyCalls`, `monthly_calls` and
// `Monthly Calls` all become `monthly_calls`.
func normalizeKey(key string) string {
	key = strings.TrimSpace(key)
	var b strings.Builder
	b.Grow(len(key) + 4)

	prevLower := false
	for i, r := range key {
		switch {
		case r == '-' || r == ' ' || r == '.':
			b.WriteByte('_')
			prevLower = false
		case r >= 'A' && r <= 'Z':
			// Insert a separator at a lower→upper boundary (camelCase) but not
			// at the very start or right after an existing separator.
			if prevLower && i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
			prevLower = false
		default:
			b.WriteRune(r)
			prevLower = r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		}
	}

	// Collapse any accidental double underscores.
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	return strings.Trim(out, "_")
}

// parseLoadValue parses an integer load value, tolerating numeric underscore
// separators (`5_000_000`) and surrounding whitespace.
func parseLoadValue(raw string) (int64, bool) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return 0, false
	}

	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}

	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int64(r-'0')
	}
	if neg {
		n = -n
	}
	return n, true
}

// closestField returns the known field within edit distance 2 of key, for
// "did you mean" diagnostics. Returns "" when nothing is close.
func closestField(key string) string {
	best := ""
	bestDist := 3 // only suggest within distance <= 2
	for _, known := range knownFields {
		d := levenshtein(key, known)
		if d < bestDist {
			bestDist = d
			best = known
		}
	}
	return best
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
