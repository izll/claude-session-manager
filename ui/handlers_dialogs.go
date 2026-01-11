package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
			// Resuming existing instance - apply to specific window
			if m.resumeWindowIndex == 0 {
				// Main window
				inst.ResumeSessionID = resumeID
				if inst.Status == session.StatusRunning {
					// Respawn just the main window with new resume ID
					inst.RespawnWindowWithResume(0, resumeID)
				} else {
					if err := inst.StartWithResume(resumeID); err != nil {
						m.err = err
					}
				}
			} else {
				// Followed window
				for idx, fw := range inst.FollowedWindows {
					if fw.Index == m.resumeWindowIndex {
						inst.FollowedWindows[idx].ResumeSessionID = resumeID
						if inst.Status == session.StatusRunning {
							// Respawn just this window with new resume ID
							inst.RespawnWindowWithResume(fw.Index, resumeID)
						}
						break
					}
				}
			}
			m.storage.UpdateInstance(inst)
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

// handleConfirmStopKeys handles keyboard input in the stop confirmation dialog
func (m Model) handleConfirmStopKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.stopTarget != nil {
			m.stopTarget.Stop()
			m.storage.UpdateInstance(m.stopTarget)
		}
		m.stopTarget = nil
		m.state = stateList
	case "n", "N", "esc":
		m.stopTarget = nil
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
	// Get actual line count from help content
	_, totalLines := buildHelpContent(m.width)
	maxLines := m.height - 3
	if maxLines < 10 {
		maxLines = 10
	}
	maxScroll := totalLines - maxLines
	if maxScroll < 0 {
		maxScroll = 0
	}

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
		if m.helpScroll < maxScroll {
			m.helpScroll++
		}
	case "home":
		m.helpScroll = 0
	case "end":
		m.helpScroll = maxScroll
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
			return m, nil
		}
	case "ctrl+s", "ctrl+enter":
		// Send message with Ctrl+S or Ctrl+Enter
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
	session.AgentCursor,
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
		// Return to appropriate state based on context
		if m.newTabIsAgent && m.newTabAgent == session.AgentCustom {
			m.state = stateNewTabAgent
		} else {
			m.state = stateSelectAgent
		}
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

			// Command exists, proceed based on context
			m.err = nil
			if m.newTabIsAgent && m.newTabAgent == session.AgentCustom {
				// Creating agent tab with custom command - go to name input
				m.nameInput.SetValue("")
				m.nameInput.Focus()
				m.state = stateNewTab
				return m, textinput.Blink
			}

			// Creating new session - proceed to path input
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
		// Save notes to session or tab
		if inst := m.getSelectedInstance(); inst != nil {
			notes := m.notesInput.Value()
			if m.notesWindowIndex == 0 {
				// Main session notes
				inst.Notes = notes
			} else {
				// Tab notes - find and update the FollowedWindow
				for i := range inst.FollowedWindows {
					if inst.FollowedWindows[i].Index == m.notesWindowIndex {
						inst.FollowedWindows[i].Notes = notes
						break
					}
				}
			}
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

// handleNewTabChoiceKeys handles keyboard input in the tab type choice dialog
func (m Model) handleNewTabChoiceKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "a", "A":
		// Agent tab - first select which agent
		m.newTabIsAgent = true
		m.newTabAgentCursor = 0
		m.err = nil
		m.state = stateNewTabAgent
		return m, nil
	case "t", "T":
		// Terminal tab
		m.newTabIsAgent = false
		m.nameInput.SetValue("")
		m.nameInput.Focus()
		m.state = stateNewTab
		return m, textinput.Blink
	}
	return m, nil
}

// handleNewTabKeys handles keyboard input in the new tab dialog
func (m Model) handleNewTabKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "enter":
		if inst := m.getSelectedInstance(); inst != nil {
			if inst.Status == session.StatusRunning {
				name := m.nameInput.Value()
				if name == "" {
					if m.newTabIsAgent {
						name = "agent"
					} else {
						name = "shell"
					}
				}

				if m.newTabIsAgent {
					// Create agent window (tracked) with selected agent type
					customCmd := ""
					if m.newTabAgent == session.AgentCustom {
						customCmd = m.customCmdInput.Value()
					}
					inst.NewAgentWindow(name, m.newTabAgent, customCmd)
				} else {
					// Create terminal window (tracked for restore)
					inst.NewWindowWithName(name)
				}
				// Refresh status bar to show tab list
				configureTmuxStatusBar(inst.TmuxSessionName(), inst.Name, inst.Color, inst.BgColor, inst.AutoYes)
				m.storage.UpdateInstance(inst) // Save followed windows
			}
		}
		m.state = stateList
		return m, nil
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

