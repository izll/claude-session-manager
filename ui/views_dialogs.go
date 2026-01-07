package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/izll/agent-session-manager/session"
)

// confirmDeleteView renders the delete confirmation dialog as an overlay
func (m Model) confirmDeleteView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	if m.deleteTarget != nil {
		boxContent.WriteString(fmt.Sprintf("  Delete session '%s'?\n\n", m.deleteTarget.Name))
	}
	boxContent.WriteString(helpStyle.Render("  y: yes  n: no"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Confirm Delete ", boxContent.String(), 40, "#FF5F87")
}

// confirmStopView renders the stop confirmation dialog as an overlay
func (m Model) confirmStopView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	if m.stopTarget != nil {
		boxContent.WriteString(fmt.Sprintf("  Stop session '%s'?\n\n", m.stopTarget.Name))
	}
	boxContent.WriteString(helpStyle.Render("  y: yes  n: no"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Confirm Stop ", boxContent.String(), 40, "#FFA500")
}

// confirmStartView renders the auto-start confirmation dialog as an overlay
func (m Model) confirmStartView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	if inst := m.getSelectedInstance(); inst != nil {
		if inst.Status == session.StatusRunning {
			boxContent.WriteString(fmt.Sprintf("  Start NEW session for '%s'?\n", inst.Name))
			boxContent.WriteString("  (will stop current and start fresh)\n\n")
		} else {
			boxContent.WriteString(fmt.Sprintf("  Start NEW session for '%s'?\n\n", inst.Name))
		}
	}
	boxContent.WriteString(helpStyle.Render("  y: yes  n: no"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Start New Session ", boxContent.String(), 50, "#87D7FF")
}

// selectStartModeView renders the start mode selection dialog
func (m Model) selectStartModeView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	if inst := m.getSelectedInstance(); inst != nil {
		boxContent.WriteString(fmt.Sprintf("  Start mode for '%s':\n\n", inst.Name))
		boxContent.WriteString("  1/r: Replace current session\n")
		boxContent.WriteString(helpStyle.Render("       (stop current, start fresh)"))
		boxContent.WriteString("\n\n")
		boxContent.WriteString("  2/n: Start parallel session\n")
		boxContent.WriteString(helpStyle.Render("       (new instance below current)"))
		boxContent.WriteString("\n\n")
	}
	boxContent.WriteString(helpStyle.Render("  esc: cancel"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Start Session ", boxContent.String(), 50, "#87D7FF")
}

// newInstanceView renders the new session creation dialog as an overlay
func (m Model) newInstanceView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")

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

// renameView renders the rename dialog as an overlay
func (m Model) renameView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")

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
func (m *Model) promptView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")

	if inst := m.getSelectedInstance(); inst != nil {
		boxContent.WriteString(fmt.Sprintf("  Session: %s\n\n", inst.Name))
	}

	// Dynamic box width
	boxWidth := 70
	if m.width > 100 {
		boxWidth = 80
	}
	if boxWidth > 90 {
		boxWidth = 90
	}

	// Set textarea width to box width minus padding
	m.promptInput.SetWidth(boxWidth - 6)

	boxContent.WriteString("  Message:\n")

	// Indent each line of textarea by 2 spaces
	textareaView := m.promptInput.View()
	lines := strings.Split(textareaView, "\n")
	for i, line := range lines {
		boxContent.WriteString("  " + line)
		if i < len(lines)-1 {
			boxContent.WriteString("\n")
		}
	}
	boxContent.WriteString("\n")

	// Show suggestion if available and input is empty
	if m.promptSuggestion != "" && m.promptInput.Value() == "" {
		suggestionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Italic(true)
		boxContent.WriteString(suggestionStyle.Render(fmt.Sprintf("  → %s", m.promptSuggestion)) + "\n")
	}

	boxContent.WriteString("\n")

	helpText := "  ctrl+s: send  esc: cancel"
	if m.promptSuggestion != "" {
		helpText = "  tab: accept  ctrl+s: send  esc: cancel"
	}
	boxContent.WriteString(helpStyle.Render(helpText))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Send Message ", boxContent.String(), boxWidth, "#7D56F4")
}

// newGroupView renders the new group dialog as an overlay
func (m Model) newGroupView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
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
	boxContent.WriteString("\n\n")

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
	boxContent.WriteString("\n\n")

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
	boxContent.WriteString("\n\n")
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
	boxContent.WriteString("\n\n")
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

// errorView renders the error/info overlay dialog
func (m Model) errorView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")

	errMsg := "Unknown error"
	if m.err != nil {
		errMsg = m.err.Error()
	}

	// Detect success messages
	isSuccess := strings.HasPrefix(errMsg, "successfully")
	title := " Error "
	color := "#FF5555"
	textColor := "#FF5555"

	if isSuccess {
		title = " Success "
		color = "#04B575"
		textColor = "#04B575"
	}

	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(textColor))
	boxContent.WriteString(textStyle.Render(fmt.Sprintf("  %s", errMsg)))
	boxContent.WriteString("\n\n")
	boxContent.WriteString(helpStyle.Render("  Press any key to close"))
	boxContent.WriteString("\n")

	// Use appropriate background based on previous state
	var background string
	switch m.previousState {
	case stateProjectSelect, stateNewProject, stateRenameProject, stateConfirmDeleteProject, stateConfirmImport:
		background = m.projectSelectView()
	default:
		background = m.listView()
	}

	return m.renderOverlayDialogWithBackground(title, boxContent.String(), 60, color, background)
}

// confirmUpdateView renders the update confirmation dialog
func (m Model) confirmUpdateView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	if m.updateAvailable != "" {
		boxContent.WriteString(fmt.Sprintf("  Update to %s?\n\n", m.updateAvailable))
	} else {
		boxContent.WriteString("  Check for updates?\n\n")
	}
	boxContent.WriteString(helpStyle.Render("  y: yes  n: no"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Update ", boxContent.String(), 40, "#FFB86C")
}

// checkingUpdateView renders the "checking for updates" message
func (m Model) checkingUpdateView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	boxContent.WriteString("  Checking for updates...\n")
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Update ", boxContent.String(), 40, "#FFB86C")
}

// updatingView shows a download/update progress overlay
func (m Model) updatingView() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(2, 4)

	content := fmt.Sprintf("Downloading %s...\n\nPlease wait...", m.updateAvailable)
	box := boxStyle.Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// updateSuccessView renders the success message after update
func (m Model) updateSuccessView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")

	title := " Success "
	color := "#04B575" // Green
	textColor := "#04B575"

	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(textColor))
	boxContent.WriteString(textStyle.Render(fmt.Sprintf("  %s", m.successMsg)))
	boxContent.WriteString("\n")
	boxContent.WriteString(helpStyle.Render("  Press any key to close"))
	boxContent.WriteString("\n")

	// Use appropriate background based on previous state
	var background string
	switch m.previousState {
	case stateProjectSelect, stateNewProject, stateRenameProject, stateConfirmDeleteProject, stateConfirmImport:
		background = m.projectSelectView()
	default:
		background = m.listView()
	}

	return m.renderOverlayDialogWithBackground(title, boxContent.String(), 60, color, background)
}

