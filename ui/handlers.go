package ui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/izll/agent-session-manager/session"
)

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
			m.state = stateError
			return nil
		}
		if err := inst.Start(); err != nil {
			m.err = err
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
	sessions, err := session.ListClaudeSessions(inst.Path)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		return fmt.Errorf("no previous Claude sessions found for this path")
	}
	m.claudeSessions = sessions
	m.sessionCursor = 0
	m.state = stateSelectClaudeSession
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
			m.state = stateError
			return
		}
		if err := inst.Start(); err != nil {
			m.err = err
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
	}
}

// handleListKeys handles keyboard input in the main list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear error on any key press
	m.err = nil

	switch msg.String() {
	case "q", "ctrl+c":
		m.saveSettings() // Save cursor position on quit
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
				m.resizeSelectedPane()
			}
		} else if m.cursor > 0 {
			m.cursor--
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
				m.resizeSelectedPane()
			}
		} else if m.cursor < len(m.instances)-1 {
			m.cursor++
			m.resizeSelectedPane()
		}

	case "shift+up", "K":
		m.handleMoveSessionUp()

	case "shift+down", "J":
		m.handleMoveSessionDown()

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
		// Resume only works for Claude agent
		if inst := m.getSelectedInstance(); inst != nil {
			config := inst.GetAgentConfig()
			if !config.SupportsResume {
				m.err = fmt.Errorf("resume not supported for %s agent", inst.Agent)
				return m, nil
			}
		}
		if err := m.handleResumeSession(); err != nil {
			m.err = err
		}

	case "s":
		m.handleStartSession()

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

	case "y":
		m.autoYes = !m.autoYes

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
		// Trigger update if available
		if m.updateAvailable != "" {
			m.state = stateUpdating
			return m, runUpdateCmd(m.updateAvailable)
		}

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
		m.state = stateList
		return m, nil
	case "enter":
		if m.nameInput.Value() != "" {
			// Create instance with the entered name and stored path
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

			// Check for existing Claude sessions (only for Claude agent)
			if m.pendingAgent == session.AgentClaude {
				sessions, err := session.ListClaudeSessions(inst.Path)
				if err != nil {
					// Non-fatal: just continue without session selection
					sessions = nil
				}
				if len(sessions) > 0 {
					m.pendingInstance = inst
					m.claudeSessions = sessions
					m.sessionCursor = 0
					m.state = stateSelectClaudeSession
					return m, nil
				}
			}

			// No existing sessions or non-Claude agent, just create new
			if err := m.storage.AddInstance(inst); err != nil {
				m.err = err
				m.state = stateList
				return m, nil
			}

			// Auto-start the new instance
			if err := inst.Start(); err != nil {
				m.err = err
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
	maxIdx := len(m.claudeSessions) // max index (0 = new session, 1+ = existing sessions)

	switch msg.String() {
	case "esc":
		m.claudeSessions = nil
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
		if m.sessionCursor > 0 && m.sessionCursor <= len(m.claudeSessions) {
			// Selected an existing session
			resumeID = m.claudeSessions[m.sessionCursor-1].SessionID
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
				m.claudeSessions = nil
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

		m.claudeSessions = nil
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
	case "esc", "q", "?":
		m.state = stateList
	}
	return m, nil
}

// handlePromptKeys handles keyboard input in the prompt dialog
func (m Model) handlePromptKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "enter":
		if m.promptInput.Value() != "" {
			if inst := m.getSelectedInstance(); inst != nil && inst.Status == session.StatusRunning {
				// Send the text followed by Enter
				text := m.promptInput.Value()
				if err := inst.SendKeys(text); err != nil {
					m.err = err
				} else {
					// Send Enter key
					inst.SendKeys("Enter")
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
	switch msg.String() {
	case "esc", "enter", "q":
		m.err = nil
		m.state = stateList
	}
	return m, nil
}
