package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/izll/agent-session-manager/session"
)

// truncateRunes truncates a string to maxLen runes and adds ellipsis if needed
func truncateRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// View implements tea.Model and renders the current UI state.
// It returns different views based on the current application state.
func (m Model) View() string {
	switch m.state {
	case stateHelp:
		return m.helpView()
	case stateConfirmDelete:
		return m.confirmDeleteView()
	case stateNewName, stateNewPath:
		return m.newInstanceView()
	case stateRename:
		return m.renameView()
	case stateSelectClaudeSession:
		return m.selectSessionView()
	case stateColorPicker:
		return m.colorPickerView()
	case statePrompt:
		return m.promptView()
	case stateNewGroup:
		return m.newGroupView()
	case stateRenameGroup:
		return m.renameGroupView()
	case stateSelectGroup:
		return m.selectGroupView()
	case stateSelectAgent:
		return m.selectAgentView()
	case stateCustomCmd:
		return m.customCmdView()
	case stateError:
		return m.errorView()
	default:
		return m.listView()
	}
}

// listView renders the main split-pane view with session list and preview
func (m Model) listView() string {
	listWidth := ListPaneWidth
	previewWidth := m.calculatePreviewWidth()
	contentHeight := m.height - 1
	if contentHeight < MinContentHeight {
		contentHeight = MinContentHeight
	}

	// Build panes using helper methods
	leftPane := m.buildSessionListPane(listWidth, contentHeight)
	rightPane := m.buildPreviewPane(contentHeight)

	// Style the panes with borders
	leftStyled := listPaneStyle.
		Width(listWidth).
		Height(contentHeight).
		Render(leftPane)

	rightStyled := previewPaneStyle.
		Width(previewWidth).
		Height(contentHeight).
		Render(rightPane)

	// Join panes horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)

	// Build final view
	var b strings.Builder
	b.WriteString(content)

	// Error display - only show in list state (dialogs show their own errors)
	if m.err != nil && m.state == stateList {
		errBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#FF0000")).
			Bold(true).
			Padding(0, 2).
			Render(fmt.Sprintf(" ⚠ Error: %v ", m.err))
		b.WriteString("\n" + errBox + "\n")
	}

	// Status bar
	b.WriteString(m.buildStatusBar())

	return b.String()
}

// helpView renders the help screen
func (m Model) helpView() string {
	var b strings.Builder

	// Styles
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true).
		Padding(0, 2).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA"))

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444"))

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true)

	// Title
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
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
		keyStyle.Render("⇧↑/K") + descStyle.Render(" move up"),
		keyStyle.Render("⇧↓/J") + descStyle.Render(" move down"),
	}
	b.WriteString("  " + strings.Join(navKeys, "  "))
	b.WriteString("\n\n")

	// Row 2: Session actions
	actionKeys := []string{
		keyStyle.Render("↵") + descStyle.Render(" attach"),
		keyStyle.Render("n") + descStyle.Render(" new"),
		keyStyle.Render("s") + descStyle.Render(" start"),
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
		keyStyle.Render("c") + descStyle.Render(" color"),
		keyStyle.Render("g") + descStyle.Render(" new group"),
		keyStyle.Render("G") + descStyle.Render(" assign group"),
	}
	b.WriteString("  " + strings.Join(featureKeys, "  "))
	b.WriteString("\n\n")

	// Row 4: Toggles
	toggleKeys := []string{
		keyStyle.Render("l") + descStyle.Render(" compact"),
		keyStyle.Render("y") + descStyle.Render(" autoyes"),
		keyStyle.Render("→") + descStyle.Render(" expand"),
		keyStyle.Render("←") + descStyle.Render(" collapse"),
	}
	b.WriteString("  " + strings.Join(toggleKeys, "  "))
	b.WriteString("\n\n")

	// Row 5: Other
	otherKeys := []string{
		keyStyle.Render("?/F1") + descStyle.Render(" help"),
		keyStyle.Render("q") + descStyle.Render(" quit"),
		keyStyle.Render("R") + descStyle.Render(" resize"),
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
		{"n New", "Create a new Claude Code session with project path"},
		{"r Resume", "Continue a previous Claude conversation"},
		{"p Prompt", "Send a message to running session without attaching"},
		{"c Color", "Customize session with colors and gradients"},
		{"g Group", "Create a new session group for organization"},
		{"G Assign", "Assign selected session to a group"},
		{"→ Right", "Expand a collapsed group"},
		{"← Left", "Collapse an expanded group"},
		{"l Compact", "Toggle compact view (less spacing between sessions)"},
		{"t Status", "Toggle status line visibility under sessions"},
		{"y AutoYes", "Toggle --dangerously-skip-permissions flag"},
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

	b.WriteString(infoStyle.Render("  Agent Session Manager (ASM) - Manage multiple AI coding agents"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  Sessions stored in: ~/.config/agent-session-manager/"))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render("  Built with Bubble Tea • github.com/izll/agent-session-manager"))
	b.WriteString("\n\n")

	// Footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render("Press ESC, ? or F1 to close")
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, footer))

	return b.String()
}

