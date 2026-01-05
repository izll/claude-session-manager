package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/izll/agent-session-manager/session"
	"github.com/izll/agent-session-manager/updater"
)

// showError displays an error in a dialog and remembers the current state to return to
func (m *Model) showError(err error) {
	m.previousState = m.state
	m.err = err
	m.state = stateError
}

// handleMoveSessionUp moves the selected session or group up in the list
func (m *Model) handleMoveSessionUp() {
	// If groups exist, handle grouped reordering
	if len(m.groups) > 0 {
		m.buildVisibleItems()
		if m.cursor <= 0 || m.cursor >= len(m.visibleItems) {
			return
		}
		currentItem := m.visibleItems[m.cursor]
		prevItem := m.visibleItems[m.cursor-1]

		// Moving a group up
		if currentItem.isGroup {
			groupIdx := m.findGroupIndex(currentItem.group.ID)
			if groupIdx > 0 {
				m.groups[groupIdx], m.groups[groupIdx-1] = m.groups[groupIdx-1], m.groups[groupIdx]
				m.storage.SaveWithGroups(m.instances, m.groups)
				m.buildVisibleItems()
				// Find new cursor position
				for i, item := range m.visibleItems {
					if item.isGroup && item.group.ID == currentItem.group.ID {
						m.cursor = i
						break
					}
				}
			}
			return
		}

		// If previous is a group header, can't move further up
		if prevItem.isGroup {
			return
		}

		// Both are sessions - swap in the instances array (keep original groups)
		currentIdx := m.findInstanceIndex(currentItem.instance.ID)
		prevIdx := m.findInstanceIndex(prevItem.instance.ID)
		if currentIdx >= 0 && prevIdx >= 0 {
			m.instances[currentIdx], m.instances[prevIdx] = m.instances[prevIdx], m.instances[currentIdx]
			m.cursor--
			m.storage.Save(m.instances)
		}
		return
	}

	// Original behavior for non-grouped view
	if m.cursor > 0 && len(m.instances) > 1 {
		m.instances[m.cursor], m.instances[m.cursor-1] = m.instances[m.cursor-1], m.instances[m.cursor]
		m.cursor--
		m.storage.Save(m.instances)
	}
}

// handleMoveSessionDown moves the selected session or group down in the list
func (m *Model) handleMoveSessionDown() {
	// If groups exist, handle grouped reordering
	if len(m.groups) > 0 {
		m.buildVisibleItems()
		if m.cursor < 0 || m.cursor >= len(m.visibleItems)-1 {
			return
		}
		currentItem := m.visibleItems[m.cursor]
		nextItem := m.visibleItems[m.cursor+1]

		// Moving a group down
		if currentItem.isGroup {
			groupIdx := m.findGroupIndex(currentItem.group.ID)
			if groupIdx >= 0 && groupIdx < len(m.groups)-1 {
				m.groups[groupIdx], m.groups[groupIdx+1] = m.groups[groupIdx+1], m.groups[groupIdx]
				m.storage.SaveWithGroups(m.instances, m.groups)
				m.buildVisibleItems()
				// Find new cursor position
				for i, item := range m.visibleItems {
					if item.isGroup && item.group.ID == currentItem.group.ID {
						m.cursor = i
						break
					}
				}
			}
			return
		}

		// If next is a group header, can't move further down
		if nextItem.isGroup {
			return
		}

		// Both are sessions - swap in the instances array (keep original groups)
		currentIdx := m.findInstanceIndex(currentItem.instance.ID)
		nextIdx := m.findInstanceIndex(nextItem.instance.ID)
		if currentIdx >= 0 && nextIdx >= 0 {
			m.instances[currentIdx], m.instances[nextIdx] = m.instances[nextIdx], m.instances[currentIdx]
			m.cursor++
			m.storage.Save(m.instances)
		}
		return
	}

	// Original behavior for non-grouped view
	if m.cursor < len(m.instances)-1 {
		m.instances[m.cursor], m.instances[m.cursor+1] = m.instances[m.cursor+1], m.instances[m.cursor]
		m.cursor++
		m.storage.Save(m.instances)
	}
}

// findInstanceIndex finds the index of an instance in the instances array by ID
func (m *Model) findInstanceIndex(id string) int {
	for i, inst := range m.instances {
		if inst.ID == id {
			return i
		}
	}
	return -1
}

// findGroupIndex finds the index of a group in the groups array by ID
func (m *Model) findGroupIndex(id string) int {
	for i, g := range m.groups {
		if g.ID == id {
			return i
		}
	}
	return -1
}

// saveSettings saves UI settings to storage
func (m *Model) saveSettings() {
	m.storage.SaveSettings(&session.Settings{
		CompactList:     m.compactList,
		HideStatusLines: m.hideStatusLines,
		SplitView:       m.splitView,
		MarkedSessionID: m.markedSessionID,
		Cursor:          m.cursor,
		SplitFocus:      m.splitFocus,
	})
}

// getScrollableContent returns the content to use for scrolling
func (m *Model) getScrollableContent() string {
	if m.scrollContent != "" {
		return m.scrollContent
	}
	return m.preview
}

// resetScroll resets scroll state when changing sessions
func (m *Model) resetScroll() {
	m.previewScroll = 0
	m.scrollContent = ""
}

// getPreviewMaxLines returns the maximum number of lines visible in the preview pane
func (m *Model) getPreviewMaxLines() int {
	contentHeight := m.height - 1
	if m.splitView {
		// In split view, each pane gets half the height
		halfHeight := (contentHeight - 1) / 2
		maxLines := halfHeight - 2 // -2 for header and margin in buildMiniPreview
		if maxLines < 2 {
			maxLines = 2
		}
		return maxLines
	}
	// Normal view
	maxLines := contentHeight - PreviewHeaderHeight
	if maxLines < MinPreviewLines {
		maxLines = MinPreviewLines
	}
	return maxLines
}

