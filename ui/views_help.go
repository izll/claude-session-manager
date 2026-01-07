package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// buildHelpContent generates the help content and returns the content string and line count
func buildHelpContent(width int) (string, int) {
	var b strings.Builder

	// Styles
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPurple)).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA"))

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444"))

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorLightGray)).
		Italic(true)

	noteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorYellow)).
		Italic(true)

	// Title
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorWhite)).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 3).
		Render(" Agent Session Manager - Help ")

	b.WriteString(lipgloss.PlaceHorizontal(width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Helper for rendering key-description pairs
	renderKey := func(key, desc string) string {
		return keyStyle.Render(key) + " " + descStyle.Render(desc)
	}

	// Column positions for alignment
	const col2Start = 38 // Second column starts here

	// Helper for two-column layout
	renderRow := func(key1, desc1, key2, desc2 string) string {
		left := renderKey(key1, desc1)
		leftLen := lipgloss.Width(left)
		padding := col2Start - leftLen
		if padding < 2 {
			padding = 2
		}
		return "  " + left + strings.Repeat(" ", padding) + renderKey(key2, desc2)
	}

	// ═══════════════════════════════════════════════════════════════════
	// NAVIGATION
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  Navigation"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString(renderRow("↑/k ↓/j", "Move up/down", "Ctrl+↑/↓", "Reorder session"))
	b.WriteString("\n")
	b.WriteString(renderRow("Alt+↑/↓", "Scroll line", "PgUp/PgDn", "Scroll half page"))
	b.WriteString("\n")
	b.WriteString("  " + renderKey("Home/End", "Scroll to top/bottom"))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════
	// SESSION ACTIONS
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  Session Actions"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString("  " + renderKey("Enter", "Attach (starts if stopped)"))
	b.WriteString("\n")
	b.WriteString(renderRow("n", "New session", "e", "Rename session"))
	b.WriteString("\n")
	b.WriteString(renderRow("s", "Start (background)", "a", "Replace/parallel start"))
	b.WriteString("\n")
	b.WriteString(renderRow("x", "Stop", "d", "Delete"))
	b.WriteString("\n")
	b.WriteString("  " + noteStyle.Render("     ↳ x/d asks session or tab when multiple tabs exist"))
	b.WriteString("\n")
	b.WriteString(renderRow("r", "Resume conversation", "p", "Send prompt"))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════
	// TABS (Multiple windows per session)
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  Tabs"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString("  " + renderKey("t", "New tab (Agent or Terminal)"))
	b.WriteString("\n")
	b.WriteString(renderRow("T", "Rename tab", "W", "Quick close tab"))
	b.WriteString("\n")
	b.WriteString(renderRow("Alt+←/→", "Switch tabs", "Ctrl+F", "Toggle tracking"))
	b.WriteString("\n")
	b.WriteString("  " + noteStyle.Render("     ↳ Stopped tabs show ○ indicator, remain visible"))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════
	// GROUPS
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  Groups"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString(renderRow("g", "Create group", "G", "Assign to group"))
	b.WriteString("\n")
	b.WriteString(renderRow("→", "Expand group", "←", "Collapse group"))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════
	// CUSTOMIZATION
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  Customization"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString(renderRow("N", "Edit notes", "c", "Colors & gradients"))
	b.WriteString("\n")
	b.WriteString("  " + noteStyle.Render("     ↳ N edits tab notes when multiple tabs exist"))
	b.WriteString("\n")
	b.WriteString(renderRow("l", "Compact mode", "o", "Toggle status lines"))
	b.WriteString("\n")
	b.WriteString(renderRow("I", "Toggle icons", "^Y", "Toggle YOLO mode"))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════
	// SPLIT VIEW
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  Split View"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString(renderRow("v", "Toggle split", "m", "Mark/pin session"))
	b.WriteString("\n")
	b.WriteString("  " + renderKey("Tab", "Switch focus between panes"))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════
	// DIFF VIEW
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  Diff View"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString(renderRow("D", "Toggle Preview/Diff", "F", "Switch Session/Full diff"))
	b.WriteString("\n")
	b.WriteString("  " + noteStyle.Render("     ↳ Session diff: changes since session start"))
	b.WriteString("\n")
	b.WriteString("  " + noteStyle.Render("     ↳ Full diff: all uncommitted changes"))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════
	// PROJECTS & OTHER
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  Projects & Other"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString(renderRow("q", "Quit to projects", "i", "Import sessions"))
	b.WriteString("\n")
	b.WriteString(renderRow("U", "Check updates", "R", "Force resize"))
	b.WriteString("\n")
	b.WriteString("  " + renderKey("?", "Help"))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════
	// ATTACHED SESSION
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  Inside Attached Session"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString("  " + renderKey("Ctrl+q", "Quick detach (auto-resizes preview)"))
	b.WriteString("\n")
	b.WriteString("  " + renderKey("Ctrl+b d", "Standard tmux detach"))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════
	// STATUS INDICATORS
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  Status Indicators"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString("  " + activeStyle.Render("●") + descStyle.Render(" Busy (working)") + "    ")
	b.WriteString(waitingStyle.Render("●") + descStyle.Render(" Waiting (needs input)"))
	b.WriteString("\n")
	b.WriteString("  " + idleStyle.Render("●") + descStyle.Render(" Idle (ready)") + "      ")
	b.WriteString(stoppedStyle.Render("○") + descStyle.Render(" Stopped"))
	b.WriteString("\n\n")

	// ═══════════════════════════════════════════════════════════════════
	// ABOUT
	// ═══════════════════════════════════════════════════════════════════
	b.WriteString(sectionStyle.Render("  About"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 65)))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render(fmt.Sprintf("  %s v%s", strings.ToUpper(AppName), AppVersion)))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  Manage multiple AI coding agents (Claude, Gemini, Aider, etc.)"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  github.com/izll/agent-session-manager"))
	b.WriteString("\n\n")

	content := b.String()
	lineCount := len(strings.Split(content, "\n"))
	return content, lineCount
}

// helpView renders the help screen
func (m Model) helpView() string {
	// Get help content
	allContent, _ := buildHelpContent(m.width)
	allLines := strings.Split(allContent, "\n")

	// Calculate visible area
	maxLines := m.height - 3
	if maxLines < 10 {
		maxLines = 10
	}

	// Apply scroll
	startIdx := m.helpScroll
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + maxLines
	if endIdx > len(allLines) {
		endIdx = len(allLines)
		startIdx = endIdx - maxLines
		if startIdx < 0 {
			startIdx = 0
		}
	}

	// Build visible content
	var visible strings.Builder
	for i := startIdx; i < endIdx && i < len(allLines); i++ {
		visible.WriteString(allLines[i])
		visible.WriteString("\n")
	}

	// Footer with scroll indicator
	scrollInfo := ""
	if len(allLines) > maxLines {
		if startIdx > 0 {
			scrollInfo = "↑ "
		}
		scrollInfo += fmt.Sprintf("Line %d-%d of %d", startIdx+1, endIdx, len(allLines))
		if endIdx < len(allLines) {
			scrollInfo += " ↓"
		}
	}
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGray)).
		Render("Press ESC or ? to close" +
			func() string {
				if scrollInfo != "" {
					return " • " + scrollInfo
				}
				return ""
			}())
	visible.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, footer))

	return visible.String()
}