// notesView renders the notes editor dialog as an overlay
func (m *Model) notesView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")

	// Determine title and context based on window index
	title := " Session Notes "
	if inst := m.getSelectedInstance(); inst != nil {
		// If there are multiple tabs, always show "Tab Notes"
		hasTabs := len(inst.FollowedWindows) > 0
		if hasTabs {
			title = " Tab Notes "
			tabName := ""
			if m.notesWindowIndex == 0 {
				// Main window - use instance name
				tabName = inst.Name
			} else {
				// Find tab name from FollowedWindows
				for _, fw := range inst.FollowedWindows {
					if fw.Index == m.notesWindowIndex {
						tabName = fw.Name
						break
					}
				}
			}
			if tabName != "" {
				boxContent.WriteString(fmt.Sprintf("  Tab: %s\n\n", tabName))
			}
		} else {
			// Single window - show session notes
			boxContent.WriteString(fmt.Sprintf("  Session: %s\n\n", inst.Name))
		}
	}

	// Dynamic box width (1.5x larger)
	boxWidth := 80
	if m.width > 120 {
		boxWidth = 90
	}
	if boxWidth > 100 {
		boxWidth = 100
	}

	// Set textarea width to box width minus padding (2 on each side + box border)
	m.notesInput.SetWidth(boxWidth - 6)

	// Indent each line of textarea by 2 spaces
	textareaView := m.notesInput.View()
	lines := strings.Split(textareaView, "\n")
	for i, line := range lines {
		boxContent.WriteString("  " + line)
		if i < len(lines)-1 {
			boxContent.WriteString("\n")
		}
	}
	boxContent.WriteString("\n\n")

	// Help text
	helpText := "  ctrl+s: save  esc: cancel  ctrl+d: clear"
	boxContent.WriteString(helpStyle.Render(helpText))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(title, boxContent.String(), boxWidth, "#7D56F4")
}