// navigatePinned changes the pinned session in split view
func (m *Model) navigatePinned(direction int) {
	if len(m.instances) == 0 {
		return
	}

	// If groups exist, use visible items
	if len(m.groups) > 0 {
		m.buildVisibleItems()

		// Find current pinned index in visible items (sessions only)
		currentIdx := -1
		for i, item := range m.visibleItems {
			if !item.isGroup && item.instance.ID == m.markedSessionID {
				currentIdx = i
				break
			}
		}

		if currentIdx == -1 {
			return
		}

		// Find next/previous session (skip groups)
		newIdx := currentIdx + direction
		for newIdx >= 0 && newIdx < len(m.visibleItems) {
			if !m.visibleItems[newIdx].isGroup {
				m.markedSessionID = m.visibleItems[newIdx].instance.ID
				return
			}
			newIdx += direction
		}
		return
	}

	// Original behavior for non-grouped view
	currentIdx := -1
	for i, inst := range m.instances {
		if inst.ID == m.markedSessionID {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		return
	}

	newIdx := currentIdx + direction
	if newIdx < 0 {
		newIdx = 0
	} else if newIdx >= len(m.instances) {
		newIdx = len(m.instances) - 1
	}

	m.markedSessionID = m.instances[newIdx].ID
}

// handleEnterSession starts (if needed) and attaches to the selected session
func (m *Model) handleEnterSession() tea.Cmd {
	var inst *session.Instance

	// In split view with focus on pinned, attach to pinned session
	if m.splitView && m.splitFocus == 1 && m.markedSessionID != "" {
		for _, i := range m.instances {
			if i.ID == m.markedSessionID {
				inst = i
				break
			}
		}
	} else {
		inst = m.getSelectedInstance()
	}

	if inst == nil {
		return nil
	}
	if inst.Status != session.StatusRunning {
		// Check if command exists before starting
		if err := session.CheckAgentCommand(inst); err != nil {
			m.err = err
			m.previousState = stateList
			m.state = stateError
			return nil
		}
		if err := inst.Start(); err != nil {
			m.err = err
			m.previousState = stateList
			m.state = stateError
			return nil
		}
		m.storage.UpdateInstance(inst)
	}
	sessionName := inst.TmuxSessionName()
	// Configure tmux for proper terminal resize following (ignore errors - non-critical)
	exec.Command("tmux", "set-option", "-t", sessionName, "window-size", "largest").Run()
	exec.Command("tmux", "set-option", "-t", sessionName, "aggressive-resize", "on").Run()
	// Enable focus events for hooks to work
	exec.Command("tmux", "set-option", "-t", sessionName, "focus-events", "on").Run()
	// Set up hook to resize window on focus gain (fixes Konsole tab switch issue)
	exec.Command("tmux", "set-hook", "-t", sessionName, "client-focus-in", "resize-window -A").Run()
	exec.Command("tmux", "set-hook", "-t", sessionName, "pane-focus-in", "resize-window -A").Run()
	// Set up Ctrl+Q to resize to preview size before detach
	tmuxWidth, tmuxHeight := m.calculateTmuxDimensions()
	inst.UpdateDetachBinding(tmuxWidth, tmuxHeight)
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return reattachMsg{}
	})
}

// handleResumeSession shows Claude sessions for the current instance
func (m *Model) handleResumeSession() error {
	inst := m.getSelectedInstance()
	if inst == nil {
		return nil
	}
	// List sessions based on agent type
	var sessions []session.AgentSession
	var err error

	switch inst.Agent {
	case session.AgentGemini:
		sessions, err = session.ListGeminiSessions(inst.Path)
	case session.AgentCodex:
		sessions, err = session.ListCodexSessions(inst.Path)
	case session.AgentOpenCode:
		sessions, err = session.ListOpenCodeSessions(inst.Path)
	case session.AgentAmazonQ:
		sessions, err = session.ListAmazonQSessions(inst.Path)
	default:
		// Claude and others
		sessions, err = session.ListAgentSessions(inst.Path)
	}

	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		return fmt.Errorf("no previous %s sessions found", inst.Agent)
	}
	m.agentSessions = sessions
	m.sessionCursor = 1 // Start with first session selected (0 is "new session")
	m.state = stateSelectAgentSession
	return nil
}

// handleStartSession starts the selected session without attaching
func (m *Model) handleStartSession() {
	inst := m.getSelectedInstance()
	if inst == nil {
		return
	}
	if inst.Status != session.StatusRunning {
		// Check if command exists before starting
		if err := session.CheckAgentCommand(inst); err != nil {
			m.err = err
			m.previousState = stateList
			m.state = stateError
			return
		}
		if err := inst.Start(); err != nil {
			m.err = err
			m.previousState = stateList
			m.state = stateError
		} else {
			m.storage.UpdateInstance(inst)
		}
	}
}

// handleStopSession stops the selected session
func (m *Model) handleStopSession() {
	inst := m.getSelectedInstance()
	if inst == nil {
		return
	}
	if inst.Status == session.StatusRunning {
		inst.Stop()
		m.storage.UpdateInstance(inst)
	}
}

// handleRenameSession opens the rename dialog for the selected session
func (m *Model) handleRenameSession() tea.Cmd {
	inst := m.getSelectedInstance()
	if inst == nil {
		return nil
	}
	m.nameInput.SetValue(inst.Name)
	m.nameInput.Focus()
	m.state = stateRename
	return textinput.Blink
}

// handleColorPicker opens the color picker for the selected session
func (m *Model) handleColorPicker() {
	inst := m.getSelectedInstance()
	if inst == nil {
		return
	}
	// Initialize preview colors
	m.previewFg = inst.Color
	m.previewBg = inst.BgColor
	m.colorMode = 0
	m.editingGroup = nil
	// Find current color index in filtered list
	m.colorCursor = 0
	filteredColors := m.getFilteredColorOptions()
	for i, c := range filteredColors {
		if c.Color == inst.Color || c.Name == inst.Color {
			m.colorCursor = i
			break
		}
	}
	m.state = stateColorPicker
}

// handleGroupColorPicker opens the color picker for a group
func (m *Model) handleGroupColorPicker(group *session.Group) {
	m.editingGroup = group
	m.previewFg = group.Color
	m.previewBg = group.BgColor
	m.colorMode = 0
	// Find current color index in filtered list
	m.colorCursor = 0
	filteredColors := m.getFilteredColorOptions()
	for i, c := range filteredColors {
		if c.Color == group.Color || c.Name == group.Color {
			m.colorCursor = i
			break
		}
	}
	m.state = stateColorPicker
}

// handleSendPrompt opens the prompt input for the selected session
func (m *Model) handleSendPrompt() {
	inst := m.getSelectedInstance()
	if inst == nil {
		return
	}
	if inst.Status != session.StatusRunning {
		m.err = fmt.Errorf("session not running")
		m.previousState = stateList
		m.state = stateError
		return
	}
	m.promptInput.SetValue("")
	inputWidth := PromptMinWidth
	if m.width > 80 {
		inputWidth = m.width/2 - 10
	}
	if inputWidth > PromptMaxWidth {
		inputWidth = PromptMaxWidth
	}
	m.promptInput.Width = inputWidth
	m.promptInput.Focus()

	// Get suggestion from agent
	m.promptSuggestion = inst.GetSuggestion()

	m.state = statePrompt
}