// handleRenameTabKeys handles keyboard input in the rename tab dialog
func (m Model) handleRenameTabKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateList
		return m, nil
	case "enter":
		if m.nameInput.Value() != "" {
			if inst := m.getSelectedInstance(); inst != nil {
				if inst.Status == session.StatusRunning {
					inst.RenameCurrentWindow(m.nameInput.Value())
				}
			}
		}
		m.state = stateList
		return m, nil
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

// handleNewTabAgentKeys handles keyboard input in the agent selection dialog for new tab
func (m Model) handleNewTabAgentKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if m.newTabAgentCursor > 0 {
			m.newTabAgentCursor--
		}

	case "down", "j":
		if m.newTabAgentCursor < len(agentTypes)-1 {
			m.newTabAgentCursor++
		}

	case "enter":
		m.newTabAgent = agentTypes[m.newTabAgentCursor]

		// If custom agent, ask for command first
		if m.newTabAgent == session.AgentCustom {
			m.err = nil
			m.customCmdInput.SetValue("")
			m.customCmdInput.Focus()
			m.state = stateCustomCmd
			// Store that we're coming from tab creation
			m.newTabIsAgent = true
			return m, textinput.Blink
		}

		// Check if the agent command exists
		config := session.AgentConfigs[m.newTabAgent]
		if _, err := exec.LookPath(config.Command); err != nil {
			m.err = fmt.Errorf("'%s' not found - is it installed?", config.Command)
			return m, nil
		}

		// Command exists, proceed to name input
		m.err = nil
		m.nameInput.SetValue("")
		m.nameInput.Focus()
		m.state = stateNewTab
		return m, textinput.Blink
	}

	return m, nil
}

// handleDeleteChoiceKeys handles keyboard input in the delete choice dialog
func (m Model) handleDeleteChoiceKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s", "S":
		// Delete session
		m.state = stateConfirmDelete
		return m, nil
	case "t", "T":
		// Delete tab - check if on main window (can't delete)
		if m.deleteTarget != nil {
			windows := m.deleteTarget.GetWindowList()
			for _, w := range windows {
				if w.Active {
					if w.Index == 0 {
						m.err = fmt.Errorf("cannot close main agent tab")
						m.previousState = stateList
						m.state = stateError
						m.deleteTarget = nil
						return m, nil
					}
					// Not main tab, confirm deletion
					m.state = stateConfirmDeleteTab
					return m, nil
				}
			}
		}
		m.deleteTarget = nil
		m.state = stateList
	case "esc":
		m.deleteTarget = nil
		m.state = stateList
	}
	return m, nil
}

// handleConfirmDeleteTabKeys handles keyboard input in the tab deletion confirmation dialog
func (m Model) handleConfirmDeleteTabKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.deleteTarget != nil {
			windows := m.deleteTarget.GetWindowList()
			for _, w := range windows {
				if w.Active && w.Index != 0 {
					if err := m.deleteTarget.CloseWindow(w.Index); err != nil {
						m.err = err
						m.previousState = stateList
						m.state = stateError
					} else {
						// Refresh status bar (may hide tabs if only 1 window left)
						configureTmuxStatusBar(m.deleteTarget.TmuxSessionName(), m.deleteTarget.Name, m.deleteTarget.Color, m.deleteTarget.BgColor, m.deleteTarget.AutoYes)
						m.storage.UpdateInstance(m.deleteTarget)
					}
					break
				}
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

// handleStopChoiceKeys handles keyboard input in the stop choice dialog
func (m Model) handleStopChoiceKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s", "S":
		// Stop session
		m.state = stateConfirmStop
		return m, nil
	case "t", "T":
		// Stop tab - confirm first
		if m.stopTarget != nil {
			m.state = stateConfirmStopTab
			return m, nil
		}
		m.stopTarget = nil
		m.state = stateList
	case "esc":
		m.stopTarget = nil
		m.state = stateList
	}
	return m, nil
}

// handleConfirmStopTabKeys handles keyboard input in the tab stop confirmation dialog
func (m Model) handleConfirmStopTabKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.stopTarget != nil {
			windows := m.stopTarget.GetWindowList()
			for _, w := range windows {
				if w.Active {
					if err := m.stopTarget.StopWindow(w.Index); err != nil {
						m.err = err
						m.previousState = stateList
						m.state = stateError
					}
					break
				}
			}
		}
		m.stopTarget = nil
		m.state = stateList
	case "n", "N", "esc":
		m.stopTarget = nil
		m.state = stateList
	}
	return m, nil
}