// confirmDeleteView renders the delete confirmation dialog as an overlay
func (m Model) confirmDeleteView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n")
	if m.deleteTarget != nil {
		boxContent.WriteString(fmt.Sprintf("  Delete session '%s'?\n\n", m.deleteTarget.Name))
	}
	boxContent.WriteString(helpStyle.Render("  y: yes  n: no"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Confirm Delete ", boxContent.String(), 40, "#FF5F87")
}

// newInstanceView renders the new session creation dialog as an overlay
func (m Model) newInstanceView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n")

	if m.state == stateNewPath {
		boxContent.WriteString("  Project Path:\n")
		boxContent.WriteString("  " + m.pathInput.View() + "\n")
	} else {
		boxContent.WriteString(fmt.Sprintf("  Path: %s\n\n", m.pathInput.Value()))
		boxContent.WriteString("  Session Name:\n")
		boxContent.WriteString("  " + m.nameInput.View() + "\n")
	}

	boxContent.WriteString("\n")
	boxContent.WriteString(helpStyle.Render("  enter: confirm  esc: cancel"))
	boxContent.WriteString("\n")

	boxWidth := 60
	if m.width > 80 {
		boxWidth = m.width / 2
	}
	if boxWidth > 80 {
		boxWidth = 80
	}

	return m.renderOverlayDialog(" New Session ", boxContent.String(), boxWidth, "#7D56F4")
}

// selectSessionView renders the Claude session selector
func (m Model) selectSessionView() string {
	var b strings.Builder

	// Header like Claude Code
	b.WriteString("Resume Session\n")

	// Search box (visual only for now)
	boxWidth := 80
	if m.width > 20 {
		boxWidth = m.width - 10
	}
	if boxWidth > 150 {
		boxWidth = 150
	}
	b.WriteString(searchBoxStyle.Width(boxWidth).Render("⌕ Search…"))
	b.WriteString("\n\n")

	// Calculate visible window
	maxVisible := SessionListMaxItems
	startIdx := 0
	if m.sessionCursor > maxVisible-2 {
		startIdx = m.sessionCursor - maxVisible + 2
	}
	if startIdx < 0 {
		startIdx = 0
	}

	totalItems := len(m.claudeSessions) + 1 // +1 for "new session"

	// Option 0: Start new session
	if startIdx == 0 {
		otherCount := len(m.claudeSessions)
		suffix := ""
		if otherCount > 0 {
			suffix = fmt.Sprintf(" (+%d other sessions)", otherCount)
		}

		if m.sessionCursor == 0 {
			b.WriteString(fmt.Sprintf("❯ ▶ Start new session%s\n", suffix))
		} else {
			b.WriteString(fmt.Sprintf("  Start new session%s\n", suffix))
		}
		b.WriteString("\n")
	}

	// List existing sessions
	visibleCount := 1
	for i, cs := range m.claudeSessions {
		itemIdx := i + 1

		if itemIdx < startIdx {
			continue
		}
		if visibleCount >= maxVisible {
			break
		}

		// Use last prompt (like Claude Code does)
		prompt := cs.LastPrompt
		if prompt == "" {
			prompt = cs.FirstPrompt
		}
		maxPromptLen := 80
		if m.width > 40 {
			maxPromptLen = m.width - 40
		}
		if len([]rune(prompt)) > maxPromptLen {
			prompt = truncateRunes(prompt, maxPromptLen)
		}

		timeAgo := formatTimeAgo(cs.UpdatedAt)
		msgText := "messages"
		if cs.MessageCount == 1 {
			msgText = "message"
		}

		// Format like Claude Code
		if itemIdx == m.sessionCursor {
			b.WriteString(selectedPromptStyle.Render(fmt.Sprintf("❯ ▶ %s", prompt)))
			b.WriteString("\n")
			b.WriteString(metaStyle.Render(fmt.Sprintf("  %s · %d %s", timeAgo, cs.MessageCount, msgText)))
			b.WriteString("\n\n")
		} else {
			b.WriteString(fmt.Sprintf("  %s\n", prompt))
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %s · %d %s", timeAgo, cs.MessageCount, msgText)))
			b.WriteString("\n\n")
		}
		visibleCount++
	}

	// Show more indicator
	remaining := totalItems - startIdx - maxVisible
	if remaining > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more sessions\n", remaining)))
	}

	return b.String()
}