// handleForceResize forces resize of the selected pane
func (m *Model) handleForceResize() {
	inst := m.getSelectedInstance()
	if inst == nil {
		return
	}
	tmuxWidth, tmuxHeight := m.calculateTmuxDimensions()
	if err := inst.ResizePane(tmuxWidth, tmuxHeight); err != nil {
		m.err = fmt.Errorf("failed to resize pane: %w", err)
		m.previousState = stateList
		m.state = stateError
	}
}

// handleToggleAutoYes toggles the auto-yes flag on the selected session
// Returns a tea.Cmd to attach to the session if it was restarted
func (m *Model) handleToggleAutoYes() tea.Cmd {
	inst := m.getSelectedInstance()
	if inst == nil {
		return nil
	}

	// Get agent type (empty string means Claude for backward compatibility)
	agentType := inst.Agent
	if agentType == "" {
		agentType = session.AgentClaude
	}

	// Special handling for Gemini - send Ctrl+Y keystroke instead
	if agentType == session.AgentGemini {
		if inst.Status == session.StatusRunning {
			if err := inst.SendKeys("C-y"); err != nil {
				m.err = fmt.Errorf("failed to send Ctrl+Y: %w", err)
				m.previousState = stateList
				m.state = stateError
			}
		}
		return nil
	}

	// Check if agent supports AutoYes
	config := session.AgentConfigs[agentType]
	if !config.SupportsAutoYes {
		m.err = fmt.Errorf("yolo mode not supported for %s agent", agentType)
		m.previousState = stateList
		m.state = stateError
		return nil
	}

	// Toggle AutoYes
	wasRunning := inst.Status == session.StatusRunning
	inst.AutoYes = !inst.AutoYes
	m.storage.UpdateInstance(inst)

	// If running, restart with new flag (no auto-attach in list view)
	if wasRunning {
		inst.Stop()
		if err := inst.Start(); err != nil {
			m.err = fmt.Errorf("failed to restart session: %w", err)
			m.previousState = stateList
			m.state = stateError
			return nil
		}
		m.storage.UpdateInstance(inst)
	}

	return nil
}

