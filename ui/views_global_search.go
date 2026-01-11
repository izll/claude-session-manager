package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/izll/agent-session-manager/session"
)

// globalSearchLoadingView renders the loading state for global search
func (m Model) globalSearchLoadingView() string {
	var content strings.Builder

	content.WriteString("\n\n")

	// Loading dots animation based on time
	dots := []string{"", ".", "..", "..."}
	dotIndex := int(time.Now().UnixMilli()/300) % 4

	loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	content.WriteString(loadingStyle.Render("Loading history" + dots[dotIndex]))
	content.WriteString("\n\n")

	sourceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	content.WriteString(sourceStyle.Render("Claude, Gemini, Aider, OpenCode, Terminal"))
	content.WriteString("\n\n")

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	content.WriteString(footerStyle.Render("ESC Cancel"))

	boxWidth := 70

	return m.renderOverlayDialog("Global Search", content.String(), boxWidth, ColorPurple)
}

// globalSearchView renders the global search as a full-screen split view (like main window)
func (m Model) globalSearchView() string {
	// Use same dimensions as main list view
	listWidth := ListPaneWidth
	previewWidth := m.calculatePreviewWidth()
	contentHeight := m.height - 2 // Leave room for status bar

	if contentHeight < MinContentHeight {
		contentHeight = MinContentHeight
	}

	// Build left pane (search + results list)
	leftPane := m.buildSearchListPane(listWidth, contentHeight)

	// Build right pane (preview)
	rightPane := m.buildSearchPreviewPane(previewWidth, contentHeight)

	// Style the panes (same as main window)
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

	// Build final view with status bar
	var b strings.Builder
	b.WriteString(content)
	b.WriteString("\n")
	// Center the status bar horizontally
	statusBar := m.buildSearchStatusBar()
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, statusBar))

	return b.String()
}

