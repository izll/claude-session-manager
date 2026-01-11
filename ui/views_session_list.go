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
	session.AgentCursor:   "🖱️",
	session.AgentCustom:   "⚙️",
	session.AgentTerminal: "🖥️",
}

// getAgentIcon returns the icon for an agent type
func getAgentIcon(agent session.AgentType) string {
	if icon, ok := agentIcons[agent]; ok {
		return icon
	}
	return "?"
}

// buildAgentIconsInline builds a string of agent icons for inline display
// maxWidth limits how many icons can be shown (each icon is ~2 chars wide)
func (m Model) buildAgentIconsInline(inst *session.Instance, maxWidth int) string {
	if maxWidth < 3 {
		return ""
	}

	// Collect all agent types (main + tabs)
	var agents []session.AgentType
	agents = append(agents, inst.Agent)

	for _, fw := range inst.FollowedWindows {
		if fw.Agent != session.AgentTerminal {
			agents = append(agents, fw.Agent)
		}
	}

	// Build icons string, respecting width limit
	// Each icon takes approximately 2-3 chars (emoji + space)
	var icons strings.Builder
	icons.WriteString(" ")
	usedWidth := 1

	for i, agent := range agents {
		icon := getAgentIcon(agent)
		iconWidth := 2 // emoji width approximation

		// Check if we have room for this icon (and maybe "..." indicator)
		if i < len(agents)-1 && usedWidth+iconWidth+3 > maxWidth {
			// Not enough room for remaining icons, add indicator
			icons.WriteString("…")
			break
		}
		if usedWidth+iconWidth > maxWidth {
			break
		}

		icons.WriteString(icon)
		usedWidth += iconWidth
	}

	return icons.String()
}

// matchesSearch checks if an instance matches the search query
func (m Model) matchesSearch(inst *session.Instance) bool {
	if !m.searchActive || m.searchQuery == "" {
		return true
	}
	query := m.searchQuery // already lowercase

	// Check session name
	if strings.Contains(strings.ToLower(inst.Name), query) {
		return true
	}
	// Check session notes
	if strings.Contains(strings.ToLower(inst.Notes), query) {
		return true
	}
	// Check followed window names and notes
	for _, fw := range inst.FollowedWindows {
		if strings.Contains(strings.ToLower(fw.Name), query) {
			return true
		}
		if strings.Contains(strings.ToLower(fw.Notes), query) {
			return true
		}
	}
	// Check live tmux window names (tabs)
	if inst.Status == session.StatusRunning {
		windows := inst.GetWindowList()
		for _, w := range windows {
			if strings.Contains(strings.ToLower(w.Name), query) {
				return true
			}
		}
	}
	return false
}

