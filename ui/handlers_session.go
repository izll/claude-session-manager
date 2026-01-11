package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/izll/agent-session-manager/session"
)

// isGradientColor checks if a color name is a gradient
func isGradientColor(color string) bool {
	_, exists := gradients[color]
	return exists
}

// RefreshTmuxStatusBar is the exported version for external calls
func RefreshTmuxStatusBar(sessionName, instanceName, fgColor, bgColor string, autoYes bool) {
	// Simple version for backward compatibility - only main window YOLO
	windowYolo := map[int]bool{0: autoYes}
	configureTmuxStatusBarWithYolo(sessionName, instanceName, fgColor, bgColor, windowYolo)
}

// RefreshTmuxStatusBarFull is the full version with per-window YOLO support
func RefreshTmuxStatusBarFull(sessionName, instanceName, fgColor, bgColor string, inst *session.Instance) {
	// Build map of window index -> autoYes
	windowYolo := map[int]bool{0: inst.AutoYes}
	for _, fw := range inst.FollowedWindows {
		windowYolo[fw.Index] = fw.AutoYes
	}
	configureTmuxStatusBarWithYolo(sessionName, instanceName, fgColor, bgColor, windowYolo)
}

// configureTmuxStatusBar is a backward compatible wrapper
func configureTmuxStatusBar(sessionName, instanceName, fgColor, bgColor string, autoYes bool) {
	windowYolo := map[int]bool{0: autoYes}
	configureTmuxStatusBarWithYolo(sessionName, instanceName, fgColor, bgColor, windowYolo)
}

// configureTmuxStatusBarWithYolo sets up the tmux status bar with per-window YOLO support
func configureTmuxStatusBarWithYolo(sessionName, instanceName, fgColor, bgColor string, windowYolo map[int]bool) {
	target := sessionName + ":"

	// Enable status bar
	exec.Command("tmux", "set-option", "-t", target, "status", "on").Run()

	// Status bar style - dark background
	exec.Command("tmux", "set-option", "-t", target, "status-style", "bg=#1a1a2e,fg=#888888").Run()

	// Get window list with names, index, active status, and dead status
	windowListOutput, _ := exec.Command("tmux", "list-windows", "-t", sessionName, "-F", "#{window_index}:#{window_name}:#{window_active}:#{pane_dead}").Output()
	windowLines := strings.Split(strings.TrimSpace(string(windowListOutput)), "\n")

	// Build status line with session name and tabs
	formattedName := formatTmuxSessionName(instanceName, fgColor, bgColor)
	var statusLeft strings.Builder
	statusLeft.WriteString(fmt.Sprintf("#[default,bg=#1a1a2e] %s ", formattedName))

	windowCount := 0
	if len(windowLines) > 0 && windowLines[0] != "" {
		windowCount = len(windowLines)
	}

	if windowCount > 1 {
		statusLeft.WriteString("#[fg=#555555]| ")

		for _, line := range windowLines {
			if line == "" {
				continue
			}
			parts := strings.Split(line, ":")
			if len(parts) < 4 {
				continue
			}
			windowIndex := parts[0]
			windowName := parts[1]
			isActive := parts[2] == "1"
			isDead := parts[3] == "1"

			deadPrefix := ""
			if isDead {
				deadPrefix = "○ "
			}

			// Show YOLO indicator for this window if it has YOLO enabled
			yoloIndicator := ""
			winIdx := 0
			fmt.Sscanf(windowIndex, "%d", &winIdx)
			if windowYolo[winIdx] {
				yoloIndicator = " #[fg=#FFA500]!"
			}

			if isActive {
				statusLeft.WriteString(fmt.Sprintf("#[fg=#FAFAFA,bold]%s%s#[nobold]%s", deadPrefix, windowName, yoloIndicator))
				statusLeft.WriteString("#[fg=#555555] | ")
			} else {
				statusLeft.WriteString(fmt.Sprintf("#[fg=#888888]%s%s%s #[fg=#555555]| ", deadPrefix, windowName, yoloIndicator))
			}
		}
	} else if windowYolo[0] {
		statusLeft.WriteString("#[fg=#FFA500,bold]YOLO !")
	}

	// Set status-left with our tab list
	exec.Command("tmux", "set-option", "-t", target, "status-left", statusLeft.String()).Run()
	exec.Command("tmux", "set-option", "-t", target, "status-left-length", "500").Run()

	// Hide tmux's built-in window list
	exec.Command("tmux", "set-option", "-t", target, "window-status-format", "").Run()
	exec.Command("tmux", "set-option", "-t", target, "window-status-current-format", "").Run()
	exec.Command("tmux", "set-option", "-t", target, "window-status-separator", "").Run()

	// Use status-format to hide window list completely
	statusFormat := fmt.Sprintf("#[align=left]%s#[align=right]#[fg=#555555]Alt+</>: tabs | Ctrl+Q: detach ", statusLeft.String())
	exec.Command("tmux", "set-option", "-t", target, "status-format[0]", statusFormat).Run()

	// Right side (backup, status-format overrides this)
	exec.Command("tmux", "set-option", "-t", target, "status-right", "").Run()

	// Hook to refresh status bar when window changes
	refreshCmd := fmt.Sprintf("asmgr refresh-status %s", sessionName)
	exec.Command("tmux", "set-hook", "-t", sessionName, "window-linked", fmt.Sprintf("run-shell '%s'", refreshCmd)).Run()
	exec.Command("tmux", "set-hook", "-t", sessionName, "window-unlinked", fmt.Sprintf("run-shell '%s'", refreshCmd)).Run()
	exec.Command("tmux", "set-hook", "-t", sessionName, "session-window-changed", fmt.Sprintf("run-shell '%s'", refreshCmd)).Run()

	// Key bindings for tab switching
	exec.Command("tmux", "bind-key", "-n", "M-Left", "previous-window").Run()
	exec.Command("tmux", "bind-key", "-n", "M-Right", "next-window").Run()
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
	} else {
		// Session is running - check if active tab is dead and respawn it
		windows := inst.GetWindowList()
		for _, w := range windows {
			if w.Active && w.Dead {
				inst.RespawnWindow(w.Index)
				break
			}
		}
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

	// Update window 0 name to agent type (session name is shown in status bar)
	exec.Command("tmux", "rename-window", "-t", sessionName+":0", inst.WindowName()).Run()

	// Configure tmux status bar to show tabs with per-window YOLO support
	RefreshTmuxStatusBarFull(sessionName, inst.Name, inst.Color, inst.BgColor, inst)

	// Set up Ctrl+Q to resize to preview size before detach
	tmuxWidth, tmuxHeight := m.calculateTmuxDimensions()
	inst.UpdateDetachBinding(tmuxWidth, tmuxHeight)
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return reattachMsg{}
	})
}

