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
				m.previousState = stateList
				m.state = stateError
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
				m.previousState = stateList
				m.state = stateError
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
				m.previousState = stateList
				m.state = stateError
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
				m.previousState = stateList
				m.state = stateError
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
		// Otherwise check for updates first (force check, ignore 24h timer)
		m.state = stateCheckingUpdate
		return m, forceCheckForUpdateCmd()
	case "n", "N", "esc":
		// Cancel - go back to list
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

// handleNotesKeys handles keyboard input in the notes editor dialog
func (m Model) handleNotesKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel without saving
		m.state = stateList
		return m, nil

	case "ctrl+s":
		// Save notes
		if inst := m.getSelectedInstance(); inst != nil {
			inst.Notes = m.notesInput.Value()
			m.storage.UpdateInstance(inst)
		}
		m.state = stateList
		return m, nil

	case "ctrl+d":
		// Clear notes
		m.notesInput.SetValue("")
		return m, nil
	}

	var cmd tea.Cmd
	m.notesInput, cmd = m.notesInput.Update(msg)
	return m, cmd
}
