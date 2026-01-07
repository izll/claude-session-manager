package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/izll/agent-session-manager/session"
)

// buildPreviewPane builds the right pane containing the preview
func (m Model) buildPreviewPane(contentHeight int) string {
	var rightPane strings.Builder

	// Header with session name (or Preview) on left and version on right
	previewWidth := m.calculatePreviewWidth()

	// Get selected instance for header
	var headerInst *session.Instance
	if len(m.groups) > 0 {
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if !item.isGroup {
				headerInst = item.instance
			}
		}
	} else if len(m.instances) > 0 && m.cursor < len(m.instances) {
		headerInst = m.instances[m.cursor]
	}

	// Build tab bar (Preview / Diff) - separator between tabs, not before
	// Use selectedStyle (no padding) for precise control
	border := dimStyle.Render("│")
	activeTab := selectedStyle.Bold(true)
	var tabBar string
	if m.showDiff {
		tabBar = dimStyle.Render("  Preview  ") + border + activeTab.Render("  Diff  ")
	} else {
		tabBar = activeTab.Render("  Preview  ") + border + dimStyle.Render("  Diff  ")
	}

	var title string
	if headerInst != nil {
		title = tabBar + dimStyle.Render("│ ") + formatSessionNameLipgloss(headerInst.Name, headerInst.Color, headerInst.BgColor)
	} else {
		title = tabBar
	}

	// Add update indicator if available
	versionText := fmt.Sprintf("%s v%s", AppName, AppVersion)
	if m.updateAvailable != "" {
		updateIcon := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorYellow)).
			Render(" ↑")
		versionText = versionText + updateIcon
	}
	version := dimStyle.Render(versionText + " ")

	titleLen := lipgloss.Width(title)
	versionLen := lipgloss.Width(version)
	spacing := previewWidth - titleLen - versionLen
	if spacing < 1 {
		spacing = 1
	}
	rightPane.WriteString(title + strings.Repeat(" ", spacing) + version)
	rightPane.WriteString("\n")
	rightPane.WriteString(dimStyle.Render(strings.Repeat("─", previewWidth)))
	rightPane.WriteString("\n")

	// Get selected instance (handles both grouped and ungrouped modes)
	var inst *session.Instance
	if len(m.groups) > 0 {
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if !item.isGroup {
				inst = item.instance
			} else {
				// Group selected - show group info with session details
				rightPane.WriteString("  " + projectLabelStyle.Render("Group: ") + projectNameStyle.Render(item.group.Name))
				rightPane.WriteString("\n")

				sessions := m.getSessionsInGroup(item.group.ID)
				runningCount := 0
				for _, s := range sessions {
					if s.Status == session.StatusRunning {
						runningCount++
					}
				}

				rightPane.WriteString("  " + projectLabelStyle.Render("Sessions: ") + fmt.Sprintf("%d", len(sessions)))
				if runningCount > 0 {
					runningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen))
					rightPane.WriteString(runningStyle.Render(fmt.Sprintf(" (%d running)", runningCount)))
				}
				rightPane.WriteString("\n")
				rightPane.WriteString(dimStyle.Render(strings.Repeat("─", previewWidth)))
				rightPane.WriteString("\n\n")

				// List sessions in group with full info
				if len(sessions) > 0 {
					for i, s := range sessions {
						// Status indicator
						var statusIcon string
						if s.Status == session.StatusRunning {
							switch m.activityState[s.ID] {
							case session.ActivityBusy:
								statusIcon = activeStyle.Render("●")
							case session.ActivityWaiting:
								statusIcon = waitingStyle.Render("●")
							default:
								statusIcon = idleStyle.Render("●")
							}
						} else {
							statusIcon = stoppedStyle.Render("○")
						}

						// Session name with status and color
						rightPane.WriteString(fmt.Sprintf("  %s %s", statusIcon, formatSessionNameLipgloss(s.Name, s.Color, s.BgColor)))
						rightPane.WriteString("\n")

						// Path
						rightPane.WriteString("    " + projectLabelStyle.Render("Path: ") + projectNameStyle.Render(s.Path))
						rightPane.WriteString("\n")

						// Agent
						agentName := "Claude Code"
						switch s.Agent {
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
						if (s.Agent == session.AgentClaude || s.Agent == "") && s.AutoYes {
							yoloStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorOrange))
							rightPane.WriteString("    " + projectLabelStyle.Render("Agent: ") + projectNameStyle.Render(agentName) + yoloStyle.Render(" ! YOLO"))
						} else {
							rightPane.WriteString("    " + projectLabelStyle.Render("Agent: ") + projectNameStyle.Render(agentName))
						}
						rightPane.WriteString("\n")

						// Resume ID (if any)
						if s.ResumeSessionID != "" {
							rightPane.WriteString("    " + projectLabelStyle.Render("Resume: ") + projectNameStyle.Render(s.ResumeSessionID[:8]))
							rightPane.WriteString("\n")
						}

						// Notes (if any)
						if s.Notes != "" {
							notesPreview := s.Notes
							if idx := strings.Index(notesPreview, "\n"); idx != -1 {
								notesPreview = notesPreview[:idx] + "…"
							}
							maxLen := previewWidth - 14
							if len([]rune(notesPreview)) > maxLen {
								notesPreview = string([]rune(notesPreview)[:maxLen-1]) + "…"
							}
							notesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorYellow)).Italic(true)
							rightPane.WriteString("    " + projectLabelStyle.Render("Notes: ") + notesStyle.Render(notesPreview))
							rightPane.WriteString("\n")
						}

						// Separator between sessions (except last)
						if i < len(sessions)-1 {
							rightPane.WriteString("\n")
						}
					}
				}

				rightPane.WriteString("\n")
				rightPane.WriteString(dimStyle.Render("  ↵ toggle • →/← expand/collapse"))
				return rightPane.String()
			}
		}
	} else if len(m.instances) > 0 && m.cursor < len(m.instances) {
		inst = m.instances[m.cursor]
	}

	if inst == nil {
		return rightPane.String()
	}

	// Diff mode - simplified header with only Path, Notes, and View mode
	if m.showDiff {
		// Path
		rightPane.WriteString("  " + projectLabelStyle.Render("Path: ") + projectNameStyle.Render(inst.Path))
		rightPane.WriteString("\n")

		// Notes if any
		if inst.Notes != "" {
			notesPreview := inst.Notes
			if idx := strings.Index(notesPreview, "\n"); idx != -1 {
				notesPreview = notesPreview[:idx] + "…"
			}
			maxNotesLen := previewWidth - 12
			if len([]rune(notesPreview)) > maxNotesLen {
				notesPreview = truncateRunes(notesPreview, maxNotesLen)
			}
			notesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorYellow)).Italic(true)
			rightPane.WriteString("  " + projectLabelStyle.Render("Notes: ") + notesStyle.Render(notesPreview))
			rightPane.WriteString("\n")
		}

		// View mode with hint
		diffModeLabel := m.diffPane.GetModeLabel()
		rightPane.WriteString("  " + projectLabelStyle.Render("View: ") + projectNameStyle.Render(diffModeLabel) + dimStyle.Render(" (F to switch)"))
		rightPane.WriteString("\n")

		// Horizontal separator
		rightPane.WriteString(dimStyle.Render(strings.Repeat("─", previewWidth)))
		rightPane.WriteString("\n")

		headerLines := strings.Count(rightPane.String(), "\n") + 1
		return m.buildDiffContent(rightPane.String(), contentHeight, headerLines, previewWidth)
	}

	// Get window list for tab display
	windows := inst.GetWindowList()
	var activeWindow *session.WindowInfo
	for i := range windows {
		if windows[i].Active {
			activeWindow = &windows[i]
			break
		}
	}

	// Display tmux tabs if more than 1 window (at top, before session info)
	if len(windows) > 1 {
		tabBorderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDarkGray))
		tabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))
		activeTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWhite)).Bold(true)
		var tabs strings.Builder
		tabs.WriteString(tabBorderStyle.Render("│"))
		for i, w := range windows {
			tabName := w.Name
			// Strip " ! " prefix from old YOLO windows
			if strings.HasPrefix(tabName, " ! ") {
				tabName = strings.TrimPrefix(tabName, " ! ")
			}
			// Add status indicator
			tabIndicator := ""
			if w.Dead {
				// Tab process has exited - show stopped indicator
				tabIndicator = stoppedStyle.Render("○ ")
			} else if w.Followed {
				// Check if this is a terminal window
				isTerminal := false
				for _, fw := range inst.FollowedWindows {
					if fw.Index == w.Index && fw.Agent == session.AgentTerminal {
						isTerminal = true
						break
					}
				}
				// Only show activity indicator for non-terminal agents
				if !isTerminal {
					activity := inst.DetectActivityForWindow(w.Index)
					switch activity {
					case session.ActivityWaiting:
						tabIndicator = waitingStyle.Render("● ")
					case session.ActivityBusy:
						tabIndicator = activeStyle.Render("● ")
					default:
						tabIndicator = idleStyle.Render("● ")
					}
				}
			}
			if w.Active {
				tabs.WriteString(" " + tabIndicator + activeTabStyle.Render(tabName) + " ")
			} else {
				tabs.WriteString(" " + tabIndicator + tabStyle.Render(tabName) + " ")
			}
			if i < len(windows)-1 {
				tabs.WriteString(tabBorderStyle.Render("│"))
			}
		}
		tabs.WriteString(tabBorderStyle.Render("│"))
		rightPane.WriteString("  " + tabs.String())
		rightPane.WriteString("\n")
		rightPane.WriteString(dimStyle.Render(strings.Repeat("─", previewWidth)))
		rightPane.WriteString("\n")
	}

	// Determine agent info based on active tab
	agentName := "Claude Code"
	agentType := inst.Agent
	customCmd := inst.CustomCommand
	autoYes := inst.AutoYes
	resumeID := inst.ResumeSessionID
	notes := inst.Notes

	// If active window is a followed agent tab, show that agent's info
	if activeWindow != nil && activeWindow.Followed {
		for _, fw := range inst.FollowedWindows {
			if fw.Index == activeWindow.Index {
				agentType = fw.Agent
				customCmd = fw.CustomCommand
				autoYes = fw.AutoYes
				resumeID = fw.ResumeSessionID
				notes = fw.Notes // Tab-specific notes
				break
			}
		}
	}

	// Get agent name from type
	switch agentType {
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

	// Instance info with styled labels and values
	rightPane.WriteString("  " + projectLabelStyle.Render("Path: ") + projectNameStyle.Render(inst.Path))
	rightPane.WriteString("\n")

	// Show yolo mode for Claude on same line
	if (agentType == session.AgentClaude || agentType == "") && autoYes {
		yoloStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorOrange))
		rightPane.WriteString("  " + projectLabelStyle.Render("Agent: ") + projectNameStyle.Render(agentName) + yoloStyle.Render(" ! YOLO"))
	} else {
		rightPane.WriteString("  " + projectLabelStyle.Render("Agent: ") + projectNameStyle.Render(agentName))
	}
	rightPane.WriteString("\n")

	if agentType == session.AgentCustom && customCmd != "" {
		rightPane.WriteString("  " + projectLabelStyle.Render("Command: ") + projectNameStyle.Render(customCmd))
		rightPane.WriteString("\n")
	}

	if resumeID != "" {
		rightPane.WriteString("  " + projectLabelStyle.Render("Resume: ") + projectNameStyle.Render(resumeID[:8]))
		rightPane.WriteString("\n")
	}

	// Display notes if any (truncated to fit)
	if notes != "" {
		// Show first line of notes or truncate if too long
		notesPreview := notes
		if idx := strings.Index(notesPreview, "\n"); idx != -1 {
			notesPreview = notesPreview[:idx] + "…"
		}
		maxNotesLen := previewWidth - 12 // "  Notes: " prefix + some margin
		if len([]rune(notesPreview)) > maxNotesLen {
			notesPreview = truncateRunes(notesPreview, maxNotesLen)
		}
		notesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorYellow)).Italic(true)
		rightPane.WriteString("  " + projectLabelStyle.Render("Notes: ") + notesStyle.Render(notesPreview))
		rightPane.WriteString("\n")
	}

	// Horizontal separator
	rightPane.WriteString(dimStyle.Render(strings.Repeat("─", previewWidth)))
	rightPane.WriteString("\n")

	// Count actual header lines dynamically (+1 for bottom padding)
	headerLines := strings.Count(rightPane.String(), "\n") + 1

	// Preview content
	if m.preview == "" {
		rightPane.WriteString(dimStyle.Render("  (no output yet)"))
		return rightPane.String()
	}

	// Use scrollContent if scrolling, otherwise use preview
	content := m.preview
	if m.previewScroll > 0 && m.scrollContent != "" {
		content = m.scrollContent
	}
	lines := strings.Split(content, "\n")
	maxLines := contentHeight - headerLines
	if maxLines < MinPreviewLines {
		maxLines = MinPreviewLines
	}

	// When scrolling, always reserve 2 lines for indicators to prevent layout shift
	if m.previewScroll > 0 {
		maxLines -= 2
	}

	// Apply scroll offset
	endIdx := len(lines) - m.previewScroll
	if endIdx < maxLines {
		endIdx = maxLines
	}
	if endIdx > len(lines) {
		endIdx = len(lines)
	}
	startIdx := endIdx - maxLines
	if startIdx < 0 {
		startIdx = 0
	}

	// Show scroll indicator at top if scrolled and not at beginning
	if m.previewScroll > 0 && startIdx > 0 {
		rightPane.WriteString(dimStyle.Render("   ↑ more"))
		rightPane.WriteString("\n")
	} else if m.previewScroll > 0 {
		// Empty line to keep layout stable
		rightPane.WriteString("\n")
	}

	for i := startIdx; i < endIdx; i++ {
		line := lines[i]
		// Truncate to available width (previewWidth - 2 for left margin)
		maxWidth := previewWidth - 2
		if displayWidth(line) > maxWidth {
			line = truncateToWidth(line, maxWidth)
		}
		rightPane.WriteString("  " + line + "\x1b[0m\n")
	}

	// Show scroll indicator at bottom if scrolled
	if m.previewScroll > 0 {
		rightPane.WriteString(dimStyle.Render(fmt.Sprintf("   ↓ more (%d lines)", m.previewScroll)))
		rightPane.WriteString("\n")
	}

	// Truncate to exactly contentHeight lines to prevent layout shift
	result := rightPane.String()
	resultLines := strings.Split(result, "\n")
	if len(resultLines) > contentHeight {
		resultLines = resultLines[:contentHeight]
	}
	return strings.Join(resultLines, "\n")
}