// handleConfirmYoloKeys handles keyboard input in the YOLO mode confirmation dialog
func (m Model) handleConfirmYoloKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.yoloTarget != nil {
			inst := m.yoloTarget
			wasRunning := inst.Status == session.StatusRunning

			// Toggle YOLO for the correct window
			if m.yoloWindowIndex == 0 {
				inst.AutoYes = m.yoloNewState
			} else {
				for idx, fw := range inst.FollowedWindows {
					if fw.Index == m.yoloWindowIndex {
						inst.FollowedWindows[idx].AutoYes = m.yoloNewState
						break
					}
				}
			}

			m.storage.UpdateInstance(inst)

			// If running, respawn the window with new flag
			if wasRunning {
				if m.yoloWindowIndex == 0 {
					// Main window - restart session
					inst.Stop()
					if err := inst.Start(); err != nil {
						m.err = fmt.Errorf("failed to restart session: %w", err)
						m.previousState = stateList
						m.state = stateError
						m.yoloTarget = nil
						return m, nil
					}
					m.storage.UpdateInstance(inst)
				} else {
					// Tab window - respawn just that window
					inst.RespawnWindow(m.yoloWindowIndex)
				}
				// Refresh tmux status bar
				RefreshTmuxStatusBarFull(inst.TmuxSessionName(), inst.Name, inst.Color, inst.BgColor, inst)
			}
		}
		m.yoloTarget = nil
		m.state = stateList
	case "n", "N", "esc":
		m.yoloTarget = nil
		m.state = stateList
	}
	return m, nil
}

// handleSearchKeys handles keyboard input in the search mode
func (m Model) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel search and clear filter
		m.searchQuery = ""
		m.searchActive = false
		m.state = stateList
		return m, nil

	case "enter", "down", "up":
		// Accept search and navigate
		query := strings.TrimSpace(m.searchInput.Value())
		if query != "" {
			m.searchQuery = strings.ToLower(query)
			m.searchActive = true
			m.cursor = 0
		} else {
			m.searchQuery = ""
			m.searchActive = false
		}
		m.state = stateList
		return m, nil
	}

	// Update input - pass as tea.Msg to ensure proper handling
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.searchInput.Focus() // Ensure input stays focused

	// Live filter as user types
	query := strings.TrimSpace(m.searchInput.Value())
	if query != "" {
		m.searchQuery = strings.ToLower(query)
		m.searchActive = true
	} else {
		m.searchQuery = ""
		m.searchActive = false
	}

	return m, cmd
}