// handleListKeys handles keyboard input in the main list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear error on any key press
	m.err = nil

	switch msg.String() {
	case "q", "ctrl+c":
		m.saveSettings() // Save cursor position on quit
		m.storage.UnlockProject()
		return m, tea.Quit

	case "up", "k":
		// In split view with focus on pinned: change pinned session
		if m.splitView && m.splitFocus == 1 && m.markedSessionID != "" {
			m.navigatePinned(-1)
			m.saveSettings()
		} else if len(m.groups) > 0 {
			m.buildVisibleItems()
			if m.cursor > 0 {
				m.cursor--
				m.resetScroll()
				m.resizeSelectedPane()
			}
		} else if m.cursor > 0 {
			m.cursor--
			m.resetScroll()
			m.resizeSelectedPane()
		}

	case "down", "j":
		// In split view with focus on pinned: change pinned session
		if m.splitView && m.splitFocus == 1 && m.markedSessionID != "" {
			m.navigatePinned(1)
			m.saveSettings()
		} else if len(m.groups) > 0 {
			m.buildVisibleItems()
			if m.cursor < len(m.visibleItems)-1 {
				m.cursor++
				m.resetScroll()
				m.resizeSelectedPane()
			}
		} else if m.cursor < len(m.instances)-1 {
			m.cursor++
			m.resetScroll()
			m.resizeSelectedPane()
		}

	case "ctrl+up":
		m.handleMoveSessionUp()

	case "ctrl+down":
		m.handleMoveSessionDown()

	case "shift+up", "pgup", "shift+pgup", "K":
		// Scroll preview up - fetch extended content on first scroll
		if m.scrollContent == "" {
			inst := m.getSelectedInstance()
			if inst != nil && inst.Status == session.StatusRunning {
				m.scrollContent, _ = inst.GetPreview(ScrollbackLines)
			}
		}
		content := m.getScrollableContent()
		if content == "" {
			break
		}
		lines := strings.Split(content, "\n")
		maxLines := m.getPreviewMaxLines()
		maxScroll := len(lines) - maxLines
		if maxScroll < 0 {
			maxScroll = 0
		}
		m.previewScroll += 5
		if m.previewScroll > maxScroll {
			m.previewScroll = maxScroll
		}

	case "shift+down", "pgdown", "shift+pgdown", "J":
		// Scroll preview down
		m.previewScroll -= 5
		if m.previewScroll < 0 {
			m.previewScroll = 0
		}

	case "home":
		// Scroll to top of preview - fetch extended content
		if m.scrollContent == "" {
			inst := m.getSelectedInstance()
			if inst != nil && inst.Status == session.StatusRunning {
				m.scrollContent, _ = inst.GetPreview(ScrollbackLines)
			}
		}
		content := m.getScrollableContent()
		if content == "" {
			break
		}
		lines := strings.Split(content, "\n")
		maxLines := m.getPreviewMaxLines()
		maxScroll := len(lines) - maxLines
		if maxScroll > 0 {
			m.previewScroll = maxScroll
		}

	case "end":
		// Scroll to bottom of preview
		m.previewScroll = 0

	case "enter":
		// Check if a group is selected
		if len(m.groups) > 0 {
			m.buildVisibleItems()
			if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
				item := m.visibleItems[m.cursor]
				if item.isGroup {
					// Toggle collapse
					m.storage.ToggleGroupCollapsed(item.group.ID)
					groups, _ := m.storage.GetGroups()
					m.groups = groups
					m.buildVisibleItems()
					return m, nil
				}
			}
		}
		if cmd := m.handleEnterSession(); cmd != nil {
			return m, cmd
		}

	case "n":
		// Start new session flow: agent selection -> path -> name
		m.agentCursor = 0
		m.pendingAgent = session.AgentClaude
		m.pendingGroupID = m.getCurrentGroupID()
		m.state = stateSelectAgent
		return m, nil

	case "r":
		// Resume only works for agents that support it
		if inst := m.getSelectedInstance(); inst != nil {
			config := inst.GetAgentConfig()
			if !config.SupportsResume {
				m.err = fmt.Errorf("resume not supported for %s agent", inst.Agent)
				m.previousState = m.state // Save current state to return after error
				m.state = stateError
				return m, nil
			}
		}
		if err := m.handleResumeSession(); err != nil {
			m.err = err
			m.previousState = m.state // Save current state to return after error
			m.state = stateError
		}

	case "s":
		m.handleStartSession()

	case "a":
		// Show start mode selection (replace or parallel)
		if inst := m.getSelectedInstance(); inst != nil {
			m.state = stateSelectStartMode
		}

	case "x":
		m.handleStopSession()

	case "d":
		// Check if a group is selected
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.isGroup {
				// Delete group
				if err := m.storage.RemoveGroup(item.group.ID); err != nil {
					m.err = err
				} else {
					// Reload groups
					groups, _ := m.storage.GetGroups()
					m.groups = groups
					m.buildVisibleItems()
					if m.cursor >= len(m.visibleItems) && m.cursor > 0 {
						m.cursor--
					}
				}
				return m, nil
			}
		}
		// Delete session
		if inst := m.getSelectedInstance(); inst != nil {
			m.deleteTarget = inst
			m.state = stateConfirmDelete
		}

	case "ctrl+y":
		if cmd := m.handleToggleAutoYes(); cmd != nil {
			return m, cmd
		}

	case "e":
		// Check if a group is selected
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.isGroup {
				// Rename group
				m.groupInput.SetValue(item.group.Name)
				m.groupInput.Focus()
				m.state = stateRenameGroup
				return m, textinput.Blink
			}
		}
		// Rename session
		if cmd := m.handleRenameSession(); cmd != nil {
			return m, cmd
		}

	case "?", "f1":
		m.state = stateHelp

	case "c":
		// Check if a group is selected
		if len(m.groups) > 0 {
			m.buildVisibleItems()
			if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
				item := m.visibleItems[m.cursor]
				if item.isGroup {
					m.handleGroupColorPicker(item.group)
					return m, nil
				}
			}
		}
		m.handleColorPicker()

	case "l":
		m.compactList = !m.compactList
		m.saveSettings()

	case "t":
		m.hideStatusLines = !m.hideStatusLines
		m.saveSettings()

	case "v":
		m.splitView = !m.splitView
		m.splitFocus = 0 // Reset focus when toggling
		m.saveSettings()

	case "tab":
		// In split view: switch focus between panels
		if m.splitView && m.markedSessionID != "" {
			m.splitFocus = 1 - m.splitFocus // Toggle between 0 and 1
			m.saveSettings()
		} else {
			// In normal view: toggle group collapse
			m.buildVisibleItems()
			if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
				item := m.visibleItems[m.cursor]
				if item.isGroup {
					m.storage.ToggleGroupCollapsed(item.group.ID)
					// Reload groups
					groups, _ := m.storage.GetGroups()
					m.groups = groups
					m.buildVisibleItems()
				}
			}
		}

	case "m":
		// Mark current session for split view
		inst := m.getSelectedInstance()
		if inst != nil {
			if m.markedSessionID == inst.ID {
				m.markedSessionID = "" // Unmark if already marked
			} else {
				m.markedSessionID = inst.ID
			}
			m.saveSettings()
		}

	case "p":
		m.handleSendPrompt()

	case "R":
		m.handleForceResize()

	case "U":
		// Show update confirmation
		m.state = stateConfirmUpdate
		return m, nil

	case "P":
		// Go back to project selection
		m.projectCursor = 0
		m.state = stateProjectSelect
		return m, nil

	case "g":
		// Create new group
		m.groupInput.SetValue("")
		m.groupInput.Focus()
		m.state = stateNewGroup
		return m, textinput.Blink

	case "G":
		// Assign session to group
		if len(m.instances) > 0 {
			// Find current session
			var inst *session.Instance
			if len(m.groups) > 0 {
				m.buildVisibleItems()
				if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
					item := m.visibleItems[m.cursor]
					if !item.isGroup {
						inst = item.instance
					}
				}
			} else if m.cursor < len(m.instances) {
				inst = m.instances[m.cursor]
			}

			// Pre-select current group
			m.groupCursor = 0 // Default to "No Group"
			if inst != nil && inst.GroupID != "" {
				for i, g := range m.groups {
					if g.ID == inst.GroupID {
						m.groupCursor = i + 1 // +1 because 0 is "No Group"
						break
					}
				}
			}
			m.state = stateSelectGroup
		}

	case "right":
		// Expand group
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.isGroup && item.group.Collapsed {
				m.storage.ToggleGroupCollapsed(item.group.ID)
				groups, _ := m.storage.GetGroups()
				m.groups = groups
				m.buildVisibleItems()
			}
		}

	case "left":
		// Collapse group
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.isGroup && !item.group.Collapsed {
				m.storage.ToggleGroupCollapsed(item.group.ID)
				groups, _ := m.storage.GetGroups()
				m.groups = groups
				m.buildVisibleItems()
			}
		}
	}

	return m, nil
}

