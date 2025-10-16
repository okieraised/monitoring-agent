package utilities

import "strings"

func StripArraySuffix(s string) string {
	// Trim successive trailing valid array suffixes: [], [N], [<=N]
	for {
		lb := strings.LastIndex(s, "[")
		if lb < 0 || !strings.HasSuffix(s, "]") || lb == 0 {
			break
		}
		inner := strings.TrimSpace(s[lb+1 : len(s)-1])
		ok := inner == "" || AllDigits(inner) || (strings.HasPrefix(inner, "<=") && AllDigits(strings.TrimSpace(inner[2:])))
		if ok {
			s = s[:lb]
			continue
		}
		break
	}
	return s
}

func AllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
