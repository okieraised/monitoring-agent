package utilities

import (
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Parse parses duration strings like "5m", "1h30m", "250ms" using time.ParseDuration.
// If the input is a bare integer (e.g., "5" or "  42 "), it is treated as seconds.
func Parse(s string) (time.Duration, error) {
	in := strings.TrimSpace(s)
	if in == "" {
		return 0, &time.ParseError{Layout: "duration", Value: s, LayoutElem: "", ValueElem: "", Message: "empty duration"}
	}

	// If it's a bare integer (with optional leading +/-, spaces), treat as seconds.
	if isBareInt(in) {
		secs, err := strconv.ParseInt(strings.TrimSpace(in), 10, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(secs) * time.Second, nil
	}

	// Fall back to Go's duration parser (supports ns, us/µs, ms, s, m, h, with combos).
	return time.ParseDuration(in)
}

// ParseOrDefault parses like Parse(), but returns def when input is empty.
// If parsing fails for a non-empty input, the error is returned.
func ParseOrDefault(s string, def time.Duration) (time.Duration, error) {
	if strings.TrimSpace(s) == "" {
		return def, nil
	}
	return Parse(s)
}

// MustParse is like Parse but panics on error.
func MustParse(s string) time.Duration {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// IsValid reports whether s can be parsed by Parse().
func IsValid(s string) bool {
	_, err := Parse(s)
	return err == nil
}

func isBareInt(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Optional leading sign
	if s[0] == '+' || s[0] == '-' {
		s = s[1:]
		if s == "" {
			return false
		}
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