// handleNewNameKeys handles keyboard input in the new session name dialog
func (m Model) handleNewNameKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pendingInstance = nil
		m.isParallelSession = false
		m.parallelOriginalID = ""
		m.state = stateList
		return m, nil
	case "enter":
		if m.nameInput.Value() != "" {
			// Check if we're creating a parallel session
			if m.isParallelSession && m.pendingInstance != nil {
				// Parallel session: just update name and insert
				inst := m.pendingInstance
				inst.Name = m.nameInput.Value()

				// Check if command exists before starting
				if err := session.CheckAgentCommand(inst); err != nil {
					m.err = err
					m.previousState = stateList
					m.state = stateError
					m.isParallelSession = false
					m.pendingInstance = nil
					m.parallelOriginalID = ""
					return m, nil
				}

				// Find the original instance and insert after it
				currentIdx := m.findInstanceIndex(m.parallelOriginalID)
				if currentIdx >= 0 {
					m.instances = append(m.instances[:currentIdx+1], append([]*session.Instance{inst}, m.instances[currentIdx+1:]...)...)
				} else {
					// Fallback: append to end
					m.instances = append(m.instances, inst)
				}

				// Start the new instance
				if err := inst.Start(); err != nil {
					m.err = err
					m.previousState = stateList
					m.state = stateError
				} else {
					m.storage.Save(m.instances)

					// Rebuild visible items and find the new instance
					if len(m.groups) > 0 {
						m.buildVisibleItems()
						for i, item := range m.visibleItems {
							if !item.isGroup && item.instance != nil && item.instance.ID == inst.ID {
								m.cursor = i
								break
							}
						}
					} else {
						// Non-grouped mode: find instance index
						for i, existingInst := range m.instances {
							if existingInst.ID == inst.ID {
								m.cursor = i
								break
							}
						}
					}
				}

				m.pendingInstance = nil
				m.isParallelSession = false
				m.parallelOriginalID = ""
				m.state = stateList
				return m, nil
			}

			// Normal session creation: create new instance
			inst, err := session.NewInstance(m.nameInput.Value(), m.pathInput.Value(), m.autoYes, m.pendingAgent)
			if err != nil {
				m.err = err
				m.state = stateList
				return m, nil
			}

			// Set custom command for custom agent
			if m.pendingAgent == session.AgentCustom {
				inst.CustomCommand = m.customCmdInput.Value()
			}

			// Assign to current group if any
			if m.pendingGroupID != "" {
				inst.GroupID = m.pendingGroupID
			}

			// Check if the agent command exists before creating session
			if err := session.CheckAgentCommand(inst); err != nil {
				m.err = err
				m.state = stateList
				return m, nil
			}

			// Check for existing agent sessions (for agents that support resume)
			agentConfig := session.AgentConfigs[m.pendingAgent]
			if agentConfig.SupportsResume {
				var sessions []session.AgentSession
				var err error

				switch m.pendingAgent {
				case session.AgentGemini:
					sessions, err = session.ListGeminiSessions(inst.Path)
				case session.AgentCodex:
					sessions, err = session.ListCodexSessions(inst.Path)
				case session.AgentOpenCode:
					sessions, err = session.ListOpenCodeSessions(inst.Path)
				case session.AgentAmazonQ:
					sessions, err = session.ListAmazonQSessions(inst.Path)
				default:
					// Claude and others
					sessions, err = session.ListAgentSessions(inst.Path)
				}

				if err != nil {
					// Non-fatal: just continue without session selection
					sessions = nil
				}
				if len(sessions) > 0 {
					m.pendingInstance = inst
					m.agentSessions = sessions
					m.sessionCursor = 1 // Start with first session selected (0 is "new session")
					m.state = stateSelectAgentSession
					return m, nil
				}
			}

			// No existing sessions or agent doesn't support resume, just create new
			if err := m.storage.AddInstance(inst); err != nil {
				m.err = err
				m.state = stateList
				return m, nil
			}

			// Auto-start the new instance
			if err := inst.Start(); err != nil {
				m.err = err
				m.previousState = stateList
				m.state = stateError
			} else {
				m.storage.UpdateInstance(inst)
			}

			m.instances = append(m.instances, inst)

			// Set cursor to the new instance
			if len(m.groups) > 0 {
				// In grouped mode, find the instance in visibleItems
				m.buildVisibleItems()
				for i, item := range m.visibleItems {
					if !item.isGroup && item.instance != nil && item.instance.ID == inst.ID {
						m.cursor = i
						break
					}
				}
			} else {
				m.cursor = len(m.instances) - 1
			}

			m.state = stateList
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

// handleNewPathKeys handles keyboard input in the new session path dialog
func (m Model) handleNewPathKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "enter":
		if m.pathInput.Value() != "" {
			// Extract folder name as default session name
			path := m.pathInput.Value()
			folderName := filepath.Base(path)
			if folderName == "." || folderName == "/" {
				folderName = "session"
			}

			m.nameInput.SetValue(folderName)
			m.nameInput.Focus()
			m.state = stateNewName
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

// handleSelectSessionKeys handles keyboard input in the Claude session selector
func (m Model) handleSelectSessionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxIdx := len(m.agentSessions) // max index (0 = new session, 1+ = existing sessions)

	switch msg.String() {
	case "q", "esc":
		m.agentSessions = nil
		m.pendingInstance = nil
		m.state = stateList
		return m, nil

	case "up", "k":
		if m.sessionCursor > 0 {
			m.sessionCursor--
		}

	case "down", "j":
		if m.sessionCursor < maxIdx {
			m.sessionCursor++
		}

	case "shift+up", "shift+pgup", "pgup":
		// Jump 5 items up
		m.sessionCursor -= 5
		if m.sessionCursor < 0 {
			m.sessionCursor = 0
		}

	case "shift+down", "shift+pgdown", "pgdown":
		// Jump 5 items down
		m.sessionCursor += 5
		if m.sessionCursor > maxIdx {
			m.sessionCursor = maxIdx
		}

	case "home":
		m.sessionCursor = 0

	case "end":
		m.sessionCursor = maxIdx

	case "enter":
		var resumeID string
		if m.sessionCursor > 0 && m.sessionCursor <= len(m.agentSessions) {
			// Selected an existing session
			resumeID = m.agentSessions[m.sessionCursor-1].SessionID
		}
		// sessionCursor == 0 means "Start new session"

		if m.pendingInstance != nil {
			// Creating new instance
			inst := m.pendingInstance
			inst.ResumeSessionID = resumeID

			if err := m.storage.AddInstance(inst); err != nil {
				m.err = err
				m.state = stateList
				m.pendingInstance = nil
				m.agentSessions = nil
				return m, nil
			}

			// Auto-start the new instance
			if err := inst.StartWithResume(resumeID); err != nil {
				m.err = err
			} else {
				m.storage.UpdateInstance(inst)
			}

			m.instances = append(m.instances, inst)

			// Set cursor to the new instance
			if len(m.groups) > 0 {
				m.buildVisibleItems()
				for i, item := range m.visibleItems {
					if !item.isGroup && item.instance != nil && item.instance.ID == inst.ID {
						m.cursor = i
						break
					}
				}
			} else {
				m.cursor = len(m.instances) - 1
			}

			m.pendingInstance = nil
		} else if inst := m.getSelectedInstance(); inst != nil {
			// Resuming existing instance
			if inst.Status == session.StatusRunning {
				inst.Stop()
			}
			inst.ResumeSessionID = resumeID
			if err := inst.StartWithResume(resumeID); err != nil {
				m.err = err
			} else {
				m.storage.UpdateInstance(inst)
			}
		}

		m.agentSessions = nil
		m.state = stateList
		return m, nil
	}

	return m, nil
}

// handleConfirmDeleteKeys handles keyboard input in the delete confirmation dialog
func (m Model) handleConfirmDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.deleteTarget != nil {
			if err := m.storage.RemoveInstance(m.deleteTarget.ID); err != nil {
				m.err = fmt.Errorf("failed to remove instance: %w", err)
			}
			// Reload instances
			instances, err := m.storage.Load()
			if err != nil {
				m.err = fmt.Errorf("failed to reload instances: %w", err)
			} else {
				m.instances = instances
			}
			if m.cursor >= len(m.instances) && m.cursor > 0 {
				m.cursor--
			}
		}
		m.deleteTarget = nil
		m.state = stateList
	case "n", "N", "esc":
		m.deleteTarget = nil
		m.state = stateList
	}
	return m, nil
}