// handleGlobalSearchKeys handles keyboard input in the global search mode
func (m Model) handleGlobalSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Close global search
		m.globalSearchResults = nil
		m.globalSearchCursor = 0
		m.globalSearchExpanded = -1
		m.globalSearchConversation = nil
		m.globalSearchScroll = 0
		m.globalSearchDebounceActive = false
		m.globalSearchConvLoading = false
		m.state = stateList
		return m, nil

	case "ctrl+r":
		// Reload history index
		m.globalSearchResults = nil
		m.globalSearchCursor = 0
		m.globalSearchConversation = nil
		m.globalSearchScroll = 0
		m.state = stateGlobalSearchLoading
		// Force reload by resetting the index
		m.historyIndex = session.NewHistoryIndex()
		return m, m.loadHistoryCmd()

	case "up":
		// Navigate up in results
		if m.globalSearchCursor > 0 {
			m.globalSearchCursor--
			m.globalSearchExpanded = -1
			return m, m.loadConversationAsync()
		}
		return m, nil

	case "down":
		// Navigate down in results
		if m.globalSearchCursor < len(m.globalSearchResults)-1 {
			m.globalSearchCursor++
			m.globalSearchExpanded = -1
			return m, m.loadConversationAsync()
		}
		return m, nil

	case "[", "alt+up":
		// Scroll preview up (3 lines)
		if m.globalSearchScroll > 0 {
			m.globalSearchScroll -= 3
			if m.globalSearchScroll < 0 {
				m.globalSearchScroll = 0
			}
		}
		return m, nil

	case "]", "alt+down":
		// Scroll preview down (3 lines)
		m.globalSearchScroll += 3
		return m, nil

	case "pgup", "alt+pgup":
		// Scroll preview up half page
		if m.globalSearchScroll > 0 {
			m.globalSearchScroll -= 15
			if m.globalSearchScroll < 0 {
				m.globalSearchScroll = 0
			}
		}
		return m, nil

	case "pgdown", "alt+pgdown":
		// Scroll preview down half page
		m.globalSearchScroll += 15
		return m, nil

	case "enter":
		// Handle selected result - jump directly if match exists
		if len(m.globalSearchResults) > 0 && m.globalSearchCursor < len(m.globalSearchResults) {
			if m.globalSearchMatchedSession != nil {
				// Match found - jump directly to session
				inst := m.globalSearchMatchedSession
				tabIndex := m.globalSearchMatchedTabIndex

				// Find session index in list
				if len(m.groups) > 0 {
					// Grouped mode: search in visibleItems
					m.buildVisibleItems()
					for idx, item := range m.visibleItems {
						if !item.isGroup && item.instance != nil && item.instance.ID == inst.ID {
							m.cursor = idx
							break
						}
					}
				} else {
					// Non-grouped mode: search in instances
					for i, s := range m.instances {
						if s.ID == inst.ID {
							m.cursor = i
							break
						}
					}
				}

				// Switch to matched tab if needed
				if tabIndex >= 0 && tabIndex < len(inst.FollowedWindows) && inst.Status == session.StatusRunning {
					windowIndex := inst.FollowedWindows[tabIndex].Index
					inst.SelectWindow(windowIndex)
				}

				// Close global search and return to list
				m.globalSearchResults = nil
				m.globalSearchCursor = 0
				m.globalSearchExpanded = -1
				m.globalSearchConversation = nil
				m.globalSearchScroll = 0
				m.globalSearchMatchedSession = nil
				m.globalSearchMatchedTabIndex = -1
				m.state = stateList
				return m, nil
			}

			// No matching session found - show action dialog
			entry := m.globalSearchResults[m.globalSearchCursor]
			m.globalSearchSelectedEntry = &entry
			m.globalSearchActionCursor = 0
			m.state = stateGlobalSearchAction
		}
		return m, nil

	case "home":
		// Jump to first result
		m.globalSearchCursor = 0
		m.globalSearchExpanded = -1
		return m, m.loadConversationAsync()

	case "end":
		// Jump to last result
		if len(m.globalSearchResults) > 0 {
			m.globalSearchCursor = len(m.globalSearchResults) - 1
		}
		m.globalSearchExpanded = -1
		return m, m.loadConversationAsync()
	}

	// Update input field
	var cmd tea.Cmd
	m.globalSearchInput, cmd = m.globalSearchInput.Update(msg)
	m.globalSearchInput.Focus() // Ensure input stays focused

	// Check if query changed - use debounce
	query := strings.TrimSpace(m.globalSearchInput.Value())
	if query != m.globalSearchPendingQuery {
		m.globalSearchPendingQuery = query
		// Start debounce timer (200ms delay)
		if !m.globalSearchDebounceActive {
			m.globalSearchDebounceActive = true
			debounceCmd := tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
				return globalSearchDebounceMsg{}
			})
			return m, tea.Batch(cmd, debounceCmd)
		}
	}

	return m, cmd
}

// findBestMatchForEntry finds the best matching session/tab for a history entry
// Returns the matched session and tab index (-1 for main session, >=0 for tab)
func (m *Model) findBestMatchForEntry(entry session.HistoryEntry) (*session.Instance, int) {
	// Special handling for Terminal entries - find terminal tabs directly
	if entry.Agent == session.AgentTerminal {
		for _, inst := range m.instances {
			if (entry.SessionID != "" && inst.ResumeSessionID == entry.SessionID) ||
				(entry.Path != "" && inst.Path == entry.Path) {
				for i, tab := range inst.FollowedWindows {
					if tab.Agent == session.AgentTerminal {
						return inst, i
					}
				}
			}
		}
		return nil, -1
	}

	// Priority 1: Match by SessionID (most specific)
	if entry.SessionID != "" {
		for _, inst := range m.instances {
			if inst.ResumeSessionID == entry.SessionID && inst.Agent == entry.Agent {
				return inst, -1
			}
			for i, tab := range inst.FollowedWindows {
				if tab.ResumeSessionID == entry.SessionID && tab.Agent == entry.Agent {
					return inst, i
				}
			}
		}
	}

	// Priority 2: Match by path + agent (good specificity)
	if entry.Path != "" {
		for _, inst := range m.instances {
			if inst.Path == entry.Path && inst.Agent == entry.Agent {
				return inst, -1
			}
			for i, tab := range inst.FollowedWindows {
				if inst.Path == entry.Path && tab.Agent == entry.Agent {
					return inst, i
				}
			}
		}
	}

	return nil, -1
}

// loadConversationAsync starts async loading of conversation for current cursor
func (m *Model) loadConversationAsync() tea.Cmd {
	if len(m.globalSearchResults) == 0 || m.globalSearchCursor >= len(m.globalSearchResults) {
		m.globalSearchMatchedSession = nil
		m.globalSearchMatchedTabIndex = -1
		return nil
	}

	entry := m.globalSearchResults[m.globalSearchCursor]

	// Find matching session for this entry
	m.globalSearchMatchedSession, m.globalSearchMatchedTabIndex = m.findBestMatchForEntry(entry)

	if entry.SessionFile == "" {
		m.globalSearchConversation = nil
		m.globalSearchConvLoading = false
		m.globalSearchScroll = 0
		return nil
	}

	// Mark as loading
	m.globalSearchConvLoading = true
	m.globalSearchConversation = nil
	m.globalSearchScroll = 0
	cursorPos := m.globalSearchCursor

	// Load in background
	return func() tea.Msg {
		conv, _ := entry.LoadConversation()
		return globalSearchConvLoadedMsg{
			conversation: conv,
			cursorPos:    cursorPos,
		}
	}
}