// colorPickerView renders the color picker dialog
func (m Model) colorPickerView() string {
	var b strings.Builder

	// Title based on what we're editing
	if m.editingGroup != nil {
		b.WriteString(titleStyle.Render(" Group Color "))
	} else {
		b.WriteString(titleStyle.Render(" Session Color "))
	}
	b.WriteString("\n\n")

	// Editing a group
	if m.editingGroup != nil {
		group := m.editingGroup

		// Get preview colors (current cursor selection for active mode)
		previewFg := m.previewFg
		previewBg := m.previewBg
		filteredColors := m.getFilteredColorOptions()
		if m.colorCursor < len(filteredColors) {
			selected := filteredColors[m.colorCursor]
			if m.colorMode == 0 {
				previewFg = selected.Color
			} else {
				previewBg = selected.Color
			}
		}

		// Show group name with preview colors
		styledName := group.Name
		nameStyle := lipgloss.NewStyle().Bold(true)
		hasBg := previewBg != "" && previewBg != "none"
		if hasBg {
			nameStyle = nameStyle.Background(lipgloss.Color(previewBg))
		}
		if previewFg != "" && previewFg != "none" && previewFg != "auto" {
			nameStyle = nameStyle.Foreground(lipgloss.Color(previewFg))
		} else if hasBg {
			// Auto or empty foreground with background - use contrast
			nameStyle = nameStyle.Foreground(lipgloss.Color(getContrastColor(previewBg)))
		}
		styledName = nameStyle.Render(group.Name)

		b.WriteString(fmt.Sprintf("  Group: 📁 %s\n", styledName))

		// Show current colors
		fgDisplay := "none"
		if group.Color != "" {
			fgDisplay = group.Color
		}
		bgDisplay := "none"
		if group.BgColor != "" {
			bgDisplay = group.BgColor
		}
		fullRowDisplay := "OFF"
		if group.FullRowColor {
			fullRowDisplay = "ON"
		}

		// Highlight active mode
		if m.colorMode == 0 {
			b.WriteString(fmt.Sprintf("  [Text: %s]  Background: %s\n", fgDisplay, bgDisplay))
		} else {
			b.WriteString(fmt.Sprintf("   Text: %s  [Background: %s]\n", fgDisplay, bgDisplay))
		}
		b.WriteString(fmt.Sprintf("  Full row: %s (f)\n", fullRowDisplay))
		b.WriteString(dimStyle.Render("  TAB: switch | f: full row"))
		b.WriteString("\n\n")
	} else if inst := m.getSelectedInstance(); inst != nil {
		// Get preview colors (current cursor selection for active mode)
		previewFg := m.previewFg
		previewBg := m.previewBg
		filteredColors := m.getFilteredColorOptions()
		if m.colorCursor < len(filteredColors) {
			selected := filteredColors[m.colorCursor]
			if m.colorMode == 0 {
				previewFg = selected.Color
			} else {
				previewBg = selected.Color
			}
		}

		// Show session name with preview colors
		styledName := inst.Name
		nameStyle := lipgloss.NewStyle()
		if previewBg != "" {
			nameStyle = nameStyle.Background(lipgloss.Color(previewBg))
		}
		if previewFg != "" {
			if previewFg == "auto" && previewBg != "" {
				nameStyle = nameStyle.Foreground(lipgloss.Color(getContrastColor(previewBg)))
			} else if _, isGradient := gradients[previewFg]; isGradient {
				if previewBg != "" {
					styledName = applyGradientWithBg(inst.Name, previewFg, previewBg)
				} else {
					styledName = applyGradient(inst.Name, previewFg)
				}
			} else {
				nameStyle = nameStyle.Foreground(lipgloss.Color(previewFg))
			}
		} else if previewBg != "" {
			nameStyle = nameStyle.Foreground(lipgloss.Color(getContrastColor(previewBg)))
		}
		if styledName == inst.Name {
			styledName = nameStyle.Render(inst.Name)
		}

		b.WriteString(fmt.Sprintf("  Session: %s\n", styledName))

		// Show current colors
		fgDisplay := "none"
		if inst.Color != "" {
			fgDisplay = inst.Color
		}
		bgDisplay := "none"
		if inst.BgColor != "" {
			bgDisplay = inst.BgColor
		}
		fullRowDisplay := "OFF"
		if inst.FullRowColor {
			fullRowDisplay = "ON"
		}

		// Highlight active mode
		if m.colorMode == 0 {
			b.WriteString(fmt.Sprintf("  [Text: %s]  Background: %s\n", fgDisplay, bgDisplay))
		} else {
			b.WriteString(fmt.Sprintf("   Text: %s  [Background: %s]\n", fgDisplay, bgDisplay))
		}
		b.WriteString(fmt.Sprintf("  Full row: %s (f)\n", fullRowDisplay))
		b.WriteString(dimStyle.Render("  TAB: switch | f: full row"))
		b.WriteString("\n\n")
	}

	// Calculate max items based on mode
	maxItems := m.getMaxColorItems()

	// Calculate visible window
	maxVisible := m.height - ColorPickerHeader
	if maxVisible < MinColorPickerRows {
		maxVisible = MinColorPickerRows
	}

	startIdx := 0
	if m.colorCursor >= maxVisible {
		startIdx = m.colorCursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > maxItems {
		endIdx = maxItems
	}

	// Show scroll indicator at top
	if startIdx > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more\n", startIdx)))
	}

	// Get filtered list of color options for current mode
	filteredOptions := m.getFilteredColorOptions()

	for displayIdx := startIdx; displayIdx < endIdx && displayIdx < len(filteredOptions); displayIdx++ {
		c := filteredOptions[displayIdx]

		// Create color preview
		var colorPreview string
		if c.Color == "" {
			if m.colorMode == 0 {
				colorPreview = "      none"
			} else {
				colorPreview = "       none" // Extra space for background mode
			}
		} else if c.Color == "auto" {
			colorPreview = " ✨   auto"
		} else if _, isGradient := gradients[c.Color]; isGradient {
			// Show gradient preview
			colorPreview = " " + applyGradient("████", c.Color) + " " + c.Name
		} else {
			style := lipgloss.NewStyle()
			if m.colorMode == 0 {
				style = style.Foreground(lipgloss.Color(c.Color))
				colorPreview = style.Render(" ████ ") + c.Name
			} else {
				style = style.Background(lipgloss.Color(c.Color))
				// For background, show solid block with contrast text
				textColor := getContrastColor(c.Color)
				style = style.Foreground(lipgloss.Color(textColor))
				colorPreview = style.Render("      ") + " " + c.Name
			}
		}

		if displayIdx == m.colorCursor {
			b.WriteString(fmt.Sprintf("  ❯%s\n", colorPreview))
		} else {
			b.WriteString(fmt.Sprintf("   %s\n", colorPreview))
		}
	}

	// Show scroll indicator at bottom
	remaining := maxItems - endIdx
	if remaining > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more\n", remaining)))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  enter: select  esc: cancel"))

	return b.String()
}

// renameView renders the rename dialog as an overlay
func (m Model) renameView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n")

	if inst := m.getSelectedInstance(); inst != nil {
		boxContent.WriteString(fmt.Sprintf("  Current: %s\n\n", inst.Name))
	}

	boxContent.WriteString("  New Name:\n")
	boxContent.WriteString("  " + m.nameInput.View() + "\n")
	boxContent.WriteString("\n")
	boxContent.WriteString(helpStyle.Render("  enter: confirm  esc: cancel"))
	boxContent.WriteString("\n")

	boxWidth := 50
	if m.width > 80 {
		boxWidth = m.width / 3
	}
	if boxWidth > 60 {
		boxWidth = 60
	}

	return m.renderOverlayDialog(" Rename Session ", boxContent.String(), boxWidth, "#7D56F4")
}

