package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/izll/agent-session-manager/session"
)

// agentIcons maps agent types to their display icons
var agentIcons = map[session.AgentType]string{
	session.AgentClaude:   "🤖",
	session.AgentGemini:   "💎",
	session.AgentAider:    "🔧",
	session.AgentCodex:    "📦",
	session.AgentAmazonQ:  "🦜",
	session.AgentOpenCode: "💻",
	session.AgentCustom:   "⚙️",
}

// getAgentIcon returns the icon for an agent type
func getAgentIcon(agent session.AgentType) string {
	if icon, ok := agentIcons[agent]; ok {
		return icon
	}
	return "?"
}

// renderSessionRow renders a single session row with all color and style logic
func (m Model) renderSessionRow(inst *session.Instance, index int, listWidth int) string {
	var row strings.Builder

	// Status indicator based on activity state
	var status string
	if inst.Status == session.StatusRunning {
		switch m.activityState[inst.ID] {
		case session.ActivityBusy:
			status = activeStyle.Render("●") // Orange - busy/working
		case session.ActivityWaiting:
			status = waitingStyle.Render("●") // Yellow - waiting for input
		default:
			if m.isActive[inst.ID] {
				status = activeStyle.Render("●") // Orange - active
			} else {
				status = idleStyle.Render("●") // Grey - idle
			}
		}
	} else {
		status = stoppedStyle.Render("○") // Red outline - stopped
	}

	// Add marker for split view
	if m.markedSessionID == inst.ID {
		status += dimStyle.Render("◆")
	}

	// Truncate name to fit
	name := inst.Name
	iconLen := 0
	if m.showAgentIcons {
		iconLen = 3 // space + emoji
	}
	maxNameLen := listWidth - 6 - iconLen
	if maxNameLen < 10 {
		maxNameLen = 10
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen-2] + "…"
	}

	// Apply session colors
	styledName := m.getStyledName(inst, name)
	selected := index == m.cursor

	// Append agent icon if enabled
	displayName := name
	displayStyledName := styledName
	if m.showAgentIcons {
		icon := " " + getAgentIcon(inst.Agent)
		displayName = name + icon
		displayStyledName = styledName + icon
	}

	// Render the row
	if selected {
		row.WriteString(m.renderSelectedRow(inst, displayName, displayStyledName, status, listWidth))
	} else {
		row.WriteString(m.renderUnselectedRow(inst, displayName, displayStyledName, status, listWidth))
	}
	row.WriteString("\n")

	// Show last output line (text white if selected, gray otherwise)
	lastLine := m.getLastLine(inst)
	lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))
	if selected {
		textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWhite))
		row.WriteString(lineStyle.Render("     └─ ") + textStyle.Render(lastLine))
	} else {
		row.WriteString(lineStyle.Render(fmt.Sprintf("     └─ %s", lastLine)))
	}
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
	// If there's a line break, only show the first line
	if idx := strings.IndexAny(cleanLine, "\n\r"); idx >= 0 {
		cleanLine = strings.TrimSpace(cleanLine[:idx])
	}
	maxLen := ListPaneWidth - 14 // Account for tree prefix + "└─ "
	if maxLen < 10 {
		maxLen = 10
	}
	return truncateRunes(cleanLine, maxLen)
}

// buildSessionListPane builds the left pane containing the session list
func (m Model) buildSessionListPane(listWidth, contentHeight int) string {
	var leftPane strings.Builder

	// Build header with status counts
	leftPane.WriteString(m.buildSessionListHeader(listWidth))

	// Show project name if active project exists
	leftPane.WriteString(m.buildProjectNameRow(listWidth))
	leftPane.WriteString("\n")

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

	// Build header with status counts
	leftPane.WriteString(m.buildSessionListHeader(listWidth))

	// Show project name if active project exists
	leftPane.WriteString(m.buildProjectNameRow(listWidth))
	leftPane.WriteString("\n")

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
	groupColor := ColorPurple
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

	// Status indicator based on activity state
	var status string
	if inst.Status == session.StatusRunning {
		switch m.activityState[inst.ID] {
		case session.ActivityBusy:
			status = activeStyle.Render("●") // Orange - busy/working
		case session.ActivityWaiting:
			status = waitingStyle.Render("●") // Yellow - waiting for input
		default:
			if m.isActive[inst.ID] {
				status = activeStyle.Render("●") // Orange - active
			} else {
				status = idleStyle.Render("●") // Grey - idle
			}
		}
	} else {
		status = stoppedStyle.Render("○") // Red outline - stopped
	}

	// Add marker for split view
	if m.markedSessionID == inst.ID {
		status += dimStyle.Render("◆")
	}

	// Truncate name to fit (accounting for prefix and icon)
	name := inst.Name
	iconLen := 0
	if m.showAgentIcons {
		iconLen = 3 // space + emoji
	}
	maxNameLen := listWidth - 10 - iconLen
	if maxNameLen < 8 {
		maxNameLen = 8
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen-1] + "…"
	}

	// Apply session colors
	styledName := m.getStyledName(inst, name)
	selected := index == m.cursor

	// Append agent icon if enabled
	displayName := name
	displayStyledName := styledName
	if m.showAgentIcons {
		icon := " " + getAgentIcon(inst.Agent)
		displayName = name + icon
		displayStyledName = styledName + icon
	}

	// Render the row
	treeStyle := dimStyle
	if selected {
		row.WriteString(fmt.Sprintf(" %s%s %s", listSelectedStyle.Render("▸"), treeStyle.Render(prefix[1:]), status))
		if inst.FullRowColor && inst.BgColor != "" {
			row.WriteString(" " + m.renderSelectedRowContent(inst, displayName, listWidth-10-iconLen))
		} else if inst.Color != "" || inst.BgColor != "" {
			row.WriteString(" " + lipgloss.NewStyle().Bold(true).Render(displayStyledName))
		} else {
			row.WriteString(" " + lipgloss.NewStyle().Bold(true).Render(displayName))
		}
	} else {
		row.WriteString(fmt.Sprintf(" %s %s %s", treeStyle.Render(prefix), status, displayStyledName))
	}
	row.WriteString("\n")

	// Show last output line with tree connector (text white if selected, gray otherwise)
	if !m.hideStatusLines {
		lastLine := m.getLastLine(inst)
		lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))
		if selected {
			textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWhite))
			row.WriteString(lineStyle.Render(fmt.Sprintf(" %s  └─ ", lastLinePrefix)) + textStyle.Render(lastLine))
		} else {
			row.WriteString(lineStyle.Render(fmt.Sprintf(" %s  └─ %s", lastLinePrefix, lastLine)))
		}
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