// handleGlobalSearchDebounce handles the debounce timer firing
func (m Model) handleGlobalSearchDebounce() (tea.Model, tea.Cmd) {
	m.globalSearchDebounceActive = false

	// Check if we're still in global search state
	if m.state != stateGlobalSearch {
		return m, nil
	}

	// Perform the search with the pending query
	query := m.globalSearchPendingQuery
	if query != m.globalSearchLastQuery {
		m.globalSearchLastQuery = query
		m.globalSearchResults = m.historyIndex.Search(query)
		m.globalSearchCursor = 0
		m.globalSearchExpanded = -1
		m.globalSearchConversation = nil
		m.globalSearchScroll = 0

		// Load conversation for first result
		return m, m.loadConversationAsync()
	}

	return m, nil
}

// handleGlobalSearchConvLoaded handles when conversation finishes loading
func (m Model) handleGlobalSearchConvLoaded(msg globalSearchConvLoadedMsg) (tea.Model, tea.Cmd) {
	// Only apply if cursor hasn't moved
	if m.globalSearchCursor == msg.cursorPos {
		m.globalSearchConversation = msg.conversation
		m.globalSearchConvLoading = false

		// Auto-scroll to first match in conversation
		query := strings.TrimSpace(m.globalSearchInput.Value())
		if query != "" && len(msg.conversation) > 0 {
			m.globalSearchScroll = m.findFirstMatchLine(msg.conversation, query)
		}
	}
	return m, nil
}

// findFirstMatchLine finds the line number of the first match in conversation
// This must match how formatConversationLines counts lines (with text wrapping)
func (m Model) findFirstMatchLine(messages []session.ConversationMessage, query string) int {
	lowerQuery := strings.ToLower(query)
	lineNum := 0

	// Use actual preview width (same calculation as in view)
	// Preview width = total width - list pane width - borders
	previewWidth := m.width - ListPaneWidth - 4
	if previewWidth < 40 {
		previewWidth = 40
	}
	// formatConversationLines uses width-2 for content, minus 4 for indent
	wrapWidth := previewWidth - 6

	for _, msg := range messages {
		// Role header line (👤 User or 🤖 Assistant)
		lineNum++

		// Message content - need to account for text wrapping like formatConversationLines does
		content := msg.Content
		// Replace tabs like wrapText does
		content = strings.ReplaceAll(content, "\t", "  ")
		paragraphs := strings.Split(content, "\n")

		for _, para := range paragraphs {
			para = strings.TrimSpace(para)
			if para == "" {
				lineNum++
				continue
			}

			// Check if this paragraph contains the match
			if strings.Contains(strings.ToLower(para), lowerQuery) {
				// Found match - scroll with offset for context
				result := lineNum - 7
				if result < 0 {
					result = 0
				}
				return result
			}

			// Count wrapped lines (approximate)
			words := strings.Fields(para)
			currentLineLen := 0
			wrappedLines := 1
			for _, word := range words {
				if currentLineLen+len(word)+1 > wrapWidth && currentLineLen > 0 {
					wrappedLines++
					currentLineLen = len(word)
				} else {
					if currentLineLen > 0 {
						currentLineLen++
					}
					currentLineLen += len(word)
				}
			}
			lineNum += wrappedLines
		}

		// Empty line between messages
		lineNum++
	}

	return 0
}