// promptView renders the prompt input dialog overlaid on the list view
func (m Model) promptView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n")

	if inst := m.getSelectedInstance(); inst != nil {
		boxContent.WriteString(fmt.Sprintf("  Session: %s\n\n", inst.Name))
	}

	boxContent.WriteString("  Message:\n")
	boxContent.WriteString("  > " + m.promptInput.View() + "\n\n")
	boxContent.WriteString(helpStyle.Render("  enter: send  esc: cancel"))
	boxContent.WriteString("\n")

	boxWidth := 60
	if m.width > 80 {
		boxWidth = m.width / 2
	}
	if boxWidth > 80 {
		boxWidth = 80
	}

	return m.renderOverlayDialog(" Send Message ", boxContent.String(), boxWidth, "#7D56F4")
}

// renderSessionRow renders a single session row with all color and style logic
func (m Model) renderSessionRow(inst *session.Instance, index int, listWidth int) string {
	var row strings.Builder

	// Status indicator based on activity
	var status string
	if inst.Status == session.StatusRunning {
		if m.isActive[inst.ID] {
			status = activeStyle.Render("●") // Orange - active
		} else {
			status = idleStyle.Render("●") // Grey - idle/waiting
		}
	} else {
		status = stoppedStyle.Render("○") // Red outline - stopped
	}

	// Truncate name to fit
	name := inst.Name
	maxNameLen := listWidth - 6
	if maxNameLen < 10 {
		maxNameLen = 10
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen-2] + "…"
	}

	// Apply session colors
	styledName := m.getStyledName(inst, name)
	selected := index == m.cursor

	// Render the row
	if selected {
		row.WriteString(m.renderSelectedRow(inst, name, styledName, status, listWidth))
	} else {
		row.WriteString(m.renderUnselectedRow(inst, name, styledName, status, listWidth))
	}
	row.WriteString("\n")

	// Show last output line
	lastLine := m.getLastLine(inst)
	row.WriteString(fmt.Sprintf("     └─ %s", lastLine))
	row.WriteString("\n")

	if !m.compactList {
		row.WriteString("\n")
	}

	return row.String()
}

// getStyledName applies color styling to a session name
func (m Model) getStyledName(inst *session.Instance, name string) string {
	style := lipgloss.NewStyle()

	// Apply background color first
	if inst.BgColor != "" {
		style = style.Background(lipgloss.Color(inst.BgColor))
	}

	// Apply foreground color
	if inst.Color != "" {
		if inst.Color == "auto" && inst.BgColor != "" {
			autoColor := getContrastColor(inst.BgColor)
			style = style.Foreground(lipgloss.Color(autoColor))
			return style.Render(name)
		} else if _, isGradient := gradients[inst.Color]; isGradient {
			if inst.BgColor != "" {
				return applyGradientWithBg(name, inst.Color, inst.BgColor)
			}
			return applyGradient(name, inst.Color)
		}
		style = style.Foreground(lipgloss.Color(inst.Color))
		return style.Render(name)
	} else if inst.BgColor != "" {
		autoColor := getContrastColor(inst.BgColor)
		style = style.Foreground(lipgloss.Color(autoColor))
		return style.Render(name)
	}

	return name
}

// renderSelectedRow renders a selected session row
func (m Model) renderSelectedRow(inst *session.Instance, name, styledName, status string, listWidth int) string {
	if inst.FullRowColor && inst.BgColor != "" {
		if _, isGradient := gradients[inst.Color]; isGradient {
			padding := listWidth - 7 - len([]rune(name))
			paddingStr := ""
			if padding > 0 {
				paddingStr = lipgloss.NewStyle().Background(lipgloss.Color(inst.BgColor)).Render(strings.Repeat(" ", padding))
			}
			gradientText := applyGradientWithBgBold(name, inst.Color, inst.BgColor)
			return fmt.Sprintf(" %s %s %s%s", listSelectedStyle.Render("▸"), status, gradientText, paddingStr)
		}
		rowStyle := lipgloss.NewStyle().Background(lipgloss.Color(inst.BgColor)).Bold(true)
		if inst.Color == "auto" || inst.Color == "" {
			rowStyle = rowStyle.Foreground(lipgloss.Color(getContrastColor(inst.BgColor)))
		} else {
			rowStyle = rowStyle.Foreground(lipgloss.Color(inst.Color))
		}
		textPart := name
		padding := listWidth - 7 - len([]rune(name))
		if padding > 0 {
			textPart += strings.Repeat(" ", padding)
		}
		return fmt.Sprintf(" %s %s %s", listSelectedStyle.Render("▸"), status, rowStyle.Render(textPart))
	} else if inst.Color != "" || inst.BgColor != "" {
		return fmt.Sprintf(" %s %s %s", listSelectedStyle.Render("▸"), status, lipgloss.NewStyle().Bold(true).Render(styledName))
	}
	return fmt.Sprintf(" %s %s %s", listSelectedStyle.Render("▸"), status, lipgloss.NewStyle().Bold(true).Render(name))
}

// renderUnselectedRow renders an unselected session row
func (m Model) renderUnselectedRow(inst *session.Instance, name, styledName, status string, listWidth int) string {
	if inst.FullRowColor && inst.BgColor != "" {
		if _, isGradient := gradients[inst.Color]; isGradient {
			padding := listWidth - 7 - len([]rune(name))
			paddingStr := ""
			if padding > 0 {
				paddingStr = lipgloss.NewStyle().Background(lipgloss.Color(inst.BgColor)).Render(strings.Repeat(" ", padding))
			}
			gradientText := applyGradientWithBg(name, inst.Color, inst.BgColor)
			return fmt.Sprintf("   %s %s%s", status, gradientText, paddingStr)
		}
		rowStyle := lipgloss.NewStyle().Background(lipgloss.Color(inst.BgColor))
		if inst.Color == "auto" || inst.Color == "" {
			rowStyle = rowStyle.Foreground(lipgloss.Color(getContrastColor(inst.BgColor)))
		} else {
			rowStyle = rowStyle.Foreground(lipgloss.Color(inst.Color))
		}
		textPart := name
		padding := listWidth - 7 - len([]rune(name))
		if padding > 0 {
			textPart += strings.Repeat(" ", padding)
		}
		return fmt.Sprintf("   %s %s", status, rowStyle.Render(textPart))
	}
	return fmt.Sprintf("   %s %s", status, styledName)
}

