package ui

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/izll/claude-session-manager/session"
)

// handleMoveSessionUp moves the selected session up in the list
func (m *Model) handleMoveSessionUp() {
	if m.cursor > 0 && len(m.instances) > 1 {
		m.instances[m.cursor], m.instances[m.cursor-1] = m.instances[m.cursor-1], m.instances[m.cursor]
		m.cursor--
		m.storage.SaveAll(m.instances)
	}
}

// handleMoveSessionDown moves the selected session down in the list
func (m *Model) handleMoveSessionDown() {
	if m.cursor < len(m.instances)-1 {
		m.instances[m.cursor], m.instances[m.cursor+1] = m.instances[m.cursor+1], m.instances[m.cursor]
		m.cursor++
		m.storage.SaveAll(m.instances)
	}
}

// handleEnterSession starts (if needed) and attaches to the selected session
func (m *Model) handleEnterSession() tea.Cmd {
	if len(m.instances) == 0 {
		return nil
	}
	inst := m.instances[m.cursor]
	if inst.Status != session.StatusRunning {
		if err := inst.Start(); err != nil {
			m.err = err
			return nil
		}
		m.storage.UpdateInstance(inst)
	}
	sessionName := inst.TmuxSessionName()
	// Configure tmux for proper terminal resize following
	if err := exec.Command("tmux", "set-option", "-t", sessionName, "window-size", "largest").Run(); err != nil {
		m.err = fmt.Errorf("failed to set tmux window-size: %w", err)
	}
	if err := exec.Command("tmux", "set-option", "-t", sessionName, "aggressive-resize", "on").Run(); err != nil {
		m.err = fmt.Errorf("failed to set tmux aggressive-resize: %w", err)
	}
	// Enable focus events for hooks to work
	exec.Command("tmux", "set-option", "-t", sessionName, "focus-events", "on").Run()
	// Set up hook to resize window on focus gain (fixes Konsole tab switch issue)
	exec.Command("tmux", "set-hook", "-t", sessionName, "client-focus-in", "resize-window -A").Run()
	exec.Command("tmux", "set-hook", "-t", sessionName, "pane-focus-in", "resize-window -A").Run()
	// Set up Ctrl+Q to resize to preview size before detach
	tmuxWidth, tmuxHeight := m.calculateTmuxDimensions()
	inst.UpdateDetachBinding(tmuxWidth, tmuxHeight)
	inst.ClosePty()
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return reattachMsg{}
	})
}

// handleResumeSession shows Claude sessions for the current instance
func (m *Model) handleResumeSession() error {
	if len(m.instances) == 0 {
		return nil
	}
	inst := m.instances[m.cursor]
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
	if len(m.instances) == 0 {
		return
	}
	inst := m.instances[m.cursor]
	if inst.Status != session.StatusRunning {
		if err := inst.Start(); err != nil {
			m.err = err
		} else {
			m.storage.UpdateInstance(inst)
		}
	}
}

// handleStopSession stops the selected session
func (m *Model) handleStopSession() {
	if len(m.instances) == 0 {
		return
	}
	inst := m.instances[m.cursor]
	if inst.Status == session.StatusRunning {
		inst.Stop()
		m.storage.UpdateInstance(inst)
	}
}

// handleRenameSession opens the rename dialog for the selected session
func (m *Model) handleRenameSession() tea.Cmd {
	if len(m.instances) == 0 {
		return nil
	}
	inst := m.instances[m.cursor]
	m.nameInput.SetValue(inst.Name)
	m.nameInput.Focus()
	m.state = stateRename
	return textinput.Blink
}

// handleColorPicker opens the color picker for the selected session
func (m *Model) handleColorPicker() {
	if len(m.instances) == 0 {
		return
	}
	inst := m.instances[m.cursor]
	// Initialize preview colors
	m.previewFg = inst.Color
	m.previewBg = inst.BgColor
	m.colorMode = 0
	// Find current color index
	m.colorCursor = 0
	for i, c := range colorOptions {
		if c.Color == inst.Color || c.Name == inst.Color {
			m.colorCursor = i
			break
		}
	}
	m.state = stateColorPicker
}