// buildSearchListPane builds the left pane with search input and results
func (m Model) buildSearchListPane(width, height int) string {
	var b strings.Builder

	// Title bar (like main window)
	b.WriteString(titleStyle.Render(" 🔍 Global Search "))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(" (ASMGR sessions only)"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(" " + m.globalSearchInput.View())
	b.WriteString("\n\n")

	// Results count or hint
	if len(m.globalSearchResults) == 0 {
		query := strings.TrimSpace(m.globalSearchInput.Value())
		if query == "" {
			b.WriteString(dimStyle.Render(" Type to search..."))
		} else {
			b.WriteString(dimStyle.Render(" No results found"))
		}
		b.WriteString("\n")
	} else {
		// Count line with agent icons
		countStr := fmt.Sprintf(" %d results", len(m.globalSearchResults))
		agentCounts := m.countResultsByAgent()
		if len(agentCounts) > 0 {
			countStr += " ("
			agentOrder := []session.AgentType{
				session.AgentClaude, session.AgentGemini, session.AgentAider,
				session.AgentCodex, session.AgentAmazonQ, session.AgentOpenCode,
				session.AgentCursor, session.AgentTerminal,
			}
			first := true
			for _, agent := range agentOrder {
				if count, ok := agentCounts[agent]; ok {
					if !first {
						countStr += " "
					}
					countStr += fmt.Sprintf("%s%d", getAgentIcon(agent), count)
					first = false
				}
			}
			countStr += ")"
		}
		b.WriteString(dimStyle.Render(countStr))
		b.WriteString("\n\n")

		// Calculate visible results
		// Header: title(1) + \n(1) + input(1) + \n(1) + count(1) + \n(1) = 6
		// Footer: scroll indicator \n(1) + info(1) = 2
		availableHeight := height - 8 // Title, input, count, spacing, scroll indicator
		maxResults := availableHeight / 2
		if maxResults < 3 {
			maxResults = 3
		}

		// Scroll window
		startIdx := 0
		if m.globalSearchCursor >= maxResults {
			startIdx = m.globalSearchCursor - maxResults + 1
		}
		endIdx := startIdx + maxResults
		if endIdx > len(m.globalSearchResults) {
			endIdx = len(m.globalSearchResults)
		}

		// Render results (similar to session list)
		for i := startIdx; i < endIdx; i++ {
			entry := m.globalSearchResults[i]
			isSelected := i == m.globalSearchCursor

			icon := getAgentIcon(entry.Agent)
			timeAgo := formatTimeAgo(entry.Timestamp)

			// First line: icon + time
			line1 := fmt.Sprintf(" %s %s", icon, timeAgo)
			line1 = truncateRunesSafe(line1, width-2)

			if isSelected {
				// Selected style (like main list)
				b.WriteString(listSelectedStyle.Render(line1))
			} else {
				b.WriteString(dimStyle.Render(line1))
			}
			b.WriteString("\n")

			// Second line: snippet with highlighted matches
			snippet := entry.Snippet
			// Remove newlines to keep it on one line
			snippet = strings.ReplaceAll(snippet, "\n", " ")
			snippet = strings.ReplaceAll(snippet, "\r", "")
			maxSnippet := width - 5
			snippet = truncateRunesSafe(snippet, maxSnippet)
			query := strings.TrimSpace(m.globalSearchInput.Value())
			if isSelected {
				b.WriteString("   ")
				b.WriteString(highlightMatch(snippet, query, metaStyle))
			} else {
				b.WriteString("   ")
				b.WriteString(highlightMatch(snippet, query, dimStyle))
			}
			b.WriteString("\n")
		}

		// Scroll indicator at bottom
		if len(m.globalSearchResults) > maxResults {
			b.WriteString("\n")
			scrollInfo := fmt.Sprintf(" %d-%d / %d", startIdx+1, endIdx, len(m.globalSearchResults))
			b.WriteString(dimStyle.Render(scrollInfo))
		}
	}

	return b.String()
}

// buildSearchPreviewPane builds the right pane with full content preview
func (m Model) buildSearchPreviewPane(width, height int) string {
	var b strings.Builder

	if len(m.globalSearchResults) == 0 || m.globalSearchCursor >= len(m.globalSearchResults) {
		// No selection
		b.WriteString(titleStyle.Render(" Preview "))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render(" Select a result to preview"))
		return b.String()
	}

	// Get selected entry
	entry := m.globalSearchResults[m.globalSearchCursor]

	// Header with agent info and timestamp
	icon := getAgentIcon(entry.Agent)
	headerText := fmt.Sprintf(" %s %s • %s ", icon, entry.Agent, formatTimeAgo(entry.Timestamp))
	b.WriteString(titleStyle.Render(headerText))
	b.WriteString("\n")

	// Path if available
	if entry.Path != "" {
		path := entry.Path
		maxPath := width - 4
		if len(path) > maxPath {
			path = "..." + path[len(path)-maxPath+3:]
		}
		b.WriteString(metaStyle.Render(" " + path))
		b.WriteString("\n")
	}

	// Show matched session info
	if m.globalSearchMatchedSession != nil {
		matchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen))
		sessionIcon := getAgentIcon(m.globalSearchMatchedSession.Agent)
		matchInfo := fmt.Sprintf(" → %s %s", sessionIcon, m.globalSearchMatchedSession.Name)
		if m.globalSearchMatchedTabIndex >= 0 && m.globalSearchMatchedTabIndex < len(m.globalSearchMatchedSession.FollowedWindows) {
			tab := m.globalSearchMatchedSession.FollowedWindows[m.globalSearchMatchedTabIndex]
			tabIcon := getAgentIcon(tab.Agent)
			tabName := tab.Name
			if tabName == "" {
				tabName = fmt.Sprintf("tab %d", m.globalSearchMatchedTabIndex+1)
			}
			matchInfo = fmt.Sprintf(" → %s %s → %s %s", sessionIcon, m.globalSearchMatchedSession.Name, tabIcon, tabName)
		}
		b.WriteString(matchStyle.Render(matchInfo))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Calculate available lines for conversation content
	// Header takes: title(1) + newline + path(1) + match(1) + empty(1) + scroll indicator(2)
	headerLines := 5 // Base: title + separator + empty before content + scroll indicator + buffer
	if entry.Path != "" {
		headerLines++
	}
	if m.globalSearchMatchedSession != nil {
		headerLines++
	}
	availableLines := height - headerLines - 2 // Extra buffer to prevent overflow

	if availableLines < 5 {
		availableLines = 5
	}

	// Build conversation lines
	var lines []string

	if m.globalSearchConvLoading {
		// Show loading animation
		dots := []string{"", ".", "..", "..."}
		dotIndex := int(time.Now().UnixMilli()/300) % 4
		lines = append(lines, dimStyle.Render(" Loading"+dots[dotIndex]))
	} else if len(m.globalSearchConversation) > 0 {
		// Format conversation with User/Assistant markers
		lines = m.formatConversationLines(m.globalSearchConversation, width-2)
	} else if entry.SessionFile != "" {
		// Session file exists but conversation not loaded - show raw content with highlighting
		contentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
		query := strings.TrimSpace(m.globalSearchInput.Value())
		wrapped := wrapText(entry.Content, width-2)
		for _, line := range strings.Split(wrapped, "\n") {
			lines = append(lines, " "+highlightMatch(line, query, contentStyle))
		}
	} else {
		// No session file (history.jsonl entry or non-Claude) - show raw content with highlighting
		contentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
		query := strings.TrimSpace(m.globalSearchInput.Value())

		// Show that this is raw content
		lines = append(lines, dimStyle.Render(" (No full conversation available)"))
		lines = append(lines, "")

		wrapped := wrapText(entry.Content, width-2)
		for _, line := range strings.Split(wrapped, "\n") {
			lines = append(lines, " "+highlightMatch(line, query, contentStyle))
		}
	}

	// Apply scroll offset
	totalLines := len(lines)
	startLine := m.globalSearchScroll
	if startLine >= totalLines {
		startLine = totalLines - 1
		if startLine < 0 {
			startLine = 0
		}
	}

	// Render visible lines
	renderedLines := 0
	for i := startLine; i < totalLines && renderedLines < availableLines; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
		renderedLines++
	}

	// Scroll indicator
	if totalLines > availableLines {
		b.WriteString("\n")
		scrollInfo := fmt.Sprintf(" [/] PgUp/Dn • %d/%d", startLine+1, totalLines)
		b.WriteString(dimStyle.Render(scrollInfo))
	}

	return b.String()
}