// getLastLine returns the last line of output for a session
func (m Model) getLastLine(inst *session.Instance) string {
	lastLine := m.lastLines[inst.ID]
	if lastLine == "" {
		if inst.Status == session.StatusRunning {
			return "loading..."
		}
		return "stopped"
	}
	// Truncate to prevent line wrap
	cleanLine := strings.TrimSpace(stripANSI(lastLine))
	maxLen := ListPaneWidth - 14 // Account for tree prefix + "└─ "
	if maxLen < 10 {
		maxLen = 10
	}
	return truncateRunes(cleanLine, maxLen)
}

// buildSessionListPane builds the left pane containing the session list
func (m Model) buildSessionListPane(listWidth, contentHeight int) string {
	var leftPane strings.Builder
	leftPane.WriteString("\n")
	leftPane.WriteString(titleStyle.Render(" Sessions "))
	leftPane.WriteString("\n\n")

	if len(m.instances) == 0 && len(m.groups) == 0 {
		leftPane.WriteString(" No sessions\n")
		leftPane.WriteString(dimStyle.Render(" Press 'n' to create"))
		return leftPane.String()
	}

	// If there are groups, use grouped view
	if len(m.groups) > 0 {
		return m.buildGroupedSessionListPane(listWidth, contentHeight)
	}

	// Otherwise, use flat view (original behavior)
	// Calculate visible range
	linesPerSession := 2
	if !m.compactList {
		linesPerSession = 3
	}
	maxVisible := contentHeight / linesPerSession
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(m.instances) {
		endIdx = len(m.instances)
	}

	// Show scroll indicator at top
	if startIdx > 0 {
		leftPane.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more\n", startIdx)))
	}

	for i := startIdx; i < endIdx; i++ {
		leftPane.WriteString(m.renderSessionRow(m.instances[i], i, listWidth))
	}

	// Show scroll indicator at bottom
	remaining := len(m.instances) - endIdx
	if remaining > 0 {
		leftPane.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more\n", remaining)))
	}

	return leftPane.String()
}

// buildGroupedSessionListPane builds the session list with groups
func (m *Model) buildGroupedSessionListPane(listWidth, contentHeight int) string {
	var leftPane strings.Builder
	leftPane.WriteString("\n")
	leftPane.WriteString(titleStyle.Render(" Sessions "))
	leftPane.WriteString("\n\n")

	// Build visible items
	m.buildVisibleItems()

	if len(m.visibleItems) == 0 {
		leftPane.WriteString(" No sessions\n")
		leftPane.WriteString(dimStyle.Render(" Press 'n' to create"))
		return leftPane.String()
	}

	// Calculate visible range
	linesPerItem := 2
	if !m.compactList {
		linesPerItem = 3
	}
	maxVisible := contentHeight / linesPerItem
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(m.visibleItems) {
		endIdx = len(m.visibleItems)
	}

	// Show scroll indicator at top
	if startIdx > 0 {
		leftPane.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more\n", startIdx)))
	}

	for i := startIdx; i < endIdx; i++ {
		item := m.visibleItems[i]
		if item.isGroup {
			leftPane.WriteString(m.renderGroupRow(item.group, i, listWidth))
		} else {
			// Check if this is the last session in its group
			isLast := m.isLastInGroup(i)
			leftPane.WriteString(m.renderGroupedSessionRow(item.instance, i, listWidth, isLast))
		}
	}

	// Show scroll indicator at bottom
	remaining := len(m.visibleItems) - endIdx
	if remaining > 0 {
		leftPane.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more\n", remaining)))
	}

	return leftPane.String()
}

// renderGroupRow renders a group header row
func (m Model) renderGroupRow(group *session.Group, index int, listWidth int) string {
	var row strings.Builder

	// Count sessions in this group
	sessionCount := len(m.getSessionsInGroup(group.ID))

	// Collapse indicator
	collapseIcon := "▼"
	if group.Collapsed {
		collapseIcon = "▶"
	}

	// Group style - use custom color if set, otherwise default purple
	groupColor := "#7D56F4"
	if group.Color != "" && group.Color != "auto" {
		groupColor = group.Color
	}
	groupStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(groupColor)).Bold(true)

	// Apply background color if set
	if group.BgColor != "" {
		groupStyle = groupStyle.Background(lipgloss.Color(group.BgColor))
		// Auto-contrast if no custom foreground or auto
		if group.Color == "" || group.Color == "auto" {
			groupStyle = groupStyle.Foreground(lipgloss.Color(getContrastColor(group.BgColor)))
		}
	}

	name := group.Name
	maxNameLen := listWidth - 12
	if len(name) > maxNameLen {
		name = name[:maxNameLen-1] + "…"
	}

	selected := index == m.cursor
	// Style both name and count together
	nameAndCount := fmt.Sprintf("%s [%d]", name, sessionCount)
	styledContent := groupStyle.Render(nameAndCount)

	// Full row background - only the name and count, not icons
	if group.FullRowColor && group.BgColor != "" {
		// Calculate remaining width for the colored part (after prefix + icons)
		prefixLen := 7 // "   📁▼ " or " ▸ 📁▼ "
		contentWidth := listWidth - prefixLen
		fullRowStyle := lipgloss.NewStyle().Background(lipgloss.Color(group.BgColor)).Width(contentWidth)
		if selected {
			row.WriteString(fmt.Sprintf(" %s 📁%s ", listSelectedStyle.Render("▸"), collapseIcon))
			row.WriteString(fullRowStyle.Render(styledContent))
			row.WriteString("\n")
		} else {
			row.WriteString(fmt.Sprintf("   📁%s ", collapseIcon))
			row.WriteString(fullRowStyle.Render(styledContent))
			row.WriteString("\n")
		}
	} else if selected {
		row.WriteString(fmt.Sprintf(" %s 📁%s %s\n",
			listSelectedStyle.Render("▸"),
			collapseIcon,
			styledContent))
	} else {
		row.WriteString(fmt.Sprintf("   📁%s %s\n",
			collapseIcon,
			styledContent))
	}

	if !m.compactList {
		// Add vertical line under group header if group has sessions and is expanded
		if !group.Collapsed && sessionCount > 0 {
			row.WriteString(dimStyle.Render("   │"))
		}
		row.WriteString("\n")
	}

	return row.String()
}