// getFilteredInstances returns instances filtered by current search query
func (m Model) getFilteredInstances() []*session.Instance {
	if !m.searchActive || m.searchQuery == "" {
		return m.instances
	}
	var filtered []*session.Instance
	for _, inst := range m.instances {
		if m.matchesSearch(inst) {
			filtered = append(filtered, inst)
		}
	}
	return filtered
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
		pinStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
		status += " " + pinStyle.Render("◆")
	}

	// Truncate name to fit
	name := inst.Name
	iconLen := 0
	if m.showAgentIcons {
		iconLen = 3 // space + emoji
	}
	maxNameLen := listWidth - 8 - iconLen // -2 extra for pin marker
	if maxNameLen < 10 {
		maxNameLen = 10
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen-2] + "…"
	}

	// Apply session colors
	styledName := m.getStyledName(inst, name)
	selected := index == m.cursor

	// Count non-terminal followed windows (agent tabs)
	agentTabCount := 0
	for _, fw := range inst.FollowedWindows {
		if fw.Agent != session.AgentTerminal {
			agentTabCount++
		}
	}

	// Append agent icon(s) if enabled
	displayName := name
	displayStyledName := styledName
	if m.showAgentIcons {
		if m.hideStatusLines && agentTabCount > 0 {
			// Status lines hidden with multiple agents: show all icons inline
			icons := m.buildAgentIconsInline(inst, listWidth-len(name)-8)
			displayName = name + icons
			displayStyledName = styledName + icons
		} else if agentTabCount == 0 {
			// Single agent or status lines visible: show single icon
			icon := " " + getAgentIcon(inst.Agent)
			displayName = name + icon
			displayStyledName = styledName + icon
		}
		// else: multiple agents with status lines visible - icons shown on each status line
	}

	// Render the row
	if selected {
		row.WriteString(m.renderSelectedRow(inst, displayName, displayStyledName, status, listWidth))
	} else {
		row.WriteString(m.renderUnselectedRow(inst, displayName, displayStyledName, status, listWidth))
	}
	row.WriteString("\n")

	// Show last output line(s) with activity-based coloring
	if !m.hideStatusLines {
		connectorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))

		// Filter out terminal windows for display
		var displayWindows []session.FollowedWindow
		for _, fw := range inst.FollowedWindows {
			if fw.Agent != session.AgentTerminal {
				displayWindows = append(displayWindows, fw)
			}
		}

		// Determine tree connector based on whether there are more lines
		hasMoreLines := len(displayWindows) > 0
		mainConnector := "└─"
		if hasMoreLines {
			mainConnector = "├─"
		}

		// Get activity-based color for main window (0)
		mainActivity := session.ActivityIdle
		if winAct, ok := m.windowActivityState[inst.ID]; ok {
			if act, ok := winAct[0]; ok {
				mainActivity = act
			}
		}
		mainTextStyle := m.getActivityTextStyle(mainActivity, selected)

		// Main agent status (window 0)
		lastLine := m.getLastLine(inst)
		mainIcon := ""
		if m.showAgentIcons && len(displayWindows) > 0 {
			agent := inst.Agent
			if agent == "" {
				agent = session.AgentClaude
			}
			mainIcon = " " + getAgentIcon(agent)
		}
		row.WriteString(connectorStyle.Render("     "+mainConnector+" ") + mainTextStyle.Render(lastLine) + mainIcon)
		row.WriteString("\n")

		// Additional followed windows (excluding terminals)
		for i, fw := range displayWindows {
			fwLine := inst.GetLastLineForWindow(fw.Index, fw.Agent)
			fwLine = m.truncateStatusLine(fwLine)

			// Get activity-based color for this window
			fwActivity := session.ActivityIdle
			if winAct, ok := m.windowActivityState[inst.ID]; ok {
				if act, ok := winAct[fw.Index]; ok {
					fwActivity = act
				}
			}
			fwTextStyle := m.getActivityTextStyle(fwActivity, selected)

			// Last item gets └─, others get ├─
			connector := "├─"
			if i == len(displayWindows)-1 {
				connector = "└─"
			}
			fwIcon := ""
			if m.showAgentIcons {
				fwIcon = " " + getAgentIcon(fw.Agent)
			}
			row.WriteString(connectorStyle.Render("     "+connector+" ") + fwTextStyle.Render(fwLine) + fwIcon)
			row.WriteString("\n")
		}
	}

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

// truncateStatusLine truncates a status line to fit the list pane
func (m Model) truncateStatusLine(line string) string {
	cleanLine := strings.TrimSpace(stripANSI(line))
	if idx := strings.IndexAny(cleanLine, "\n\r"); idx >= 0 {
		cleanLine = strings.TrimSpace(cleanLine[:idx])
	}
	maxLen := ListPaneWidth - 14
	if maxLen < 10 {
		maxLen = 10
	}
	return truncateRunes(cleanLine, maxLen)
}