// formatConversationLines formats conversation messages like Claude Code output
func (m Model) formatConversationLines(messages []session.ConversationMessage, width int) []string {
	var lines []string

	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen)).Bold(true)
	assistantStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorCyan)).Bold(true)
	contentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))

	// Get current search query for highlighting
	query := strings.TrimSpace(m.globalSearchInput.Value())

	for _, msg := range messages {
		// Role header
		if msg.Role == "user" {
			lines = append(lines, userStyle.Render(" 👤 User"))
		} else {
			lines = append(lines, assistantStyle.Render(" 🤖 Assistant"))
		}

		// Message content - wrap and indent, with highlighting
		wrapped := wrapText(msg.Content, width-4)
		for _, line := range strings.Split(wrapped, "\n") {
			if line == "" {
				lines = append(lines, "")
			} else {
				// Apply highlighting to the content
				highlighted := highlightMatch(line, query, contentStyle)
				lines = append(lines, "    "+highlighted)
			}
		}

		// Empty line between messages
		lines = append(lines, "")
	}

	return lines
}

// buildSearchStatusBar builds the status bar for global search
func (m Model) buildSearchStatusBar() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorLightGray))

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444"))

	sep := separatorStyle.Render(" │ ")

	items := []string{
		keyStyle.Render("↑↓") + descStyle.Render(" nav"),
		keyStyle.Render("Enter") + descStyle.Render(" open"),
		keyStyle.Render("[/] Alt+↑↓ PgUp/Dn") + descStyle.Render(" scroll"),
		keyStyle.Render("^R") + descStyle.Render(" reload"),
		keyStyle.Render("ESC") + descStyle.Render(" close"),
	}

	return strings.Join(items, sep)
}

// countResultsByAgent returns a map of agent type to result count
func (m Model) countResultsByAgent() map[session.AgentType]int {
	counts := make(map[session.AgentType]int)
	for _, entry := range m.globalSearchResults {
		counts[entry.Agent]++
	}
	return counts
}

// NOTE: getAgentIcon is defined in views_session_list.go
// NOTE: formatTimeAgo is defined in views_status.go

// truncatePath truncates a path to maxLen characters, keeping the end
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}

// wrapText wraps text to fit within maxWidth characters while preserving paragraph structure
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}

	// Replace tabs with spaces
	text = strings.ReplaceAll(text, "\t", "  ")

	// Split by newlines to preserve paragraph structure
	paragraphs := strings.Split(text, "\n")
	var result strings.Builder

	for pi, para := range paragraphs {
		if pi > 0 {
			result.WriteString("\n")
		}

		// Skip empty lines but preserve them
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// Wrap this paragraph
		words := strings.Fields(para)
		lineLen := 0

		for i, word := range words {
			wordLen := len(word)
			if lineLen+wordLen+1 > maxWidth && lineLen > 0 {
				result.WriteString("\n")
				lineLen = 0
			} else if i > 0 && lineLen > 0 {
				result.WriteString(" ")
				lineLen++
			}
			result.WriteString(word)
			lineLen += wordLen
		}
	}

	return result.String()
}