// newTabChoiceView renders the Agent/Terminal choice dialog
func (m Model) newTabChoiceView() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 1)

	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	boxContent.WriteString("  What type of tab?\n\n")
	boxContent.WriteString("  " + keyStyle.Render("a") + " Agent   - Start new agent in tab\n")
	boxContent.WriteString(dimStyle.Render("              (status tracked, same as main)"))
	boxContent.WriteString("\n\n")
	boxContent.WriteString("  " + keyStyle.Render("t") + " Terminal - Open shell in project dir\n")
	boxContent.WriteString(dimStyle.Render("              (for commands, not tracked)"))
	boxContent.WriteString("\n\n")
	boxContent.WriteString(helpStyle.Render("  esc: cancel"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" New Tab ", boxContent.String(), 50, "#7D56F4")
}

// newTabView renders the new tab creation dialog
func (m Model) newTabView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")

	if m.newTabIsAgent {
		boxContent.WriteString("  New Agent Tab Name:\n")
	} else {
		boxContent.WriteString("  New Terminal Tab Name:\n")
	}
	boxContent.WriteString("  " + m.nameInput.View() + "\n\n")
	boxContent.WriteString(helpStyle.Render("  enter: create  esc: cancel"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" New Tab ", boxContent.String(), 50, "#7D56F4")
}