// renderGroupedSessionRow renders a session row with indent for grouped view
func (m Model) renderGroupedSessionRow(inst *session.Instance, index int, listWidth int, isLast bool) string {
	var row strings.Builder

	// Tree connectors for grouped sessions
	var prefix, lastLinePrefix string
	if inst.GroupID != "" {
		if isLast {
			prefix = "  └──"
			lastLinePrefix = "    " // 4 spaces to align └─ under ●
		} else {
			prefix = "  ├──"
			lastLinePrefix = "  │ " // │ aligns with ├, space before └─
		}
	} else {
		prefix = " "
		lastLinePrefix = ""
	}

	// Status indicator based on activity
	var status string
	if inst.Status == session.StatusRunning {
		if m.isActive[inst.ID] {
			status = activeStyle.Render("●") // Orange - active
		} else {
			status = idleStyle.Render("●") // Grey - idle/waiting
		}
	} else {
		status = stoppedStyle.Render("○") // Red outline - stopped
	}

	// Truncate name to fit (accounting for prefix)
	name := inst.Name
	maxNameLen := listWidth - 10
	if maxNameLen < 8 {
		maxNameLen = 8
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen-1] + "…"
	}

	// Apply session colors
	styledName := m.getStyledName(inst, name)
	selected := index == m.cursor

	// Render the row
	treeStyle := dimStyle
	if selected {
		row.WriteString(fmt.Sprintf(" %s%s %s", listSelectedStyle.Render("▸"), treeStyle.Render(prefix[1:]), status))
		if inst.FullRowColor && inst.BgColor != "" {
			row.WriteString(" " + m.renderSelectedRowContent(inst, name, listWidth-10))
		} else if inst.Color != "" || inst.BgColor != "" {
			row.WriteString(" " + lipgloss.NewStyle().Bold(true).Render(styledName))
		} else {
			row.WriteString(" " + lipgloss.NewStyle().Bold(true).Render(name))
		}
	} else {
		row.WriteString(fmt.Sprintf(" %s %s %s", treeStyle.Render(prefix), status, styledName))
	}
	row.WriteString("\n")

	// Show last output line with tree connector (└─ aligns under ● status icon)
	if !m.hideStatusLines {
		lastLine := m.getLastLine(inst)
		row.WriteString(fmt.Sprintf(" %s  └─ %s", treeStyle.Render(lastLinePrefix), lastLine))
		row.WriteString("\n")
	}

	// Add empty row spacing when not in compact mode
	if !m.compactList {
		// Add vertical line in empty row for non-last grouped sessions
		if inst.GroupID != "" && !isLast {
			row.WriteString(treeStyle.Render("   │"))
		}
		row.WriteString("\n")
	}

	return row.String()
}

// renderSelectedRowContent renders the content part of a selected row
func (m Model) renderSelectedRowContent(inst *session.Instance, name string, maxWidth int) string {
	if _, isGradient := gradients[inst.Color]; isGradient {
		padding := maxWidth - len([]rune(name))
		paddingStr := ""
		if padding > 0 {
			paddingStr = lipgloss.NewStyle().Background(lipgloss.Color(inst.BgColor)).Render(strings.Repeat(" ", padding))
		}
		gradientText := applyGradientWithBgBold(name, inst.Color, inst.BgColor)
		return gradientText + paddingStr
	}

	rowStyle := lipgloss.NewStyle().Background(lipgloss.Color(inst.BgColor)).Bold(true)
	if inst.Color == "auto" || inst.Color == "" {
		rowStyle = rowStyle.Foreground(lipgloss.Color(getContrastColor(inst.BgColor)))
	} else {
		rowStyle = rowStyle.Foreground(lipgloss.Color(inst.Color))
	}

	textPart := name
	padding := maxWidth - len([]rune(name))
	if padding > 0 {
		textPart += strings.Repeat(" ", padding)
	}
	return rowStyle.Render(textPart)
}

