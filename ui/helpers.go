package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// renderOverlayDialog renders a dialog box overlaid on the list view background
// Background is visible on all sides of the dialog
func (m Model) renderOverlayDialog(title string, boxContent string, boxWidth int, borderColor string) string {
	return m.renderOverlayDialogWithBackground(title, boxContent, boxWidth, borderColor, m.listView())
}

// renderOverlayDialogWithBackground renders a dialog box overlaid on a custom background
func (m Model) renderOverlayDialogWithBackground(title string, boxContent string, boxWidth int, borderColor string, background string) string {
	bgLines := strings.Split(background, "\n")

	// Ensure background has enough lines to cover the screen
	for len(bgLines) < m.height {
		bgLines = append(bgLines, strings.Repeat(" ", m.width))
	}

	// Create the box style
	dialogBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Background(lipgloss.Color("#1a1a2e")).
		Padding(0, 1).
		Width(boxWidth)

	box := dialogBoxStyle.Render(titleStyle.Render(title) + boxContent)
	boxLines := strings.Split(box, "\n")

	// Calculate position to center the box
	boxHeight := len(boxLines)
	startY := (m.height - boxHeight) / 2
	if startY < 0 {
		startY = 0
	}

	// Calculate horizontal start position (use first box line width as reference)
	boxDisplayWidth := displayWidth(boxLines[0])
	startX := (m.width - boxDisplayWidth) / 2
	if startX < 0 {
		startX = 0
	}

	// Overlay the box on the background
	for i, boxLine := range boxLines {
		bgY := startY + i
		if bgY >= 0 && bgY < len(bgLines) {
			origLine := bgLines[bgY]
			boxLineWidth := displayWidth(boxLine)

			// Get left part of background
			leftPart := truncateToWidth(origLine, startX)
			leftWidth := displayWidth(stripANSI(leftPart))

			// Pad left part if needed
			if leftWidth < startX {
				leftPart += strings.Repeat(" ", startX-leftWidth)
			}

			// Get right part of background
			rightStart := startX + boxLineWidth
			rightPart := skipToWidth(origLine, rightStart)

			bgLines[bgY] = leftPart + "\x1b[0m" + boxLine + "\x1b[0m" + rightPart
		}
	}

	return strings.Join(bgLines, "\n")
}

// displayWidth returns the display width of a string (accounting for double-width chars)
func displayWidth(s string) int {
	return runewidth.StringWidth(stripANSI(s))
}

// truncateToWidth truncates a string to fit within maxWidth display columns, preserving ANSI
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	var result strings.Builder
	width := 0
	runes := []rune(s)
	i := 0

	for i < len(runes) {
		// Check for ANSI escape sequence
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			start := i
			i += 2
			for i < len(runes) && !((runes[i] >= 'A' && runes[i] <= 'Z') || (runes[i] >= 'a' && runes[i] <= 'z')) {
				i++
			}
			if i < len(runes) {
				i++
			}
			result.WriteString(string(runes[start:i]))
		} else {
			charWidth := runewidth.RuneWidth(runes[i])
			if width+charWidth > maxWidth {
				break
			}
			result.WriteRune(runes[i])
			width += charWidth
			i++
		}
	}

	return result.String()
}

// skipToWidth skips characters until reaching startWidth display columns, then returns the rest
func skipToWidth(s string, startWidth int) string {
	var result strings.Builder
	width := 0
	runes := []rune(s)
	i := 0
	started := false

	for i < len(runes) {
		// Check for ANSI escape sequence
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			start := i
			i += 2
			for i < len(runes) && !((runes[i] >= 'A' && runes[i] <= 'Z') || (runes[i] >= 'a' && runes[i] <= 'z')) {
				i++
			}
			if i < len(runes) {
				i++
			}
			if started {
				result.WriteString(string(runes[start:i]))
			}
		} else {
			charWidth := runewidth.RuneWidth(runes[i])
			if width >= startWidth {
				started = true
				result.WriteRune(runes[i])
			}
			width += charWidth
			i++
		}
	}

	return result.String()
}

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
