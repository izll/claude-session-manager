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
func (m Model) promptView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")

	if inst := m.getSelectedInstance(); inst != nil {
		boxContent.WriteString(fmt.Sprintf("  Session: %s\n\n", inst.Name))
	}

	boxContent.WriteString("  Message:\n")
	boxContent.WriteString("  > " + m.promptInput.View() + "\n")

	// Show suggestion if available and input is empty
	if m.promptSuggestion != "" && m.promptInput.Value() == "" {
		suggestionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Italic(true)
		boxContent.WriteString(suggestionStyle.Render(fmt.Sprintf("    → %s", m.promptSuggestion)) + "\n")
	}

	boxContent.WriteString("\n")
	helpText := "  enter: send  esc: cancel"
	if m.promptSuggestion != "" {
		helpText = "  tab: accept  enter: send  esc: cancel"
	}
	boxContent.WriteString(helpStyle.Render(helpText))
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

	if inst := m.getSelectedInstance(); inst != nil {
		boxContent.WriteString(fmt.Sprintf("  Session: %s\n\n", inst.Name))
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

	return m.renderOverlayDialog(" Session Notes ", boxContent.String(), boxWidth, "#7D56F4")
}