// renameTabView renders the tab rename dialog
func (m Model) renameTabView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	boxContent.WriteString("  New Tab Name:\n")
	boxContent.WriteString("  " + m.nameInput.View() + "\n\n")
	boxContent.WriteString(helpStyle.Render("  enter: rename  esc: cancel"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Rename Tab ", boxContent.String(), 50, "#7D56F4")
}

// newTabAgentView renders the agent selection dialog for new tab
func (m Model) newTabAgentView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	boxContent.WriteString("  Select Agent for Tab:\n\n")

	// Agent options (same as selectAgentView but for tab)
	agents := []struct {
		agent session.AgentType
		icon  string
		name  string
	}{
		{session.AgentClaude, "🤖", "Claude Code"},
		{session.AgentGemini, "✨", "Gemini"},
		{session.AgentAider, "🔧", "Aider"},
		{session.AgentCodex, "🧠", "Codex CLI"},
		{session.AgentAmazonQ, "📦", "Amazon Q"},
		{session.AgentOpenCode, "💻", "OpenCode"},
		{session.AgentCustom, "⚙️", "Custom"},
	}

	for i, a := range agents {
		if m.newTabAgentCursor == i {
			boxContent.WriteString(fmt.Sprintf("  ❯ %s %s\n", a.icon, a.name))
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

	return m.renderOverlayDialog(" Agent Tab ", boxContent.String(), 40, "#7D56F4")
}

// deleteChoiceView renders the delete choice dialog (session vs tab)
func (m Model) deleteChoiceView() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 1)

	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	if m.deleteTarget != nil {
		boxContent.WriteString(fmt.Sprintf("  Session: %s\n\n", m.deleteTarget.Name))
	}
	boxContent.WriteString("  What to delete?\n\n")
	boxContent.WriteString("  " + keyStyle.Render("s") + " Session - Delete entire session\n")
	boxContent.WriteString(dimStyle.Render("            (stops and removes all tabs)"))
	boxContent.WriteString("\n\n")
	boxContent.WriteString("  " + keyStyle.Render("t") + " Tab     - Close current tab only\n")
	boxContent.WriteString(dimStyle.Render("            (main agent tab cannot be closed)"))
	boxContent.WriteString("\n\n")
	boxContent.WriteString(helpStyle.Render("  esc: cancel"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Delete ", boxContent.String(), 50, "#FF5F87")
}

// confirmDeleteTabView renders the tab deletion confirmation dialog
func (m Model) confirmDeleteTabView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	if m.deleteTarget != nil {
		windows := m.deleteTarget.GetWindowList()
		for _, w := range windows {
			if w.Active {
				boxContent.WriteString(fmt.Sprintf("  Close tab '%s'?\n\n", w.Name))
				break
			}
		}
	}
	boxContent.WriteString(helpStyle.Render("  y: yes  n: no"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Confirm Close Tab ", boxContent.String(), 45, "#FF5F87")
}

// stopChoiceView renders the stop choice dialog (session vs tab)
func (m Model) stopChoiceView() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 1)

	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	if m.stopTarget != nil {
		boxContent.WriteString(fmt.Sprintf("  Session: %s\n\n", m.stopTarget.Name))
	}
	boxContent.WriteString("  What to stop?\n\n")
	boxContent.WriteString("  " + keyStyle.Render("s") + " Session - Stop entire session\n")
	boxContent.WriteString(dimStyle.Render("            (kills all tabs)"))
	boxContent.WriteString("\n\n")
	boxContent.WriteString("  " + keyStyle.Render("t") + " Tab     - Stop current tab only\n")
	boxContent.WriteString(dimStyle.Render("            (sends Ctrl+C/D to tab)"))
	boxContent.WriteString("\n\n")
	boxContent.WriteString(helpStyle.Render("  esc: cancel"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Stop ", boxContent.String(), 50, "#FFA500")
}

// confirmStopTabView renders the tab stop confirmation dialog
func (m Model) confirmStopTabView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	if m.stopTarget != nil {
		windows := m.stopTarget.GetWindowList()
		for _, w := range windows {
			if w.Active {
				boxContent.WriteString(fmt.Sprintf("  Stop tab '%s'?\n\n", w.Name))
				break
			}
		}
	}
	boxContent.WriteString(helpStyle.Render("  y: yes  n: no"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Confirm Stop Tab ", boxContent.String(), 45, "#FFA500")
}

// confirmYoloView renders the YOLO mode confirmation dialog
func (m Model) confirmYoloView() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	var boxContent strings.Builder

	inst := m.yoloTarget
	if inst == nil {
		return m.listView()
	}

	// Determine what we're toggling
	var targetName string
	if m.yoloWindowIndex == 0 {
		targetName = inst.Name
	} else {
		for _, fw := range inst.FollowedWindows {
			if fw.Index == m.yoloWindowIndex {
				targetName = fmt.Sprintf("%s (tab: %s)", inst.Name, fw.Name)
				break
			}
		}
	}

	if m.yoloNewState {
		boxContent.WriteString(fmt.Sprintf("\n\n Enable YOLO mode for:\n %s?\n\n", targetName))
		boxContent.WriteString(" ⚠️  Agent will auto-approve all actions!\n\n")
	} else {
		boxContent.WriteString(fmt.Sprintf("\n\n Disable YOLO mode for:\n %s?\n\n", targetName))
	}

	boxContent.WriteString(helpStyle.Render("  y: yes  n: no"))
	boxContent.WriteString("\n")

	return m.renderOverlayDialog(" Confirm YOLO ", boxContent.String(), 45, "#FFA500")
}