// handleSendPrompt opens the prompt input for the selected session
func (m *Model) handleSendPrompt() {
	if len(m.instances) == 0 {
		return
	}
	inst := m.instances[m.cursor]
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
	if len(m.instances) == 0 || m.cursor >= len(m.instances) {
		return
	}
	inst := m.instances[m.cursor]
	tmuxWidth, tmuxHeight := m.calculateTmuxDimensions()
	if err := inst.ResizePane(tmuxWidth, tmuxHeight); err != nil {
		m.err = fmt.Errorf("failed to resize pane: %w", err)
	}
}

// handleListKeys handles keyboard input in the main list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.resizeSelectedPane()
		}

	case "down", "j":
		if m.cursor < len(m.instances)-1 {
			m.cursor++
			m.resizeSelectedPane()
		}

	case "shift+up", "K":
		m.handleMoveSessionUp()

	case "shift+down", "J":
		m.handleMoveSessionDown()

	case "enter":
		if cmd := m.handleEnterSession(); cmd != nil {
			return m, cmd
		}

	case "n":
		m.state = stateNewPath
		m.pathInput.SetValue("")
		m.pathInput.Focus()
		return m, textinput.Blink

	case "r":
		if err := m.handleResumeSession(); err != nil {
			m.err = err
		}

	case "s":
		m.handleStartSession()

	case "x":
		m.handleStopSession()

	case "d":
		if len(m.instances) > 0 {
			m.deleteTarget = m.instances[m.cursor]
			m.state = stateConfirmDelete
		}

	case "y":
		m.autoYes = !m.autoYes

	case "e":
		if cmd := m.handleRenameSession(); cmd != nil {
			return m, cmd
		}

	case "?":
		m.state = stateHelp

	case "c":
		m.handleColorPicker()

	case "l":
		m.compactList = !m.compactList

	case "p":
		m.handleSendPrompt()

	case "R":
		m.handleForceResize()
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
			inst, err := session.NewInstance(m.nameInput.Value(), m.pathInput.Value(), m.autoYes)
			if err != nil {
				m.err = err
				m.state = stateList
				return m, nil
			}

			// Check for existing Claude sessions
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

			// No existing sessions, just create new
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
			m.cursor = len(m.instances) - 1
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
			m.cursor = len(m.instances) - 1
			m.pendingInstance = nil
		} else if len(m.instances) > 0 {
			// Resuming existing instance
			inst := m.instances[m.cursor]
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
		if m.nameInput.Value() != "" && len(m.instances) > 0 {
			inst := m.instances[m.cursor]
			inst.Name = m.nameInput.Value()
			m.storage.UpdateInstance(inst)
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
		if m.promptInput.Value() != "" && len(m.instances) > 0 {
			inst := m.instances[m.cursor]
			if inst.Status == session.StatusRunning {
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
		if m.colorCursor < len(colorOptions) {
			selected := colorOptions[m.colorCursor]
			if m.colorMode == 0 {
				m.previewFg = selected.Color
			} else {
				m.previewBg = selected.Color
			}
		}
		// Switch between foreground and background mode
		m.colorMode = 1 - m.colorMode
		// Find cursor position for the other mode's current value
		m.colorCursor = 0
		targetColor := m.previewFg
		if m.colorMode == 1 {
			targetColor = m.previewBg
		}
		for i, c := range colorOptions {
			if c.Color == targetColor {
				m.colorCursor = i
				break
			}
		}
		// Reset cursor if it's beyond the new max
		if m.colorCursor >= maxItems {
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
		if len(m.instances) > 0 {
			inst := m.instances[m.cursor]
			inst.FullRowColor = !inst.FullRowColor
			m.storage.UpdateInstance(inst)
		}

	case "enter":
		if len(m.instances) > 0 {
			inst := m.instances[m.cursor]
			selected := colorOptions[m.colorCursor]

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