// highlightMatch highlights the query matches in text with a bright style
func highlightMatch(text, query string, baseStyle lipgloss.Style) string {
	if query == "" {
		return baseStyle.Render(text)
	}

	// Case-insensitive search
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	// Highlight style - yellow/orange on the base
	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000000")).
		Background(lipgloss.Color("#FFAA00")).
		Bold(true)

	var result strings.Builder
	lastEnd := 0

	for {
		idx := strings.Index(lowerText[lastEnd:], lowerQuery)
		if idx == -1 {
			// No more matches - render remaining text
			if lastEnd < len(text) {
				result.WriteString(baseStyle.Render(text[lastEnd:]))
			}
			break
		}

		// Absolute position
		matchStart := lastEnd + idx
		matchEnd := matchStart + len(query)

		// Render text before match
		if matchStart > lastEnd {
			result.WriteString(baseStyle.Render(text[lastEnd:matchStart]))
		}

		// Render highlighted match (preserve original case)
		result.WriteString(highlightStyle.Render(text[matchStart:matchEnd]))

		lastEnd = matchEnd
	}

	return result.String()
}

// truncateRunesSafe truncates a string to maxRunes characters (UTF-8 safe)
func truncateRunesSafe(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return "..."
	}
	return string(runes[:maxRunes-3]) + "..."
}

// globalSearchConfirmJumpView renders the confirm jump dialog
func (m Model) globalSearchConfirmJumpView() string {
	var content strings.Builder

	content.WriteString("\n\n")

	// Show matched session info
	if m.globalSearchMatchedSession != nil {
		inst := m.globalSearchMatchedSession
		tabIndex := m.globalSearchMatchedTabIndex

		// Session name with icon
		icon := getAgentIcon(inst.Agent)
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorCyan)).Bold(true)
		content.WriteString(nameStyle.Render(fmt.Sprintf("%s %s", icon, inst.Name)))
		content.WriteString("\n")

		// Show tab info if matched to a tab
		if tabIndex >= 0 && tabIndex < len(inst.FollowedWindows) {
			tab := inst.FollowedWindows[tabIndex]
			tabIcon := getAgentIcon(tab.Agent)
			tabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorYellow))
			tabName := tab.Name
			if tabName == "" {
				tabName = fmt.Sprintf("tab %d", tabIndex+1)
			}
			content.WriteString(tabStyle.Render(fmt.Sprintf("  └─ %s %s", tabIcon, tabName)))
			content.WriteString("\n")
		}
		content.WriteString("\n")

		// Path
		if inst.Path != "" {
			path := inst.Path
			if len(path) > 45 {
				path = "..." + path[len(path)-42:]
			}
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
			content.WriteString(dimStyle.Render(path))
			content.WriteString("\n\n")
		}

		// Status
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen))
		if inst.Status != session.StatusRunning {
			statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
		}
		content.WriteString(statusStyle.Render(fmt.Sprintf("Status: %s", inst.Status)))
		content.WriteString("\n\n")
	}

	// Show search entry snippet
	if m.globalSearchSelectedEntry != nil {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
		snippet := m.globalSearchSelectedEntry.Snippet
		if len(snippet) > 50 {
			snippet = snippet[:50] + "..."
		}
		content.WriteString(dimStyle.Render("\"" + snippet + "\""))
		content.WriteString("\n\n")
	}

	// Divider
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("─────────────────────────────────"))
	content.WriteString("\n\n")

	// Instructions
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(ColorGreen)).
		Bold(true).
		Padding(0, 1)

	escStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#666666")).
		Bold(true).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))

	jumpText := " Jump to session"
	if m.globalSearchMatchedTabIndex >= 0 {
		jumpText = " Jump to tab"
	}
	content.WriteString(keyStyle.Render("Enter") + descStyle.Render(jumpText))
	content.WriteString("\n")
	content.WriteString(escStyle.Render("ESC") + descStyle.Render("   Back to search"))

	boxWidth := 45

	return m.renderOverlayDialog("Session Found", content.String(), boxWidth, ColorGreen)
}

// globalSearchActionView renders the action selection dialog for global search results
func (m Model) globalSearchActionView() string {
	var content strings.Builder

	content.WriteString("\n")

	// Show selected entry info
	if m.globalSearchSelectedEntry != nil {
		entry := m.globalSearchSelectedEntry
		icon := getAgentIcon(entry.Agent)
		timeAgo := formatTimeAgo(entry.Timestamp)

		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorCyan)).Bold(true)
		content.WriteString(headerStyle.Render(fmt.Sprintf("%s %s • %s", icon, entry.Agent, timeAgo)))
		content.WriteString("\n\n")

		// Show snippet
		snippet := entry.Snippet
		if len(snippet) > 60 {
			snippet = snippet[:60] + "..."
		}
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
		content.WriteString(dimStyle.Render(snippet))
		content.WriteString("\n\n")

		// Show path if available
		if entry.Path != "" {
			path := entry.Path
			if len(path) > 50 {
				path = "..." + path[len(path)-47:]
			}
			content.WriteString(dimStyle.Render(path))
			content.WriteString("\n\n")
		}
	}

	// Divider
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("───────────────────────────────"))
	content.WriteString("\n\n")

	// Options
	options := []string{
		"1  New session",
		"2  Add to group",
		"3  Add as tab to current session",
	}

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorLightGray)).
		Padding(0, 1)

	for i, opt := range options {
		if i == m.globalSearchActionCursor {
			content.WriteString(selectedStyle.Render("▸ " + opt))
		} else {
			content.WriteString(normalStyle.Render("  " + opt))
		}
		content.WriteString("\n")
	}

	content.WriteString("\n")

	// Footer
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	content.WriteString(footerStyle.Render("↑/↓ Select • Enter Confirm • ESC Back"))

	boxWidth := 50

	return m.renderOverlayDialog("Open Session", content.String(), boxWidth, ColorPurple)
}

