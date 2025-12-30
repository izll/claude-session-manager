package ui

import (
	"regexp"
	"strings"
)

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// truncateWithANSI truncates a string to maxLen visible characters while preserving ANSI codes
func truncateWithANSI(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	var result strings.Builder
	visibleCount := 0
	i := 0
	runes := []rune(s)

	for i < len(runes) {
		// Check for ANSI escape sequence
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Find end of ANSI sequence
			start := i
			i += 2
			for i < len(runes) && !((runes[i] >= 'A' && runes[i] <= 'Z') || (runes[i] >= 'a' && runes[i] <= 'z')) {
				i++
			}
			if i < len(runes) {
				i++ // include the final letter
			}
			// Always include ANSI codes
			result.WriteString(string(runes[start:i]))
		} else {
			if visibleCount >= maxLen {
				result.WriteString("…")
				// Add reset code to ensure colors don't leak
				result.WriteString("\x1b[0m")
				break
			}
			result.WriteRune(runes[i])
			visibleCount++
			i++
		}
	}

	return result.String()
}