// handleConfirmStartKeys handles keyboard input in the auto-start confirmation dialog
func (m Model) handleConfirmStartKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		inst := m.getSelectedInstance()
		if inst != nil {
			// Stop if already running
			if inst.Status == session.StatusRunning {
				inst.Stop()
			}

			// Clear any resume session ID to ensure we start fresh
			inst.ResumeSessionID = ""

			// Check if command exists before starting
			if err := session.CheckAgentCommand(inst); err != nil {
				m.err = err
				m.previousState = stateList
				m.state = stateError
				return m, nil
			}

			// Start completely new session (no resume)
			if err := inst.Start(); err != nil {
				m.err = err
				m.previousState = stateList
				m.state = stateError
			} else {
				m.storage.UpdateInstance(inst)
			}
		}
		m.state = stateList
	case "n", "N", "esc":
		m.state = stateList
	}
	return m, nil
}

// handleSelectStartModeKeys handles keyboard input for start mode selection
func (m Model) handleSelectStartModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1", "r", "R":
		// Replace current session - go to confirm dialog
		m.state = stateConfirmStart
	case "2", "n", "N":
		// Start parallel session - ask for name first
		inst := m.getSelectedInstance()
		if inst != nil {
			// Create a new instance based on the current one
			newInst, err := session.NewInstance(inst.Name, inst.Path, inst.AutoYes, inst.Agent)
			if err != nil {
				m.err = err
				m.previousState = stateList
				m.state = stateError
				return m, nil
			}

			// Copy relevant settings
			newInst.CustomCommand = inst.CustomCommand
			newInst.GroupID = inst.GroupID
			newInst.Color = inst.Color
			newInst.BgColor = inst.BgColor
			newInst.FullRowColor = inst.FullRowColor

			// Store as pending instance for name input
			m.pendingInstance = newInst
			m.isParallelSession = true
			m.parallelOriginalID = inst.ID

			// Set default name to current session name and ask for name
			m.nameInput.SetValue(inst.Name)
			m.nameInput.Focus()
			m.state = stateNewName
			return m, textinput.Blink
		}
		m.state = stateList
	case "esc":
		m.state = stateList
	}
	return m, nil
}

// handleRenameKeys handles keyboard input in the rename dialog
func (m Model) handleRenameKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "enter":
		if m.nameInput.Value() != "" {
			if inst := m.getSelectedInstance(); inst != nil {
				inst.Name = m.nameInput.Value()
				m.storage.UpdateInstance(inst)
			}
			m.state = stateList
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

// handleHelpKeys handles keyboard input in the help view
func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?", "f1", "F1":
		m.state = stateList
		m.helpScroll = 0 // Reset scroll when closing
		return m, nil
	case "up", "k", "shift+up", "pgup":
		// Scroll up
		if m.helpScroll > 0 {
			m.helpScroll--
		}
	case "down", "j", "shift+down", "pgdown":
		// Scroll down
		m.helpScroll++
	case "home":
		m.helpScroll = 0
	case "end":
		m.helpScroll = 999 // Will be clamped in view
	}
	return m, nil
}

// handlePromptKeys handles keyboard input in the prompt dialog
func (m Model) handlePromptKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "tab":
		// Accept suggestion if available and input is empty
		if m.promptSuggestion != "" && m.promptInput.Value() == "" {
			m.promptInput.SetValue(m.promptSuggestion)
			m.promptInput.CursorEnd()
			return m, nil
		}
	case "enter":
		if m.promptInput.Value() != "" {
			if inst := m.getSelectedInstance(); inst != nil && inst.Status == session.StatusRunning {
				// Send prompt text followed by Enter in a single command
				text := m.promptInput.Value()
				if err := inst.SendPrompt(text); err != nil {
					m.err = err
				}
			}
			m.state = stateList
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.promptInput, cmd = m.promptInput.Update(msg)
	return m, cmd
}

// handleColorPickerKeys handles keyboard input in the color picker
func (m Model) handleColorPickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxItems := m.getMaxColorItems()

	switch msg.String() {
	case "esc":
		m.state = stateList
		m.colorMode = 0
		return m, nil

	case "tab":
		// Save current preview before switching
		currentFiltered := m.getFilteredColorOptions()
		if m.colorCursor < len(currentFiltered) {
			selected := currentFiltered[m.colorCursor]
			if m.colorMode == 0 {
				m.previewFg = selected.Color
			} else {
				m.previewBg = selected.Color
			}
		}
		// Switch between foreground and background mode
		m.colorMode = 1 - m.colorMode
		// Find cursor position for the other mode's current value in filtered list
		m.colorCursor = 0
		targetColor := m.previewFg
		if m.colorMode == 1 {
			targetColor = m.previewBg
		}
		filteredColors := m.getFilteredColorOptions()
		for i, c := range filteredColors {
			if c.Color == targetColor {
				m.colorCursor = i
				break
			}
		}
		// Reset cursor if it's beyond the new max
		if m.colorCursor >= len(filteredColors) {
			m.colorCursor = 0
		}

	case "up", "k":
		if m.colorCursor > 0 {
			m.colorCursor--
		}

	case "down", "j":
		if m.colorCursor < maxItems-1 {
			m.colorCursor++
		}

	case "pgup", "ctrl+u":
		m.colorCursor -= 10
		if m.colorCursor < 0 {
			m.colorCursor = 0
		}

	case "pgdown", "ctrl+d":
		m.colorCursor += 10
		if m.colorCursor >= maxItems {
			m.colorCursor = maxItems - 1
		}

	case "home":
		m.colorCursor = 0

	case "end":
		m.colorCursor = maxItems - 1

	case "f":
		// Toggle full row color
		if m.editingGroup != nil {
			m.editingGroup.FullRowColor = !m.editingGroup.FullRowColor
			m.storage.SaveWithGroups(m.instances, m.groups)
		} else if inst := m.getSelectedInstance(); inst != nil {
			inst.FullRowColor = !inst.FullRowColor
			m.storage.UpdateInstance(inst)
		}

	case "enter":
		filteredColors := m.getFilteredColorOptions()
		if m.colorCursor >= len(filteredColors) {
			return m, nil
		}
		selected := filteredColors[m.colorCursor]

		// Editing group color
		if m.editingGroup != nil {
			// Update preview with current selection
			if m.colorMode == 0 {
				m.previewFg = selected.Color
			} else {
				m.previewBg = selected.Color
			}

			// Save both colors
			if m.previewFg == "" || m.previewFg == "none" {
				m.editingGroup.Color = ""
			} else {
				m.editingGroup.Color = m.previewFg
			}
			if m.previewBg == "" || m.previewBg == "none" {
				m.editingGroup.BgColor = ""
			} else {
				m.editingGroup.BgColor = m.previewBg
			}

			m.storage.SaveWithGroups(m.instances, m.groups)
			m.editingGroup = nil
			m.state = stateList
			m.colorMode = 0
			return m, nil
		}

		// Editing session color
		if inst := m.getSelectedInstance(); inst != nil {
			// Update preview with current selection
			if m.colorMode == 0 {
				m.previewFg = selected.Color
			} else {
				m.previewBg = selected.Color
			}

			// Save both colors
			if m.previewFg == "" || m.previewFg == "none" {
				inst.Color = ""
			} else {
				inst.Color = m.previewFg
			}
			if m.previewBg == "" || m.previewBg == "none" {
				inst.BgColor = ""
			} else {
				inst.BgColor = m.previewBg
			}

			m.storage.UpdateInstance(inst)
			m.state = stateList
			m.colorMode = 0
		}
		return m, nil
	}

	return m, nil
}