// globalSearchNewNameView renders the new session name input dialog
func (m Model) globalSearchNewNameView() string {
	var content strings.Builder

	content.WriteString("\n")

	// Show selected entry info
	if m.globalSearchSelectedEntry != nil {
		entry := m.globalSearchSelectedEntry
		icon := getAgentIcon(entry.Agent)
		timeAgo := formatTimeAgo(entry.Timestamp)

		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorCyan)).Bold(true)
		content.WriteString(headerStyle.Render(fmt.Sprintf("%s %s • %s", icon, entry.Agent, timeAgo)))
		content.WriteString("\n\n")

		// Show snippet
		snippet := entry.Snippet
		if len(snippet) > 45 {
			snippet = snippet[:45] + "..."
		}
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
		content.WriteString(dimStyle.Render("\"" + snippet + "\""))
		content.WriteString("\n\n")
	}

	// Divider
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("─────────────────────────────────────────"))
	content.WriteString("\n\n")

	// Label
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))
	content.WriteString(labelStyle.Render("Session name:"))
	content.WriteString("\n")

	// Input field
	content.WriteString(m.nameInput.View())
	content.WriteString("\n\n")

	// Footer
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(ColorGreen)).
		Bold(true).
		Padding(0, 1)

	escStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#666666")).
		Bold(true).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))

	content.WriteString(keyStyle.Render("Enter") + descStyle.Render(" Create"))
	content.WriteString("  ")
	content.WriteString(escStyle.Render("ESC") + descStyle.Render(" Back"))

	boxWidth := 50

	return m.renderOverlayDialog("New Session", content.String(), boxWidth, ColorPurple)
}

// globalSearchSelectMatchView renders the match selection dialog
func (m Model) globalSearchSelectMatchView() string {
	var content strings.Builder

	content.WriteString("\n\n")

	// Show search entry snippet
	if m.globalSearchSelectedEntry != nil {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
		snippet := m.globalSearchSelectedEntry.Snippet
		if len(snippet) > 45 {
			snippet = snippet[:45] + "..."
		}
		content.WriteString(dimStyle.Render("\"" + snippet + "\""))
		content.WriteString("\n\n")
	}

	// Divider
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("─────────────────────────────────────────"))
	content.WriteString("\n\n")

	// Label
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))
	content.WriteString(labelStyle.Render(fmt.Sprintf("Found %d matches:", len(m.globalSearchMatches))))
	content.WriteString("\n\n")

	// List matches
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorLightGray)).
		Padding(0, 1)

	for i, match := range m.globalSearchMatches {
		icon := getAgentIcon(match.Session.Agent)
		if match.TabIndex >= 0 && match.TabIndex < len(match.Session.FollowedWindows) {
			icon = getAgentIcon(match.Session.FollowedWindows[match.TabIndex].Agent)
		}

		label := fmt.Sprintf("%s %s", icon, match.TabName)
		if len(label) > 40 {
			label = label[:40] + "..."
		}

		if i == m.globalSearchMatchCursor {
			content.WriteString(selectedStyle.Render("▸ " + label))
		} else {
			content.WriteString(normalStyle.Render("  " + label))
		}
		content.WriteString("\n")
	}

	content.WriteString("\n")

	// Footer
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(ColorGreen)).
		Bold(true).
		Padding(0, 1)

	escStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#666666")).
		Bold(true).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorLightGray))

	content.WriteString(keyStyle.Render("Enter") + descStyle.Render(" Jump"))
	content.WriteString("  ")
	content.WriteString(escStyle.Render("ESC") + descStyle.Render(" Back"))

	boxWidth := 50

	return m.renderOverlayDialog("Select Session", content.String(), boxWidth, ColorPurple)
}