// handleGlobalSearchConfirmJumpKeys handles keyboard input in the confirm jump dialog
func (m Model) handleGlobalSearchConfirmJumpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Go back to global search
		m.globalSearchMatchedSession = nil
		m.globalSearchSelectedEntry = nil
		m.state = stateGlobalSearch
		return m, nil

	case "enter":
		// Jump to the matched session
		if m.globalSearchMatchedSession == nil {
			m.state = stateGlobalSearch
			return m, nil
		}

		inst := m.globalSearchMatchedSession
		tabIndex := m.globalSearchMatchedTabIndex

		// Clear global search state
		m.globalSearchResults = nil
		m.globalSearchCursor = 0
		m.globalSearchExpanded = -1
		m.globalSearchConversation = nil
		m.globalSearchScroll = 0
		m.globalSearchDebounceActive = false
		m.globalSearchConvLoading = false
		m.globalSearchSelectedEntry = nil
		m.globalSearchMatchedSession = nil
		m.globalSearchMatchedTabIndex = -1
		m.state = stateList

		// Update cursor to point to this session
		if len(m.groups) > 0 {
			m.buildVisibleItems()
			for idx, item := range m.visibleItems {
				if !item.isGroup && item.instance != nil && item.instance.ID == inst.ID {
					m.cursor = idx
					break
				}
			}
		} else {
			for i, existingInst := range m.instances {
				if existingInst.ID == inst.ID {
					m.cursor = i
					break
				}
			}
		}

		// Switch to matched tab if needed
		if tabIndex >= 0 && tabIndex < len(inst.FollowedWindows) && inst.Status == session.StatusRunning {
			// Get actual tmux window index from FollowedWindows
			windowIndex := inst.FollowedWindows[tabIndex].Index
			inst.SelectWindow(windowIndex)
		}

		return m, nil
	}

	return m, nil
}

// handleGlobalSearchSelectMatchKeys handles keyboard input in the match selection dialog
func (m Model) handleGlobalSearchSelectMatchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxCursor := len(m.globalSearchMatches) - 1
	if maxCursor < 0 {
		maxCursor = 0
	}

	switch msg.String() {
	case "esc":
		// Go back to global search
		m.globalSearchMatches = nil
		m.globalSearchMatchCursor = 0
		m.globalSearchSelectedEntry = nil
		m.state = stateGlobalSearch
		return m, nil

	case "up", "k":
		if m.globalSearchMatchCursor > 0 {
			m.globalSearchMatchCursor--
		}
		return m, nil

	case "down", "j":
		if m.globalSearchMatchCursor < maxCursor {
			m.globalSearchMatchCursor++
		}
		return m, nil

	case "enter":
		if len(m.globalSearchMatches) == 0 {
			m.state = stateGlobalSearch
			return m, nil
		}

		selected := m.globalSearchMatches[m.globalSearchMatchCursor]
		inst := selected.Session
		tabIndex := selected.TabIndex

		// Clear global search state
		m.globalSearchResults = nil
		m.globalSearchCursor = 0
		m.globalSearchExpanded = -1
		m.globalSearchConversation = nil
		m.globalSearchScroll = 0
		m.globalSearchDebounceActive = false
		m.globalSearchConvLoading = false
		m.globalSearchSelectedEntry = nil
		m.globalSearchMatches = nil
		m.globalSearchMatchCursor = 0
		m.state = stateList

		// Update cursor to point to this session
		if len(m.groups) > 0 {
			m.buildVisibleItems()
			for idx, item := range m.visibleItems {
				if !item.isGroup && item.instance != nil && item.instance.ID == inst.ID {
					m.cursor = idx
					break
				}
			}
		} else {
			for i, existingInst := range m.instances {
				if existingInst.ID == inst.ID {
					m.cursor = i
					break
				}
			}
		}

		// Switch to matched tab if needed
		if tabIndex >= 0 && tabIndex < len(inst.FollowedWindows) && inst.Status == session.StatusRunning {
			// Get actual tmux window index from FollowedWindows
			windowIndex := inst.FollowedWindows[tabIndex].Index
			inst.SelectWindow(windowIndex)
		}

		return m, nil
	}

	return m, nil
}

// handleGlobalSearchNewNameKeys handles keyboard input in the new session name dialog
func (m Model) handleGlobalSearchNewNameKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Go back to action selection
		m.state = stateGlobalSearchAction
		return m, nil

	case "enter":
		// Create session with entered name
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			return m, nil
		}

		if m.globalSearchSelectedEntry == nil {
			m.state = stateList
			return m, nil
		}

		return m.createSessionFromSearchEntry(m.globalSearchSelectedEntry, "", name)
	}

	// Handle text input
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