// buildPreviewPane builds the right pane containing the preview
func (m Model) buildPreviewPane(contentHeight int) string {
	var rightPane strings.Builder
	rightPane.WriteString("\n")
	rightPane.WriteString(titleStyle.Render(" Preview "))
	rightPane.WriteString("\n\n")

	// Get selected instance (handles both grouped and ungrouped modes)
	var inst *session.Instance
	if len(m.groups) > 0 {
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if !item.isGroup {
				inst = item.instance
			} else {
				// Group selected - show group info
				rightPane.WriteString(dimStyle.Render(fmt.Sprintf("  Group: %s", item.group.Name)))
				rightPane.WriteString("\n")
				sessionCount := len(m.getSessionsInGroup(item.group.ID))
				rightPane.WriteString(dimStyle.Render(fmt.Sprintf("  Sessions: %d", sessionCount)))
				rightPane.WriteString("\n\n")
				rightPane.WriteString(dimStyle.Render("  Press Enter to toggle collapse"))
				return rightPane.String()
			}
		}
	} else if len(m.instances) > 0 && m.cursor < len(m.instances) {
		inst = m.instances[m.cursor]
	}

	if inst == nil {
		return rightPane.String()
	}

	// Instance info
	rightPane.WriteString(dimStyle.Render(fmt.Sprintf("  Path: %s", inst.Path)))
	rightPane.WriteString("\n")

	// Show agent type
	agentName := "Claude Code"
	switch inst.Agent {
	case session.AgentGemini:
		agentName = "Gemini"
	case session.AgentAider:
		agentName = "Aider"
	case session.AgentCodex:
		agentName = "Codex CLI"
	case session.AgentAmazonQ:
		agentName = "Amazon Q"
	case session.AgentOpenCode:
		agentName = "OpenCode"
	case session.AgentCustom:
		agentName = "Custom"
	}
	rightPane.WriteString(dimStyle.Render(fmt.Sprintf("  Agent: %s", agentName)))
	rightPane.WriteString("\n")

	if inst.Agent == session.AgentCustom && inst.CustomCommand != "" {
		rightPane.WriteString(dimStyle.Render(fmt.Sprintf("  Command: %s", inst.CustomCommand)))
		rightPane.WriteString("\n")
	}

	if inst.ResumeSessionID != "" {
		rightPane.WriteString(dimStyle.Render(fmt.Sprintf("  Resume: %s", inst.ResumeSessionID[:8])))
		rightPane.WriteString("\n")
	}
	rightPane.WriteString("\n")

	// Preview content
	if m.preview == "" {
		rightPane.WriteString(dimStyle.Render("  (no output yet)"))
		return rightPane.String()
	}

	lines := strings.Split(m.preview, "\n")
	maxLines := contentHeight - PreviewHeaderHeight
	if maxLines < MinPreviewLines {
		maxLines = MinPreviewLines
	}
	startIdx := len(lines) - maxLines
	if startIdx < 0 {
		startIdx = 0
	}
	if startIdx > 0 {
		rightPane.WriteString(dimStyle.Render("   ..."))
		rightPane.WriteString("\n")
	}
	for i := startIdx; i < len(lines); i++ {
		rightPane.WriteString("  " + lines[i] + "\x1b[0m\n")
	}

	return rightPane.String()
}

// buildStatusBar builds the status bar at the bottom
func (m Model) buildStatusBar() string {
	// Styles for status bar
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444"))

	onStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)

	offStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))

	sep := separatorStyle.Render(" │ ")

	// Priority groups (high to low)
	// P1: Essential - always shown
	p1 := []string{
		keyStyle.Render("n") + descStyle.Render(" new"),
		keyStyle.Render("enter") + descStyle.Render(" attach"),
		keyStyle.Render("?") + descStyle.Render(" help"),
		keyStyle.Render("q") + descStyle.Render(" quit"),
	}

	// P2: Common actions
	p2 := []string{
		keyStyle.Render("s") + descStyle.Render(" start"),
		keyStyle.Render("x") + descStyle.Render(" stop"),
		keyStyle.Render("d") + descStyle.Render(" delete"),
		keyStyle.Render("p") + descStyle.Render(" prompt"),
	}

	// P3: Less common
	p3 := []string{
		keyStyle.Render("r") + descStyle.Render(" resume"),
		keyStyle.Render("e") + descStyle.Render(" rename"),
		keyStyle.Render("c") + descStyle.Render(" color"),
	}

	// P4: Group management
	p4 := []string{
		keyStyle.Render("g") + descStyle.Render(" group"),
		keyStyle.Render("G") + descStyle.Render(" assign"),
	}

	// P5: Toggles
	compactStatus := offStyle.Render("OFF")
	if m.compactList {
		compactStatus = onStyle.Render("ON")
	}
	statusLinesStatus := onStyle.Render("ON")
	if m.hideStatusLines {
		statusLinesStatus = offStyle.Render("OFF")
	}
	autoYesStatus := offStyle.Render("OFF")
	if m.autoYes {
		autoYesStatus = onStyle.Render("ON")
	}
	p5 := []string{
		keyStyle.Render("l") + descStyle.Render(" compact ") + compactStatus,
		keyStyle.Render("t") + descStyle.Render(" status ") + statusLinesStatus,
		keyStyle.Render("y") + descStyle.Render(" autoyes ") + autoYesStatus,
	}

	// Calculate widths and determine what fits
	sepWidth := 3 // " │ "

	// Try adding priority groups until it doesn't fit
	var items []string
	items = append(items, p1...)

	testWidth := func(newItems []string) int {
		total := 0
		for i, item := range newItems {
			total += len(stripANSI(item))
			if i < len(newItems)-1 {
				total += sepWidth
			}
		}
		return total
	}

	availableWidth := m.width - 4 // margin

	// Add P2 if fits
	testItems := append(items, p2...)
	if testWidth(testItems) <= availableWidth {
		items = testItems
		// Add P3 if fits
		testItems = append(items, p3...)
		if testWidth(testItems) <= availableWidth {
			items = testItems
			// Add P4 if fits
			testItems = append(items, p4...)
			if testWidth(testItems) <= availableWidth {
				items = testItems
				// Add P5 if fits
				testItems = append(items, p5...)
				if testWidth(testItems) <= availableWidth {
					items = testItems
				}
			}
		}
	}

	statusText := strings.Join(items, sep)

	return "\n" + lipgloss.PlaceHorizontal(m.width, lipgloss.Center, statusText)
}