// buildSplitPreviewPane builds split view with two preview panes
func (m Model) buildSplitPreviewPane(contentHeight int) string {
	var result strings.Builder
	previewWidth := m.calculatePreviewWidth()

	// Get selected instance
	selectedInst := m.getSelectedInstance()

	// Get marked instance
	var markedInst *session.Instance
	if m.markedSessionID != "" {
		for _, inst := range m.instances {
			if inst.ID == m.markedSessionID {
				markedInst = inst
				break
			}
		}
	}

	// Calculate heights for each pane
	halfHeight := (contentHeight - 1) / 2 // -1 for separator

	// Top pane: marked session (pinned)
	topFocused := m.splitFocus == 1
	topScroll := 0
	if topFocused {
		topScroll = m.previewScroll
	}
	if markedInst != nil {
		result.WriteString("\n") // Add spacing at top
		result.WriteString(m.buildMiniPreview(markedInst, halfHeight, previewWidth, "Pinned", topFocused, topScroll))
	} else {
		result.WriteString("\n")
		result.WriteString(dimStyle.Render("  Press 'm' to pin a session"))
		result.WriteString("\n")
	}

	// Separator
	result.WriteString(dimStyle.Render(strings.Repeat("─", previewWidth-2)))
	result.WriteString("\n")

	// Bottom pane: selected session
	bottomFocused := m.splitFocus == 0
	bottomScroll := 0
	if bottomFocused {
		bottomScroll = m.previewScroll
	}
	if selectedInst != nil && (markedInst == nil || selectedInst.ID != markedInst.ID) {
		result.WriteString(m.buildMiniPreview(selectedInst, halfHeight, previewWidth, "Selected", bottomFocused, bottomScroll))
	} else if selectedInst != nil {
		result.WriteString(dimStyle.Render("  (same as pinned)"))
	}

	return result.String()
}