// handleNewGroupKeys handles keyboard input in the new group dialog
func (m Model) handleNewGroupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "enter":
		if m.groupInput.Value() != "" {
			group, err := m.storage.AddGroup(m.groupInput.Value())
			if err != nil {
				m.err = err
			} else {
				m.groups = append(m.groups, group)
				m.buildVisibleItems()
			}
			m.state = stateList
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.groupInput, cmd = m.groupInput.Update(msg)
	return m, cmd
}

// handleRenameGroupKeys handles keyboard input in the rename group dialog
func (m Model) handleRenameGroupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "enter":
		if m.groupInput.Value() != "" {
			m.buildVisibleItems()
			if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
				item := m.visibleItems[m.cursor]
				if item.isGroup {
					if err := m.storage.RenameGroup(item.group.ID, m.groupInput.Value()); err != nil {
						m.err = err
					} else {
						item.group.Name = m.groupInput.Value()
					}
				}
			}
			m.state = stateList
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.groupInput, cmd = m.groupInput.Update(msg)
	return m, cmd
}

// handleSelectGroupKeys handles keyboard input in the group selection dialog
func (m Model) handleSelectGroupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxIdx := len(m.groups) // 0 = ungrouped, 1+ = groups

	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil

	case "up", "k":
		if m.groupCursor > 0 {
			m.groupCursor--
		}

	case "down", "j":
		if m.groupCursor < maxIdx {
			m.groupCursor++
		}

	case "enter":
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
			var groupID string
			if m.groupCursor > 0 && m.groupCursor <= len(m.groups) {
				groupID = m.groups[m.groupCursor-1].ID
			}
			inst.GroupID = groupID
			m.storage.UpdateInstance(inst)
			m.buildVisibleItems()
		}
		m.state = stateList
		return m, nil
	}

	return m, nil
}

// agentTypes defines the available agent types in order
var agentTypes = []session.AgentType{
	session.AgentClaude,
	session.AgentGemini,
	session.AgentAider,
	session.AgentCodex,
	session.AgentAmazonQ,
	session.AgentOpenCode,
	session.AgentCustom,
}

// handleSelectAgentKeys handles keyboard input in the agent selection dialog
func (m Model) handleSelectAgentKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear error on navigation
	if msg.String() == "up" || msg.String() == "k" || msg.String() == "down" || msg.String() == "j" {
		m.err = nil
	}

	switch msg.String() {
	case "esc":
		m.err = nil
		m.state = stateList
		return m, nil

	case "up", "k":
		if m.agentCursor > 0 {
			m.agentCursor--
		}

	case "down", "j":
		if m.agentCursor < len(agentTypes)-1 {
			m.agentCursor++
		}

	case "enter":
		m.pendingAgent = agentTypes[m.agentCursor]

		// If custom agent, ask for command first (can't check yet)
		if m.pendingAgent == session.AgentCustom {
			m.err = nil
			m.customCmdInput.SetValue("")
			m.customCmdInput.Focus()
			m.state = stateCustomCmd
			return m, textinput.Blink
		}

		// Check if the agent command exists
		config := session.AgentConfigs[m.pendingAgent]
		if _, err := exec.LookPath(config.Command); err != nil {
			m.err = fmt.Errorf("'%s' not found - is it installed?", config.Command)
			return m, nil
		}

		// Command exists, proceed to path input
		m.err = nil
		m.pathInput.SetValue("")
		m.pathInput.Focus()
		m.state = stateNewPath
		return m, textinput.Blink
	}

	return m, nil
}

// handleCustomCmdKeys handles keyboard input in the custom command dialog
func (m Model) handleCustomCmdKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.err = nil
		m.state = stateSelectAgent
		return m, nil

	case "enter":
		if m.customCmdInput.Value() != "" {
			// Check if the command exists
			parts := strings.Fields(m.customCmdInput.Value())
			if len(parts) > 0 {
				if _, err := exec.LookPath(parts[0]); err != nil {
					m.err = fmt.Errorf("'%s' not found - is it installed?", parts[0])
					return m, nil
				}
			}

			// Command exists, proceed to path input
			m.err = nil
			m.pathInput.SetValue("")
			m.pathInput.Focus()
			m.state = stateNewPath
			return m, textinput.Blink
		}
	}

	// Clear error when typing
	m.err = nil

	var cmd tea.Cmd
	m.customCmdInput, cmd = m.customCmdInput.Update(msg)
	return m, cmd
}

// handleErrorKeys handles keyboard input in the error overlay
func (m Model) handleErrorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key closes the error dialog
	m.err = nil
	// Return to appropriate state based on previous state
	switch m.previousState {
	case stateProjectSelect, stateNewProject, stateRenameProject, stateConfirmDeleteProject, stateConfirmImport:
		m.state = stateProjectSelect
	default:
		m.state = stateList
	}
	m.previousState = stateList
	return m, nil
}

// handleUpdateSuccessKeys handles keyboard input in the update success overlay
func (m Model) handleUpdateSuccessKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key closes the success dialog
	m.successMsg = ""
	// Return to appropriate state based on previous state
	switch m.previousState {
	case stateProjectSelect, stateNewProject, stateRenameProject, stateConfirmDeleteProject, stateConfirmImport:
		m.state = stateProjectSelect
	default:
		m.state = stateList
	}
	m.previousState = stateList
	return m, nil
}