// handleResumeSession shows agent sessions for the current instance's active tab
func (m *Model) handleResumeSession() error {
	inst := m.getSelectedInstance()
	if inst == nil {
		return nil
	}

	// Determine agent type based on active window
	agentType := inst.Agent
	if agentType == "" {
		agentType = session.AgentClaude
	}
	activeWindowIndex := 0

	// Check if there's an active followed window
	if inst.Status == session.StatusRunning {
		windows := inst.GetWindowList()
		for _, w := range windows {
			if w.Active {
				activeWindowIndex = w.Index
				if w.Followed {
					// Find the followed window's agent type
					for _, fw := range inst.FollowedWindows {
						if fw.Index == w.Index {
							agentType = fw.Agent
							break
						}
					}
				}
				break
			}
		}
	}

	// Terminal windows don't support resume
	if agentType == session.AgentTerminal {
		return fmt.Errorf("terminal windows don't support session resume")
	}

	// List sessions based on agent type
	var sessions []session.AgentSession
	var err error

	switch agentType {
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
		return fmt.Errorf("no previous %s sessions found", agentType)
	}
	m.agentSessions = sessions
	m.resumeAgentType = agentType           // Store which agent type we're resuming
	m.resumeWindowIndex = activeWindowIndex // Store which window to resume
	m.sessionCursor = 1                     // Start with first session selected (0 is "new session")
	m.state = stateSelectAgentSession
	return nil
}

// handleStartSession starts the selected session without attaching
func (m *Model) handleStartSession() {
	inst := m.getSelectedInstance()
	if inst == nil {
		return
	}
	// Update status based on actual tmux session state
	inst.UpdateStatus()
	m.storage.UpdateInstance(inst)

	if inst.Status != session.StatusRunning {
		// Session is stopped - start it
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
	} else {
		// Session is running - check if window 0 (main agent) is dead
		windows := inst.GetWindowList()
		for _, w := range windows {
			if w.Index == 0 && w.Dead {
				// Window 0 is dead - respawn it with resume ID if available
				var err error
				if inst.ResumeSessionID != "" {
					err = inst.RespawnWindowWithResume(0, inst.ResumeSessionID)
				} else {
					err = inst.RespawnWindow(0)
				}
				if err != nil {
					m.err = err
					m.previousState = stateList
					m.state = stateError
				}
				return
			}
		}
	}
}

// handleStopSession shows confirmation dialog for stopping the selected session
func (m *Model) handleStopSession() {
	inst := m.getSelectedInstance()
	if inst == nil {
		return
	}
	if inst.Status == session.StatusRunning {
		m.stopTarget = inst
		m.state = stateConfirmStop
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
	m.promptInput.SetWidth(inputWidth)
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

// handleToggleAutoYes shows confirmation dialog for toggling YOLO mode on the active tab
// Returns a tea.Cmd (currently nil, confirmation happens in handleConfirmYoloKeys)
func (m *Model) handleToggleAutoYes() tea.Cmd {
	inst := m.getSelectedInstance()
	if inst == nil {
		return nil
	}

	// Determine active window and its agent type
	activeWindowIndex := 0
	agentType := inst.Agent
	if agentType == "" {
		agentType = session.AgentClaude
	}
	currentYolo := inst.AutoYes

	// Check if session is running and has an active followed window
	if inst.Status == session.StatusRunning {
		windows := inst.GetWindowList()
		for _, w := range windows {
			if w.Active {
				activeWindowIndex = w.Index
				if w.Followed {
					// Find the followed window's settings
					for _, fw := range inst.FollowedWindows {
						if fw.Index == w.Index {
							agentType = fw.Agent
							currentYolo = fw.AutoYes
							break
						}
					}
				}
				break
			}
		}
	}

	// Special handling for Gemini - send Ctrl+Y keystroke instead (Gemini has its own confirmation)
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

	// Terminal windows don't support YOLO
	if agentType == session.AgentTerminal {
		m.err = fmt.Errorf("terminal windows don't support YOLO mode")
		m.previousState = stateList
		m.state = stateError
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

	// Show confirmation dialog
	m.yoloTarget = inst
	m.yoloWindowIndex = activeWindowIndex
	m.yoloNewState = !currentYolo
	m.state = stateConfirmYolo

	return nil
}