// handleGlobalSearchActionKeys handles keyboard input in the global search action dialog
func (m Model) handleGlobalSearchActionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	maxCursor := 2 // 0=new session, 1=to group, 2=as tab

	switch msg.String() {
	case "esc":
		// Go back to global search
		m.globalSearchSelectedEntry = nil
		m.state = stateGlobalSearch
		return m, nil

	case "up", "k":
		if m.globalSearchActionCursor > 0 {
			m.globalSearchActionCursor--
		}
		return m, nil

	case "down", "j":
		if m.globalSearchActionCursor < maxCursor {
			m.globalSearchActionCursor++
		}
		return m, nil

	case "1":
		m.globalSearchActionCursor = 0
		return m, nil

	case "2":
		m.globalSearchActionCursor = 1
		return m, nil

	case "3":
		m.globalSearchActionCursor = 2
		return m, nil

	case "enter":
		if m.globalSearchSelectedEntry == nil {
			m.state = stateList
			return m, nil
		}

		entry := m.globalSearchSelectedEntry

		switch m.globalSearchActionCursor {
		case 0:
			// New session - ask for name first
			m.nameInput.Reset()
			// Pre-fill with snippet
			suggestedName := entry.Snippet
			if len(suggestedName) > 30 {
				suggestedName = suggestedName[:30]
			}
			suggestedName = strings.ReplaceAll(suggestedName, "\n", " ")
			suggestedName = strings.ReplaceAll(suggestedName, "\t", " ")
			m.nameInput.SetValue(suggestedName)
			m.nameInput.CursorEnd()
			m.nameInput.Focus()
			m.state = stateGlobalSearchNewName
			return m, nil

		case 1:
			// Add to group - transition to group selector
			m.groupCursor = 0
			if len(m.groups) == 0 {
				// No groups exist - create as ungrouped
				return m.createSessionFromSearchEntry(entry, "", "")
			}
			m.state = stateSelectGroup
			// Store that we're coming from global search
			m.pendingGroupID = "__from_global_search__"
			return m, nil

		case 2:
			// Add as new tab - need to select session
			if len(m.instances) == 0 {
				// No sessions exist - create new session instead
				return m.createSessionFromSearchEntry(entry, "", "")
			}
			// Use session cursor for selection, transition to a custom selector
			m.sessionCursor = 0
			return m.addSearchEntryAsTab(entry)
		}
	}

	return m, nil
}

// createSessionFromSearchEntry creates a new session from a global search entry
func (m *Model) createSessionFromSearchEntry(entry *session.HistoryEntry, groupID string, customName string) (Model, tea.Cmd) {
	// Only Claude entries can be resumed
	if entry.Agent != session.AgentClaude || entry.SessionID == "" {
		m.err = fmt.Errorf("only Claude sessions can be opened")
		m.previousState = stateGlobalSearchAction
		m.state = stateError
		return *m, nil
	}

	// Determine path from entry or use current working dir
	path := entry.Path
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			path = "."
		}
	}

	// Use custom name if provided, otherwise generate from snippet
	name := customName
	if name == "" {
		name = "claude"
		if entry.Snippet != "" {
			name = entry.Snippet
			if len(name) > 30 {
				name = name[:30] + "..."
			}
			// Clean up name
			name = strings.ReplaceAll(name, "\n", " ")
			name = strings.ReplaceAll(name, "\t", " ")
		}
	}

	// Create new instance
	inst, err := session.NewInstance(name, path, false, session.AgentClaude)
	if err != nil {
		m.err = err
		m.previousState = stateGlobalSearchAction
		m.state = stateError
		return *m, nil
	}

	// Set resume session ID
	inst.ResumeSessionID = entry.SessionID

	// Assign to group if specified
	if groupID != "" && groupID != "__from_global_search__" {
		inst.GroupID = groupID
	}

	// Add to storage
	if err := m.storage.AddInstance(inst); err != nil {
		m.err = err
		m.previousState = stateGlobalSearchAction
		m.state = stateError
		return *m, nil
	}

	// Start the session with resume
	if err := inst.StartWithResume(entry.SessionID); err != nil {
		m.err = err
		m.previousState = stateGlobalSearchAction
		m.state = stateError
		return *m, nil
	}
	m.storage.UpdateInstance(inst)

	m.instances = append(m.instances, inst)

	// Clear global search state
	m.globalSearchResults = nil
	m.globalSearchCursor = 0
	m.globalSearchExpanded = -1
	m.globalSearchConversation = nil
	m.globalSearchScroll = 0
	m.globalSearchSelectedEntry = nil

	// Move cursor to new session
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

	m.state = stateList
	return *m, nil
}