// handleConfirmUpdateKeys handles keyboard input in the update confirmation overlay
func (m Model) handleConfirmUpdateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// If we already know the version, start download immediately
		if m.updateAvailable != "" {
			if updater.IsPackageManaged() {
				// Check if deb
				if _, err := os.Stat("/var/lib/dpkg/info/asmgr.list"); err == nil {
					m.state = stateDownloadingDeb
					return m, runDebDownload(m.updateAvailable)
				}
				// Otherwise rpm
				m.state = stateDownloadingRpm
				return m, runRpmDownload(m.updateAvailable)
			}
			m.state = stateUpdating
			return m, runUpdateCmd(m.updateAvailable)
		}
		// Otherwise check for updates first
		m.state = stateCheckingUpdate
		return m, checkForUpdateCmd()
	case "n", "N", "esc":
		// Cancel - go back to list
		m.state = stateList
		return m, nil
	}
	return m, nil
}

// handleProjectSelectKeys handles keyboard input in the project selection view
func (m Model) handleProjectSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Total items: projects + "New Project" + "Continue without project"
	totalItems := len(m.projects) + 2

	switch msg.String() {
	case "q", "ctrl+c":
		m.storage.UnlockProject()
		return m, tea.Quit

	case "up", "k":
		if m.projectCursor > 0 {
			m.projectCursor--
		}

	case "down", "j":
		if m.projectCursor < totalItems-1 {
			m.projectCursor++
		}

	case "enter":
		if m.projectCursor < len(m.projects) {
			// Selected a project
			project := m.projects[m.projectCursor]
			if err := m.switchToProject(project); err != nil {
				m.previousState = stateProjectSelect
				m.err = err
				m.state = stateError
				return m, nil
			}
			m.state = stateList
		} else if m.projectCursor == len(m.projects) {
			// "Continue without project"
			if err := m.switchToProject(nil); err != nil {
				m.previousState = stateProjectSelect
				m.err = err
				m.state = stateError
				return m, nil
			}
			m.state = stateList
		} else {
			// "New Project" (last option)
			m.projectInput.Reset()
			m.projectInput.Focus()
			m.state = stateNewProject
			return m, textinput.Blink
		}

	case "n":
		// Shortcut for new project
		m.projectInput.Reset()
		m.projectInput.Focus()
		m.state = stateNewProject
		return m, textinput.Blink

	case "e":
		// Rename project
		if m.projectCursor < len(m.projects) {
			project := m.projects[m.projectCursor]
			m.projectInput.SetValue(project.Name)
			m.projectInput.Focus()
			m.state = stateRenameProject
			return m, textinput.Blink
		}

	case "d":
		// Delete project
		if m.projectCursor < len(m.projects) {
			m.deleteProjectTarget = m.projects[m.projectCursor]
			m.state = stateConfirmDeleteProject
		}

	case "i":
		// Import sessions from default (no project) into selected project
		if m.projectCursor < len(m.projects) {
			// Check if there are sessions to import
			defaultCount := m.storage.GetProjectSessionCount("")
			if defaultCount == 0 {
				m.previousState = stateProjectSelect
				m.err = fmt.Errorf("no sessions to import (default is empty)")
				m.state = stateError
				return m, nil
			}
			m.importTarget = m.projects[m.projectCursor]
			m.state = stateConfirmImport
		} else {
			m.previousState = stateProjectSelect
			m.err = fmt.Errorf("select a project first to import sessions into")
			m.state = stateError
		}
	}

	return m, nil
}

// handleNewProjectKeys handles keyboard input when creating a new project
func (m Model) handleNewProjectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateProjectSelect
		return m, nil

	case "enter":
		name := strings.TrimSpace(m.projectInput.Value())
		if name == "" {
			m.previousState = stateNewProject
			m.err = fmt.Errorf("project name cannot be empty")
			m.state = stateError
			return m, nil
		}

		project, err := m.storage.AddProject(name)
		if err != nil {
			m.previousState = stateNewProject
			m.err = err
			m.state = stateError
			return m, nil
		}

		// Reload projects list
		projectsData, _ := m.storage.LoadProjects()
		m.projects = projectsData.Projects

		// Switch to the new project
		if err := m.switchToProject(project); err != nil {
			m.previousState = stateProjectSelect
			m.err = err
			m.state = stateError
			return m, nil
		}

		m.state = stateList
		return m, nil
	}

	var cmd tea.Cmd
	m.projectInput, cmd = m.projectInput.Update(msg)
	return m, cmd
}

// handleRenameProjectKeys handles keyboard input when renaming a project
func (m Model) handleRenameProjectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateProjectSelect
		return m, nil

	case "enter":
		name := strings.TrimSpace(m.projectInput.Value())
		if name == "" {
			m.previousState = stateRenameProject
			m.err = fmt.Errorf("project name cannot be empty")
			m.state = stateError
			return m, nil
		}

		if m.projectCursor < len(m.projects) {
			project := m.projects[m.projectCursor]
			if err := m.storage.RenameProject(project.ID, name); err != nil {
				m.previousState = stateRenameProject
				m.err = err
				m.state = stateError
				return m, nil
			}
			project.Name = name
		}

		m.state = stateProjectSelect
		return m, nil
	}

	var cmd tea.Cmd
	m.projectInput, cmd = m.projectInput.Update(msg)
	return m, cmd
}

// handleConfirmDeleteProjectKeys handles keyboard input in the project deletion confirmation
func (m Model) handleConfirmDeleteProjectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.deleteProjectTarget != nil {
			if err := m.storage.RemoveProject(m.deleteProjectTarget.ID); err != nil {
				m.previousState = stateProjectSelect
				m.err = err
				m.state = stateError
				return m, nil
			}

			// Reload projects list
			projectsData, _ := m.storage.LoadProjects()
			m.projects = projectsData.Projects

			// Adjust cursor if needed
			if m.projectCursor >= len(m.projects) && m.projectCursor > 0 {
				m.projectCursor--
			}
		}
		m.deleteProjectTarget = nil
		m.state = stateProjectSelect

	case "n", "N", "esc":
		m.deleteProjectTarget = nil
		m.state = stateProjectSelect
	}

	return m, nil
}

// handleConfirmImportKeys handles keyboard input in the import confirmation
func (m Model) handleConfirmImportKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.importTarget != nil {
			count, err := m.storage.ImportDefaultSessions(m.importTarget.ID)
			m.previousState = stateProjectSelect
			if err != nil {
				m.err = err
				m.state = stateError
			} else {
				m.err = fmt.Errorf("successfully imported %d sessions into '%s'", count, m.importTarget.Name)
				m.state = stateError // Use dialog for success message too
			}
		}
		m.importTarget = nil

	case "n", "N", "esc":
		m.importTarget = nil
		m.state = stateProjectSelect
	}

	return m, nil
}