// formatTimeAgo formats a time as a relative string (e.g., "5 min ago")
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		return fmt.Sprintf("%d min ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	}
}

// newGroupView renders the new group dialog as an overlay
func (m Model) newGroupView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n")
	boxContent.WriteString("  Group Name:\n")
	boxContent.WriteString("  " + m.groupInput.View() + "\n")
	boxContent.WriteString("\n")
	boxContent.WriteString(helpStyle.Render("  enter: create  esc: cancel"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" New Group ", boxContent.String(), 50, "#7D56F4")
}

// renameGroupView renders the rename group dialog as an overlay
func (m Model) renameGroupView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n")

	m.buildVisibleItems()
	if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
		item := m.visibleItems[m.cursor]
		if item.isGroup {
			boxContent.WriteString(fmt.Sprintf("  Current: %s\n\n", item.group.Name))
		}
	}

	boxContent.WriteString("  New Name:\n")
	boxContent.WriteString("  " + m.groupInput.View() + "\n")
	boxContent.WriteString("\n")
	boxContent.WriteString(helpStyle.Render("  enter: confirm  esc: cancel"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Rename Group ", boxContent.String(), 50, "#7D56F4")
}

// selectGroupView renders the group selection dialog as an overlay
func (m *Model) selectGroupView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n")

	// Find current session (works in both grouped and ungrouped modes)
	var inst *session.Instance
	if len(m.groups) > 0 {
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if !item.isGroup {
				inst = item.instance
			}
		}
	} else if len(m.instances) > 0 && m.cursor < len(m.instances) {
		inst = m.instances[m.cursor]
	}

	if inst != nil {
		boxContent.WriteString(fmt.Sprintf("  Session: %s\n\n", inst.Name))
	}

	boxContent.WriteString("  Select Group:\n\n")

	// Ungrouped option
	if m.groupCursor == 0 {
		boxContent.WriteString("  ❯ (No Group)\n")
	} else {
		boxContent.WriteString("    (No Group)\n")
	}

	// Groups
	for i, group := range m.groups {
		if m.groupCursor == i+1 {
			boxContent.WriteString(fmt.Sprintf("  ❯ 📁 %s\n", group.Name))
		} else {
			boxContent.WriteString(fmt.Sprintf("    📁 %s\n", group.Name))
		}
	}

	boxContent.WriteString("\n")
	boxContent.WriteString(helpStyle.Render("  enter: select  esc: cancel"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Assign to Group ", boxContent.String(), 50, "#7D56F4")
}

// selectAgentView renders the agent type selection dialog as an overlay
func (m Model) selectAgentView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n")
	boxContent.WriteString("  Select Agent Type:\n\n")

	// Agent options with descriptions
	agents := []struct {
		agent session.AgentType
		icon  string
		name  string
		desc  string
	}{
		{session.AgentClaude, "🤖", "Claude Code", "Anthropic CLI (resume, auto-yes)"},
		{session.AgentGemini, "✨", "Gemini", "Google AI CLI"},
		{session.AgentAider, "🔧", "Aider", "AI pair programming (auto-yes)"},
		{session.AgentCodex, "🧠", "Codex CLI", "OpenAI coding agent (auto-yes)"},
		{session.AgentAmazonQ, "📦", "Amazon Q", "AWS AI assistant (auto-yes)"},
		{session.AgentOpenCode, "💻", "OpenCode", "Terminal AI assistant"},
		{session.AgentCustom, "⚙️", "Custom", "Custom command"},
	}

	for i, a := range agents {
		if m.agentCursor == i {
			boxContent.WriteString(fmt.Sprintf("  ❯ %s %s\n", a.icon, a.name))
			boxContent.WriteString(dimStyle.Render(fmt.Sprintf("       %s", a.desc)))
			boxContent.WriteString("\n")
		} else {
			boxContent.WriteString(fmt.Sprintf("    %s %s\n", a.icon, a.name))
		}
	}

	// Show error if any
	if m.err != nil {
		boxContent.WriteString("\n")
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
		boxContent.WriteString(errStyle.Render(fmt.Sprintf("  ⚠ %v", m.err)))
		boxContent.WriteString("\n")
	}

	boxContent.WriteString("\n")
	boxContent.WriteString(helpStyle.Render("  enter: select  esc: cancel"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" New Session ", boxContent.String(), 50, "#7D56F4")
}

// customCmdView renders the custom command input dialog as an overlay
func (m Model) customCmdView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n")
	boxContent.WriteString("  Enter the command to run:\n\n")
	boxContent.WriteString("  " + m.customCmdInput.View() + "\n")
	boxContent.WriteString("\n")
	boxContent.WriteString(dimStyle.Render("  Example: aider --model gpt-4"))

	// Show error if any
	if m.err != nil {
		boxContent.WriteString("\n\n")
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Bold(true)
		boxContent.WriteString(errStyle.Render(fmt.Sprintf("  ⚠ %v", m.err)))
	}

	boxContent.WriteString("\n\n")
	boxContent.WriteString(helpStyle.Render("  enter: confirm  esc: back"))
	boxContent.WriteString("\n")

	boxWidth := 60
	if m.width > 80 {
		boxWidth = m.width / 2
	}
	if boxWidth > 80 {
		boxWidth = 80
	}

	return m.renderOverlayDialog(" Custom Command ", boxContent.String(), boxWidth, "#7D56F4")
}

// errorView renders the error overlay dialog
func (m Model) errorView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n")

	errMsg := "Unknown error"
	if m.err != nil {
		errMsg = m.err.Error()
	}

	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	boxContent.WriteString(errStyle.Render(fmt.Sprintf("  %s", errMsg)))
	boxContent.WriteString("\n\n")
	boxContent.WriteString(helpStyle.Render("  Press any key to close"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Error ", boxContent.String(), 60, "#FF5555")
}