// addSearchEntryAsTab adds a global search entry as a new tab to the currently selected session
func (m *Model) addSearchEntryAsTab(entry *session.HistoryEntry) (Model, tea.Cmd) {
	// Only Claude entries can be resumed
	if entry.Agent != session.AgentClaude || entry.SessionID == "" {
		m.err = fmt.Errorf("only Claude sessions can be opened as tabs")
		m.previousState = stateGlobalSearchAction
		m.state = stateError
		return *m, nil
	}

	// Get selected session
	inst := m.getSelectedInstance()
	if inst == nil {
		m.err = fmt.Errorf("no session selected")
		m.previousState = stateGlobalSearchAction
		m.state = stateError
		return *m, nil
	}

	// Session must be running to add tabs
	if inst.Status != session.StatusRunning {
		m.err = fmt.Errorf("session must be running to add tabs")
		m.previousState = stateGlobalSearchAction
		m.state = stateError
		return *m, nil
	}

	// Generate tab name from snippet
	tabName := "claude"
	if entry.Snippet != "" {
		tabName = entry.Snippet
		if len(tabName) > 20 {
			tabName = tabName[:20] + "..."
		}
		tabName = strings.ReplaceAll(tabName, "\n", " ")
		tabName = strings.ReplaceAll(tabName, "\t", " ")
	}

	// Create a new tab with the resume session ID (uses existing NewForkedTab which does exactly this)
	if err := inst.NewForkedTab(tabName, entry.SessionID); err != nil {
		m.err = err
		m.previousState = stateGlobalSearchAction
		m.state = stateError
		return *m, nil
	}

	// Refresh status bar
	configureTmuxStatusBar(inst.TmuxSessionName(), inst.Name, inst.Color, inst.BgColor, inst.AutoYes)
	m.storage.UpdateInstance(inst)

	// Clear global search state
	m.globalSearchResults = nil
	m.globalSearchCursor = 0
	m.globalSearchExpanded = -1
	m.globalSearchConversation = nil
	m.globalSearchScroll = 0
	m.globalSearchSelectedEntry = nil

	m.state = stateList
	return *m, nil
}

// handleForkDialogKeys handles keyboard input in the fork dialog
func (m Model) handleForkDialogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel fork
		m.forkTarget = nil
		m.state = stateList
		return m, nil

	case "tab":
		// Toggle between tab and session
		m.forkToTab = !m.forkToTab
		return m, nil

	case "up", "k":
		// Select "New Tab"
		m.forkToTab = true
		return m, nil

	case "down", "j":
		// Select "New Session"
		m.forkToTab = false
		return m, nil

	case "enter":
		// Execute fork
		if m.forkTarget == nil {
			m.state = stateList
			return m, nil
		}

		forkName := strings.TrimSpace(m.forkNameInput.Value())
		if forkName == "" {
			forkName = m.forkTarget.Name + " (fork)"
		}

		// Execute fork with --fork-session
		newSessionID, err := m.forkTarget.ForkSession()
		if err != nil {
			m.err = fmt.Errorf("fork failed: %w", err)
			m.previousState = stateList
			m.state = stateError
			m.forkTarget = nil
			return m, nil
		}

		if m.forkToTab {
			// Fork to new tab in same session
			if err := m.forkTarget.NewForkedTab(forkName, newSessionID); err != nil {
				m.err = fmt.Errorf("failed to create fork tab: %w", err)
				m.previousState = stateList
				m.state = stateError
			} else {
				// Update status bar
				configureTmuxStatusBar(m.forkTarget.TmuxSessionName(), m.forkTarget.Name, m.forkTarget.Color, m.forkTarget.BgColor, m.forkTarget.AutoYes)
				m.storage.UpdateInstance(m.forkTarget)
			}
		} else {
			// Fork to new session
			newInst, err := session.NewInstance(forkName, m.forkTarget.Path, false, session.AgentClaude)
			if err != nil {
				m.err = fmt.Errorf("failed to create fork session: %w", err)
				m.previousState = stateList
				m.state = stateError
				m.forkTarget = nil
				return m, nil
			}

			// Copy settings from original
			newInst.GroupID = m.forkTarget.GroupID
			newInst.Color = m.forkTarget.Color
			newInst.BgColor = m.forkTarget.BgColor
			newInst.FullRowColor = m.forkTarget.FullRowColor
			newInst.ResumeSessionID = newSessionID
			newInst.Notes = fmt.Sprintf("Forked from: %s", m.forkTarget.Name)

			// Add to storage
			if err := m.storage.AddInstance(newInst); err != nil {
				m.err = fmt.Errorf("failed to save fork session: %w", err)
				m.previousState = stateList
				m.state = stateError
				m.forkTarget = nil
				return m, nil
			}

			// Start the forked session
			if err := newInst.StartWithResume(newSessionID); err != nil {
				m.err = fmt.Errorf("failed to start fork session: %w", err)
				m.previousState = stateList
				m.state = stateError
			} else {
				m.storage.UpdateInstance(newInst)
			}

			m.instances = append(m.instances, newInst)

			// Move cursor to new session
			if len(m.groups) > 0 {
				m.buildVisibleItems()
				for i, item := range m.visibleItems {
					if !item.isGroup && item.instance != nil && item.instance.ID == newInst.ID {
						m.cursor = i
						break
					}
				}
			} else {
				m.cursor = len(m.instances) - 1
			}
		}

		m.forkTarget = nil
		m.state = stateList
		return m, nil
	}

	// Update name input
	var cmd tea.Cmd
	m.forkNameInput, cmd = m.forkNameInput.Update(msg)
	return m, cmd
}
