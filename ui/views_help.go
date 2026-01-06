package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// helpView renders the help screen
func (m Model) helpView() string {
	var b strings.Builder

	// Styles
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorWhite)).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 2).
		MarginBottom(1)

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

	// Title
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorWhite)).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 3).
		Render(" Agent Session Manager - Help ")

	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Quick reference - keyboard shortcuts in a row
	b.WriteString(sectionStyle.Render("  ⌨  Quick Reference"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 60)))
	b.WriteString("\n\n")

	// Row 1: Navigation
	navKeys := []string{
		keyStyle.Render("↑/k") + descStyle.Render(" up"),
		keyStyle.Render("↓/j") + descStyle.Render(" down"),
		keyStyle.Render("^↑") + descStyle.Render(" move"),
		keyStyle.Render("⇧↑") + descStyle.Render(" scroll"),
	}
	b.WriteString("  " + strings.Join(navKeys, "  "))
	b.WriteString("\n\n")

	// Row 2: Session actions
	actionKeys := []string{
		keyStyle.Render("↵") + descStyle.Render(" attach"),
		keyStyle.Render("n") + descStyle.Render(" new"),
		keyStyle.Render("s") + descStyle.Render(" start"),
		keyStyle.Render("a") + descStyle.Render(" replace/start"),
		keyStyle.Render("x") + descStyle.Render(" stop"),
		keyStyle.Render("d") + descStyle.Render(" delete"),
		keyStyle.Render("e") + descStyle.Render(" rename"),
	}
	b.WriteString("  " + strings.Join(actionKeys, "  "))
	b.WriteString("\n\n")

	// Row 3: Features
	featureKeys := []string{
		keyStyle.Render("r") + descStyle.Render(" resume"),
		keyStyle.Render("p") + descStyle.Render(" prompt"),
		keyStyle.Render("N") + descStyle.Render(" notes"),
		keyStyle.Render("c") + descStyle.Render(" color"),
		keyStyle.Render("g") + descStyle.Render(" new group"),
		keyStyle.Render("G") + descStyle.Render(" assign group"),
	}
	b.WriteString("  " + strings.Join(featureKeys, "  "))
	b.WriteString("\n\n")

	// Row 4: Toggles & Split View
	toggleKeys := []string{
		keyStyle.Render("l") + descStyle.Render(" compact"),
		keyStyle.Render("t") + descStyle.Render(" status"),
		keyStyle.Render("I") + descStyle.Render(" icons"),
		keyStyle.Render("v") + descStyle.Render(" split"),
		keyStyle.Render("m") + descStyle.Render(" mark"),
	}
	b.WriteString("  " + strings.Join(toggleKeys, "  "))
	b.WriteString("\n\n")

	// Row 5: Projects
	projectKeys := []string{
		keyStyle.Render("P") + descStyle.Render(" projects"),
		keyStyle.Render("i") + descStyle.Render(" import"),
	}
	b.WriteString("  " + strings.Join(projectKeys, "  "))
	b.WriteString("\n\n")

	// Row 6: Other
	otherKeys := []string{
		keyStyle.Render("?/F1") + descStyle.Render(" help"),
		keyStyle.Render("q") + descStyle.Render(" quit"),
		keyStyle.Render("R") + descStyle.Render(" resize"),
		keyStyle.Render("U") + descStyle.Render(" update"),
	}
	b.WriteString("  " + strings.Join(otherKeys, "  "))
	b.WriteString("\n\n")

	// Detailed sections
	b.WriteString(sectionStyle.Render("  📋 Detailed Descriptions"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 60)))
	b.WriteString("\n\n")

	details := []struct {
		key  string
		desc string
	}{
		{"↵ Enter", "Start session (if stopped) and attach to tmux session"},
		{"n New", "Create a new agent session with project path"},
		{"s Start", "Start session in background without attaching"},
		{"a Replace/Start", "Replace current session OR start parallel session"},
		{"r Resume", "Continue a previous session or start new"},
		{"p Prompt", "Send a message to running session without attaching"},
		{"N Notes", "Add/edit notes for the session (persists across resumes)"},
		{"c Color", "Customize session with colors and gradients"},
		{"g Group", "Create a new session group for organization"},
		{"G Assign", "Assign selected session to a group"},
		{"→ Right", "Expand a collapsed group"},
		{"← Left", "Collapse an expanded group"},
		{"l Compact", "Toggle compact view (less spacing between sessions)"},
		{"t Status", "Toggle status line visibility under sessions"},
		{"I Icons", "Toggle agent type icons in session list (Claude🤖 Gemini💎 etc.)"},
		{"^Y Yolo", "Toggle auto-approve/yolo mode on selected session (restarts if running)"},
		{"v Split", "Toggle split view to show two previews"},
		{"m Mark", "Mark session for split view (pinned on top)"},
		{"⇥ Tab", "Switch focus between split panels"},
		{"U Update", "Download and install new version (if available)"},
		{"P Projects", "Return to project selector"},
		{"i Import", "Import sessions from default into current project"},
		{"⇧↑/PgUp", "Scroll preview up"},
		{"⇧↓/PgDn", "Scroll preview down"},
		{"Home/End", "Jump to top/bottom of preview"},
		{"Ctrl+↑/↓", "Move session up/down in list"},
	}

	for _, d := range details {
		b.WriteString("  " + headerStyle.Render(d.key) + " " + descStyle.Render(d.desc) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("  🔗 In Attached Session"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 60)))
	b.WriteString("\n\n")

	b.WriteString("  " + keyStyle.Render("Ctrl+q") + descStyle.Render(" Quick detach (resizes preview, works everywhere)") + "\n")
	b.WriteString("  " + keyStyle.Render("Ctrl+b d") + descStyle.Render(" Standard tmux detach") + "\n")

	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("  ℹ  About"))
	b.WriteString("\n")
	b.WriteString(separatorStyle.Render("  " + strings.Repeat("─", 60)))
	b.WriteString("\n\n")

	b.WriteString(infoStyle.Render(fmt.Sprintf("  Agent Session Manager (%s) v%s", strings.ToUpper(AppName), AppVersion)))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  Manage multiple AI coding agents"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  Sessions stored in: ~/.config/agent-session-manager/"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  Built with Bubble Tea • github.com/izll/agent-session-manager"))
	b.WriteString("\n\n")

	// Get all content lines
	allContent := b.String()
	allLines := strings.Split(allContent, "\n")

	// Calculate visible area
	maxLines := m.height - 3 // -3 for footer and margins
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
		Render("Press ESC, ? or F1 to close" +
			func() string {
				if scrollInfo != "" {
					return " • " + scrollInfo
				}
				return ""
			}())
	visible.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, footer))

	return visible.String()
}