// buildDiffContent builds the diff view content
func (m *Model) buildDiffContent(header string, contentHeight, headerLines, previewWidth int) string {
	var result strings.Builder
	result.WriteString(header)

	// Set diff pane size (full width, viewport handles content)
	diffHeight := contentHeight - headerLines
	if diffHeight < MinPreviewLines {
		diffHeight = MinPreviewLines
	}
	m.diffPane.SetSize(previewWidth, diffHeight)

	// Get diff content from diff pane - viewport handles everything
	result.WriteString(m.diffPane.View())

	return result.String()
}

// buildMiniPreview builds a compact preview for split view
func (m Model) buildMiniPreview(inst *session.Instance, height, width int, label string, focused bool, scrollOffset int) string {
	var preview strings.Builder

	if inst == nil {
		preview.WriteString(dimStyle.Render(fmt.Sprintf("  %s: (none)\n", label)))
		return preview.String()
	}

	// Focus indicator
	focusIndicator := " "
	if focused {
		focusIndicator = titleStyle.Render("▶")
	}

	// Header with session name using configured colors
	preview.WriteString(focusIndicator + " ")
	preview.WriteString(formatSessionNameLipgloss(inst.Name, inst.Color, inst.BgColor))
	// Add status indicator after name
	if inst.Status != session.StatusRunning {
		preview.WriteString(stoppedStyle.Render(" ○"))
	} else if m.isActive[inst.ID] {
		preview.WriteString(activeStyle.Render(" ●"))
	}
	preview.WriteString("\n")

	// Get preview content for this instance
	content := ""
	if inst.Status == session.StatusRunning {
		// Use scrollContent if scrolling, otherwise fetch normal preview
		if scrollOffset > 0 && m.scrollContent != "" {
			content = m.scrollContent
		} else {
			content, _ = inst.GetPreview(PreviewLineCount)
		}
	}

	maxLines := height - 2 // -2 for header and margin
	if maxLines < 2 {
		maxLines = 2
	}

	if content == "" {
		preview.WriteString(dimStyle.Render("  (no output)"))
		preview.WriteString("\n")
		// Fill remaining lines with empty space
		for i := 1; i < maxLines; i++ {
			preview.WriteString("\n")
		}
		return preview.String()
	}

	// Apply scroll offset (similar to buildPreviewPane)
	lines := strings.Split(content, "\n")
	endIdx := len(lines) - scrollOffset
	if endIdx < maxLines {
		endIdx = maxLines
	}
	if endIdx > len(lines) {
		endIdx = len(lines)
	}
	startIdx := endIdx - maxLines
	if startIdx < 0 {
		startIdx = 0
	}

	// Show scroll indicator at top if not at beginning
	if startIdx > 0 {
		preview.WriteString(dimStyle.Render("   ↑ more"))
		preview.WriteString("\n")
		maxLines-- // Account for indicator line
	}

	displayedLines := 0
	for i := startIdx; i < endIdx && displayedLines < maxLines; i++ {
		line := lines[i]
		// Truncate to available width (width - 2 for left margin)
		maxWidth := width - 2
		if displayWidth(line) > maxWidth {
			line = truncateToWidth(line, maxWidth)
		}
		preview.WriteString("  " + line + "\x1b[0m\n")
		displayedLines++
	}

	// Show scroll indicator at bottom if scrolled
	if scrollOffset > 0 {
		preview.WriteString(dimStyle.Render(fmt.Sprintf("   ↓ more (%d lines)", scrollOffset)))
		preview.WriteString("\n")
		displayedLines++
	}

	// Fill remaining lines with empty space
	for i := displayedLines; i < maxLines; i++ {
		preview.WriteString("\n")
	}

	return preview.String()
}