// getActivityTextStyle returns text style based on activity state
func (m Model) getActivityTextStyle(activity session.SessionActivity, selected bool) lipgloss.Style {
	switch activity {
	case session.ActivityBusy:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorOrange))
	case session.ActivityWaiting:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorCyan))
	default:
		// Idle - light gray
		return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))
	}
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

	// Get filtered instances
	instances := m.getFilteredInstances()

	if len(instances) == 0 && len(m.groups) == 0 && !m.hasFavorites() {
		if m.searchActive {
			leftPane.WriteString(" No matches\n")
			leftPane.WriteString(dimStyle.Render(" Press ESC to clear"))
		} else {
			leftPane.WriteString(" No sessions\n")
			leftPane.WriteString(dimStyle.Render(" Press 'n' to create"))
		}
		return leftPane.String()
	}

	// If there are groups or favorites, use grouped view
	if len(m.groups) > 0 || m.hasFavorites() {
		return m.buildGroupedSessionListPane(listWidth, contentHeight)
	}

	// Otherwise, use flat view (original behavior)
	// Calculate actual line count per session (dynamic based on tabs)
	getSessionHeight := func(inst *session.Instance) int {
		lines := 1 // Session name row
		if !m.hideStatusLines {
			lines++ // Main status line
			// Count additional status lines for followed windows (non-terminal)
			for _, fw := range inst.FollowedWindows {
				if fw.Agent != session.AgentTerminal {
					lines++
				}
			}
		}
		if !m.compactList {
			lines++ // Empty line between sessions
		}
		return lines
	}

	// Calculate which sessions fit in view
	// Adjust cursor if needed
	cursor := m.cursor
	if cursor >= len(instances) {
		cursor = len(instances) - 1
	}
	if cursor < 0 {
		cursor = 0
	}

	// Find start index by counting lines backwards from cursor
	// Calculate fixed header overhead dynamically
	headerHeight := 2 // Header with separator
	if m.activeProject != nil {
		headerHeight += 2 // Project name row with separator
	}
	headerHeight += 1 // Extra newline after header/project
	headerHeight += 3 // Reserve for scroll indicators + safety buffer
	availableHeight := contentHeight - headerHeight
	if availableHeight < 5 {
		availableHeight = 5
	}
	startIdx := 0
	endIdx := len(instances)

	// First, calculate total height up to and including cursor
	heightToCursor := 0
	for i := 0; i <= cursor && i < len(instances); i++ {
		heightToCursor += getSessionHeight(instances[i])
	}

	// If cursor position exceeds available height, scroll
	if heightToCursor > availableHeight {
		// Find start index that fits cursor in view
		usedHeight := 0
		for i := cursor; i >= 0; i-- {
			h := getSessionHeight(instances[i])
			if usedHeight+h > availableHeight {
				startIdx = i + 1
				break
			}
			usedHeight += h
		}
	}

	// Calculate end index based on available height
	usedHeight := 0
	for i := startIdx; i < len(instances); i++ {
		h := getSessionHeight(instances[i])
		if usedHeight+h > availableHeight {
			endIdx = i
			break
		}
		usedHeight += h
	}

	// Show scroll indicator at top
	if startIdx > 0 {
		leftPane.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more\n", startIdx)))
	}

	for i := startIdx; i < endIdx; i++ {
		// When filtering, use filtered index for cursor; otherwise use original index
		var cursorIdx int
		if m.searchActive {
			cursorIdx = i // Filtered index
		} else {
			cursorIdx = m.findInstanceIndex(instances[i].ID) // Original index
		}
		leftPane.WriteString(m.renderSessionRow(instances[i], cursorIdx, listWidth))
	}

	// Show scroll indicator at bottom
	remaining := len(instances) - endIdx
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
		if m.searchActive {
			leftPane.WriteString(" No matches\n")
			leftPane.WriteString(dimStyle.Render(" Press ESC to clear"))
		} else {
			leftPane.WriteString(" No sessions\n")
			leftPane.WriteString(dimStyle.Render(" Press 'n' to create"))
		}
		return leftPane.String()
	}

	// Calculate actual line count per item (dynamic based on tabs)
	getItemHeight := func(item visibleItem) int {
		if item.isGroup {
			lines := 1 // Group header row
			if !m.compactList {
				lines++ // Empty line / vertical connector
			}
			return lines
		}
		// Separator
		if item.instance == nil {
			return 1 // Single empty line
		}
		// Session
		inst := item.instance
		lines := 1 // Session name row
		if !m.hideStatusLines {
			lines++ // Main status line
			// Count additional status lines for followed windows (non-terminal)
			for _, fw := range inst.FollowedWindows {
				if fw.Agent != session.AgentTerminal {
					lines++
				}
			}
		}
		if !m.compactList {
			lines++ // Empty line between sessions
		}
		return lines
	}

	// Calculate which items fit in view
	// Calculate fixed header overhead dynamically
	headerHeight := 2 // Header with separator
	if m.activeProject != nil {
		headerHeight += 2 // Project name row with separator
	}
	headerHeight += 1 // Extra newline after header/project
	headerHeight += 3 // Reserve for scroll indicators + safety buffer
	availableHeight := contentHeight - headerHeight
	if availableHeight < 5 {
		availableHeight = 5
	}
	startIdx := 0
	endIdx := len(m.visibleItems)

	// Ensure cursor is within bounds
	cursor := m.cursor
	if cursor >= len(m.visibleItems) {
		cursor = len(m.visibleItems) - 1
	}
	if cursor < 0 {
		cursor = 0
	}

	// First, calculate total height up to and including cursor
	heightToCursor := 0
	for i := 0; i <= cursor && i < len(m.visibleItems); i++ {
		heightToCursor += getItemHeight(m.visibleItems[i])
	}

	// If cursor position exceeds available height, scroll
	if heightToCursor > availableHeight {
		// Find start index that fits cursor in view
		usedHeight := 0
		for i := cursor; i >= 0; i-- {
			h := getItemHeight(m.visibleItems[i])
			if usedHeight+h > availableHeight {
				startIdx = i + 1
				break
			}
			usedHeight += h
		}
	}

	// Calculate end index based on available height
	usedHeight := 0
	for i := startIdx; i < len(m.visibleItems); i++ {
		h := getItemHeight(m.visibleItems[i])
		if usedHeight+h > availableHeight {
			endIdx = i
			break
		}
		usedHeight += h
	}

	// Show scroll indicator at top
	if startIdx > 0 {
		leftPane.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more\n", startIdx)))
	}

	for i := startIdx; i < endIdx; i++ {
		item := m.visibleItems[i]
		if item.isGroup {
			leftPane.WriteString(m.renderGroupRow(item.group, i, listWidth))
		} else if item.instance == nil {
			// Separator - empty line
			leftPane.WriteString("\n")
		} else {
			// Check if this is the last session in its group
			isLast := m.isLastInGroup(i)
			// Favorites are in a group context even if their GroupID is empty
			inGroupContext := item.instance.Favorite
			leftPane.WriteString(m.renderGroupedSessionRow(item.instance, i, listWidth, isLast, inGroupContext))
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

	// Determine if this is the favorites group
	isFavorites := group.ID == FavoritesGroupID

	// Count sessions in this group
	var sessionCount int
	if isFavorites {
		// Count favorites directly
		for _, inst := range m.instances {
			if inst.Favorite {
				sessionCount++
			}
		}
	} else {
		sessionCount = len(m.getSessionsInGroup(group.ID))
	}

	// Collapse indicator
	collapseIcon := "▼"
	if group.Collapsed {
		collapseIcon = "▶"
	}

	// Group icon - star for favorites, folder for regular groups
	groupIcon := "📁"
	if isFavorites {
		groupIcon = "⭐"
	}

	// Group style - use custom color if set, otherwise default purple (gold for favorites)
	groupColor := ColorPurple
	if isFavorites {
		groupColor = ColorYellow // Gold color for favorites
	} else if group.Color != "" && group.Color != "auto" {
		groupColor = group.Color
	}
	groupStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(groupColor)).Bold(true)

	// Apply background color if set (not for favorites)
	if !isFavorites && group.BgColor != "" {
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

	// Full row background - only the name and count, not icons (not for favorites)
	if !isFavorites && group.FullRowColor && group.BgColor != "" {
		// Calculate remaining width for the colored part (after prefix + icons)
		prefixLen := 7 // "   📁▼ " or " ▸ 📁▼ "
		contentWidth := listWidth - prefixLen
		fullRowStyle := lipgloss.NewStyle().Background(lipgloss.Color(group.BgColor)).Width(contentWidth)
		if selected {
			row.WriteString(fmt.Sprintf(" %s %s%s ", listSelectedStyle.Render("▸"), groupIcon, collapseIcon))
			row.WriteString(fullRowStyle.Render(styledContent))
			row.WriteString("\n")
		} else {
			row.WriteString(fmt.Sprintf("   %s%s ", groupIcon, collapseIcon))
			row.WriteString(fullRowStyle.Render(styledContent))
			row.WriteString("\n")
		}
	} else if selected {
		row.WriteString(fmt.Sprintf(" %s %s%s %s\n",
			listSelectedStyle.Render("▸"),
			groupIcon,
			collapseIcon,
			styledContent))
	} else {
		row.WriteString(fmt.Sprintf("   %s%s %s\n",
			groupIcon,
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
// inGroupContext indicates if the session should be rendered with group tree connectors
// (even if inst.GroupID is empty, e.g., for favorites group)
func (m Model) renderGroupedSessionRow(inst *session.Instance, index int, listWidth int, isLast bool, inGroupContext bool) string {
	var row strings.Builder

	// Tree connectors for grouped sessions
	var prefix, lastLinePrefix string
	if inst.GroupID != "" || inGroupContext {
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
		pinStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
		status += " " + pinStyle.Render("◆")
	}

	// Truncate name to fit (accounting for prefix and icon)
	name := inst.Name
	iconLen := 0
	if m.showAgentIcons {
		iconLen = 3 // space + emoji
	}
	maxNameLen := listWidth - 12 - iconLen // -2 extra for pin marker
	if maxNameLen < 8 {
		maxNameLen = 8
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen-1] + "…"
	}

	// Apply session colors
	styledName := m.getStyledName(inst, name)
	selected := index == m.cursor

	// Count non-terminal followed windows (agent tabs)
	agentTabCount := 0
	for _, fw := range inst.FollowedWindows {
		if fw.Agent != session.AgentTerminal {
			agentTabCount++
		}
	}

	// Append agent icon(s) if enabled
	displayName := name
	displayStyledName := styledName
	if m.showAgentIcons {
		if m.hideStatusLines && agentTabCount > 0 {
			// Status lines hidden with multiple agents: show all icons inline
			icons := m.buildAgentIconsInline(inst, listWidth-len(name)-12)
			displayName = name + icons
			displayStyledName = styledName + icons
		} else if agentTabCount == 0 {
			// Single agent or status lines visible: show single icon
			icon := " " + getAgentIcon(inst.Agent)
			displayName = name + icon
			displayStyledName = styledName + icon
		}
		// else: multiple agents with status lines visible - icons shown on each status line
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

	// Show last output line(s) with tree connector and activity-based coloring
	if !m.hideStatusLines {
		connectorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))

		// Filter out terminal windows for display
		var displayWindows []session.FollowedWindow
		for _, fw := range inst.FollowedWindows {
			if fw.Agent != session.AgentTerminal {
				displayWindows = append(displayWindows, fw)
			}
		}

		// Determine tree connector based on whether there are more lines
		hasMoreLines := len(displayWindows) > 0
		mainConnector := "└─"
		if hasMoreLines {
			mainConnector = "├─"
		}

		// Get activity-based color for main window (0)
		mainActivity := session.ActivityIdle
		if winAct, ok := m.windowActivityState[inst.ID]; ok {
			if act, ok := winAct[0]; ok {
				mainActivity = act
			}
		}
		mainTextStyle := m.getActivityTextStyle(mainActivity, selected)

		// Main agent status (window 0)
		lastLine := m.getLastLine(inst)
		mainIcon := ""
		if m.showAgentIcons && len(displayWindows) > 0 {
			agent := inst.Agent
			if agent == "" {
				agent = session.AgentClaude
			}
			mainIcon = " " + getAgentIcon(agent)
		}
		row.WriteString(connectorStyle.Render(fmt.Sprintf(" %s  %s ", lastLinePrefix, mainConnector)) + mainTextStyle.Render(lastLine) + mainIcon)
		row.WriteString("\n")

		// Additional followed windows (excluding terminals)
		for i, fw := range displayWindows {
			fwLine := inst.GetLastLineForWindow(fw.Index, fw.Agent)
			fwLine = m.truncateStatusLine(fwLine)

			// Get activity-based color for this window
			fwActivity := session.ActivityIdle
			if winAct, ok := m.windowActivityState[inst.ID]; ok {
				if act, ok := winAct[fw.Index]; ok {
					fwActivity = act
				}
			}
			fwTextStyle := m.getActivityTextStyle(fwActivity, selected)

			// Last item gets └─, others get ├─
			connector := "├─"
			if i == len(displayWindows)-1 {
				connector = "└─"
			}
			fwIcon := ""
			if m.showAgentIcons {
				fwIcon = " " + getAgentIcon(fw.Agent)
			}
			row.WriteString(connectorStyle.Render(fmt.Sprintf(" %s  %s ", lastLinePrefix, connector)) + fwTextStyle.Render(fwLine) + fwIcon)
			row.WriteString("\n")
		}
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
