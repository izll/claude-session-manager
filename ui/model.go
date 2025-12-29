package ui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/izll/claude-session-manager/session"
)

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// truncateWithANSI truncates a string to maxLen visible characters while preserving ANSI codes
func truncateWithANSI(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	var result strings.Builder
	visibleCount := 0
	i := 0
	runes := []rune(s)

	for i < len(runes) {
		// Check for ANSI escape sequence
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Find end of ANSI sequence
			start := i
			i += 2
			for i < len(runes) && !((runes[i] >= 'A' && runes[i] <= 'Z') || (runes[i] >= 'a' && runes[i] <= 'z')) {
				i++
			}
			if i < len(runes) {
				i++ // include the final letter
			}
			// Always include ANSI codes
			result.WriteString(string(runes[start:i]))
		} else {
			if visibleCount >= maxLen {
				result.WriteString("…")
				// Add reset code to ensure colors don't leak
				result.WriteString("\x1b[0m")
				break
			}
			result.WriteRune(runes[i])
			visibleCount++
			i++
		}
	}

	return result.String()
}

type state int

const (
	stateList state = iota
	stateNewName
	stateNewPath
	stateSelectClaudeSession // New state for selecting Claude session to resume
	stateConfirmDelete
	stateRename
	stateHelp
	stateColorPicker
	statePrompt // Send text to session
)

type Model struct {
	instances       []*session.Instance
	storage         *session.Storage
	cursor          int
	state           state
	width           int
	height          int
	nameInput       textinput.Model
	pathInput       textinput.Model
	promptInput     textinput.Model          // Input for sending text to session
	autoYes         bool
	deleteTarget    *session.Instance
	preview         string
	err             error
	claudeSessions  []session.ClaudeSession // Claude sessions for current instance
	sessionCursor   int                      // Cursor for Claude session selection
	pendingInstance *session.Instance        // Instance being created
	lastLines       map[string]string        // Last output line for each instance (by ID)
	prevContent     map[string]string        // Previous content hash to detect activity
	isActive        map[string]bool          // Whether instance has recent activity
	colorCursor     int                      // Cursor for color picker
	colorMode       int                      // 0 = foreground, 1 = background
	previewFg       string                   // Preview foreground color
	previewBg       string                   // Preview background color
	compactList     bool                     // No extra line between sessions
}

type tickMsg time.Time
type reattachMsg struct{}

func NewModel() (Model, error) {
	storage, err := session.NewStorage()
	if err != nil {
		return Model{}, err
	}

	instances, err := storage.Load()
	if err != nil {
		return Model{}, err
	}

	nameInput := textinput.New()
	nameInput.Placeholder = "Session name"
	nameInput.CharLimit = 50

	pathInput := textinput.New()
	pathInput.Placeholder = "/path/to/project"
	pathInput.CharLimit = 256

	promptInput := textinput.New()
	promptInput.Placeholder = "Enter message to send..."
	promptInput.CharLimit = 1000

	m := Model{
		instances:   instances,
		storage:     storage,
		state:       stateList,
		nameInput:   nameInput,
		pathInput:   pathInput,
		promptInput: promptInput,
		lastLines:   make(map[string]string),
		prevContent: make(map[string]string),
		isActive:    make(map[string]bool),
	}

	// Initialize status and last lines for all instances
	for _, inst := range instances {
		inst.UpdateStatus()
		m.lastLines[inst.ID] = inst.GetLastLine()
	}

	// Initialize preview for first instance
	if len(instances) > 0 {
		m.preview, _ = instances[0].GetPreview(100)
	}

	return m, nil
}

func (m Model) Init() tea.Cmd {
	// Set terminal tab color (works in some terminals like iTerm2, Konsole)
	// Purple color to match the theme
	fmt.Print("\033]6;1;bg;red;brightness;125\007")
	fmt.Print("\033]6;1;bg;green;brightness;86\007")
	fmt.Print("\033]6;1;bg;blue;brightness;244\007")

	return tea.Batch(
		tickCmd(),
		tea.EnterAltScreen,
		tea.SetWindowTitle("Claude Session Manager"),
		tea.EnableMouseCellMotion,
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case reattachMsg:
		// Re-enable mouse after returning from tmux attach
		return m, tea.EnableMouseCellMotion

	case tea.MouseMsg:
		// Handle mouse wheel scrolling in list view
		if m.state == stateList && len(m.instances) > 0 {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case tea.MouseButtonWheelDown:
				if m.cursor < len(m.instances)-1 {
					m.cursor++
				}
				return m, nil
			}
		}

	case tickMsg:
		// Remember currently selected instance ID
		var selectedID string
		if len(m.instances) > 0 && m.cursor < len(m.instances) {
			selectedID = m.instances[m.cursor].ID
		}

		// Update all instance statuses and last lines
		for _, inst := range m.instances {
			inst.UpdateStatus()
			// Update last line for status display
			currentLine := inst.GetLastLine()
			m.lastLines[inst.ID] = currentLine

			// Detect activity by comparing with previous content
			if inst.Status == session.StatusRunning {
				prevLine := m.prevContent[inst.ID]
				if currentLine != prevLine && prevLine != "" {
					m.isActive[inst.ID] = true
				} else {
					m.isActive[inst.ID] = false
				}
				m.prevContent[inst.ID] = currentLine
			} else {
				m.isActive[inst.ID] = false
			}
		}

		// Keep sessions in user-defined order (no auto-sorting)

		// Restore cursor to previously selected instance
		if selectedID != "" {
			for i, inst := range m.instances {
				if inst.ID == selectedID {
					m.cursor = i
					break
				}
			}
		}

		// Update preview for selected instance (more lines for better visibility)
		if len(m.instances) > 0 && m.cursor < len(m.instances) {
			preview, _ := m.instances[m.cursor].GetPreview(100)
			m.preview = preview
		}
		return m, tickCmd()

	case tea.KeyMsg:
		switch m.state {
		case stateList:
			return m.handleListKeys(msg)
		case stateNewName:
			return m.handleNewNameKeys(msg)
		case stateNewPath:
			return m.handleNewPathKeys(msg)
		case stateSelectClaudeSession:
			return m.handleSelectSessionKeys(msg)
		case stateConfirmDelete:
			return m.handleConfirmDeleteKeys(msg)
		case stateRename:
			return m.handleRenameKeys(msg)
		case stateHelp:
			return m.handleHelpKeys(msg)
		case stateColorPicker:
			return m.handleColorPickerKeys(msg)
		case statePrompt:
			return m.handlePromptKeys(msg)
		}
	}

	if m.state == stateNewName || m.state == stateRename {
		m.nameInput, cmd = m.nameInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.state == stateNewPath {
		m.pathInput, cmd = m.pathInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.state == statePrompt {
		m.promptInput, cmd = m.promptInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.instances)-1 {
			m.cursor++
		}

	case "shift+up", "K":
		// Move session up
		if m.cursor > 0 && len(m.instances) > 1 {
			m.instances[m.cursor], m.instances[m.cursor-1] = m.instances[m.cursor-1], m.instances[m.cursor]
			m.cursor--
			m.storage.SaveAll(m.instances)
		}

	case "shift+down", "J":
		// Move session down
		if m.cursor < len(m.instances)-1 {
			m.instances[m.cursor], m.instances[m.cursor+1] = m.instances[m.cursor+1], m.instances[m.cursor]
			m.cursor++
			m.storage.SaveAll(m.instances)
		}

	case "enter":
		if len(m.instances) > 0 {
			inst := m.instances[m.cursor]
			if inst.Status != session.StatusRunning {
				// Start first
				if err := inst.Start(); err != nil {
					m.err = err
					return m, nil
				}
				m.storage.UpdateInstance(inst)
			}
			// Attach using exec.Command with -d flag to force resize to current terminal
			cmd := exec.Command("tmux", "attach-session", "-d", "-t", inst.TmuxSessionName())
			return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
				return reattachMsg{}
			})
		}

	case "n":
		m.state = stateNewPath
		m.pathInput.SetValue("")
		m.pathInput.Focus()
		return m, textinput.Blink

	case "r":
		// Resume: show Claude sessions for current instance
		if len(m.instances) > 0 {
			inst := m.instances[m.cursor]
			sessions, err := session.ListClaudeSessions(inst.Path)
			if err != nil {
				m.err = err
				return m, nil
			}
			if len(sessions) == 0 {
				m.err = fmt.Errorf("no previous Claude sessions found for this path")
				return m, nil
			}
			m.claudeSessions = sessions
			m.sessionCursor = 0
			m.state = stateSelectClaudeSession
		}

	case "s":
		if len(m.instances) > 0 {
			inst := m.instances[m.cursor]
			if inst.Status != session.StatusRunning {
				if err := inst.Start(); err != nil {
					m.err = err
				} else {
					m.storage.UpdateInstance(inst)
				}
			}
		}

	case "x":
		if len(m.instances) > 0 {
			inst := m.instances[m.cursor]
			if inst.Status == session.StatusRunning {
				inst.Stop()
				m.storage.UpdateInstance(inst)
			}
		}

	case "d":
		if len(m.instances) > 0 {
			m.deleteTarget = m.instances[m.cursor]
			m.state = stateConfirmDelete
		}

	case "y":
		m.autoYes = !m.autoYes

	case "e":
		// Rename session
		if len(m.instances) > 0 {
			inst := m.instances[m.cursor]
			m.nameInput.SetValue(inst.Name)
			m.nameInput.Focus()
			m.state = stateRename
			return m, textinput.Blink
		}

	case "?":
		m.state = stateHelp

	case "c":
		// Color picker
		if len(m.instances) > 0 {
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

	case "l":
		// Toggle compact list (no spacing between sessions)
		m.compactList = !m.compactList

	case "p":
		// Send prompt to session
		if len(m.instances) > 0 {
			inst := m.instances[m.cursor]
			if inst.Status == session.StatusRunning {
				m.promptInput.SetValue("")
				m.promptInput.Focus()
				m.state = statePrompt
				return m, textinput.Blink
			} else {
				m.err = fmt.Errorf("session not running")
			}
		}
	}

	return m, nil
}

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
			sessions, _ := session.ListClaudeSessions(inst.Path)
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

func (m Model) handleConfirmDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.deleteTarget != nil {
			m.storage.RemoveInstance(m.deleteTarget.ID)
			// Reload instances
			instances, _ := m.storage.Load()
			m.instances = instances
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

func (m Model) handleHelpKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?":
		m.state = stateList
	}
	return m, nil
}

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

func (m Model) handleColorPickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Calculate max items based on mode
	// colorOptions has: none, auto, 22 solid colors, 15 gradients = 39 total
	// For background: skip auto (index 1) and gradients (last 15)
	maxItems := len(colorOptions)
	if m.colorMode == 1 {
		// Background mode - exclude gradients (last 15 items) and auto
		maxItems = len(colorOptions) - 15
	}

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

func (m Model) View() string {
	switch m.state {
	case stateHelp:
		return m.helpView()
	case stateConfirmDelete:
		return m.confirmDeleteView()
	case stateNewName, stateNewPath:
		return m.newInstanceView()
	case stateRename:
		return m.renameView()
	case stateSelectClaudeSession:
		return m.selectSessionView()
	case stateColorPicker:
		return m.colorPickerView()
	case statePrompt:
		return m.promptView()
	default:
		return m.listView()
	}
}

// Gradient definitions for text coloring
var gradients = map[string][]string{
	"gradient-rainbow":  {"#FF0000", "#FF7F00", "#FFFF00", "#00FF00", "#00FFFF", "#0000FF", "#8B00FF"},
	"gradient-sunset":   {"#FF512F", "#F09819", "#FF8C00", "#DD2476", "#FF416C"},
	"gradient-ocean":    {"#00D2FF", "#3A7BD5", "#00D2D3", "#54A0FF", "#2E86DE"},
	"gradient-forest":   {"#134E5E", "#11998E", "#38EF7D", "#A8E063", "#56AB2F"},
	"gradient-fire":     {"#FF0000", "#FF4500", "#FF6347", "#FF8C00", "#FFD700"},
	"gradient-ice":      {"#E0FFFF", "#B0E0E6", "#87CEEB", "#00CED1", "#4682B4"},
	"gradient-neon":     {"#FF00FF", "#00FFFF", "#39FF14", "#FF6600", "#BF00FF"},
	"gradient-galaxy":   {"#0F0C29", "#302B63", "#8E2DE2", "#4A00E0", "#24243E"},
	"gradient-pastel":   {"#FFB6C1", "#FFDAB9", "#FFFACD", "#98FB98", "#ADD8E6", "#E6E6FA"},
	"gradient-pink":     {"#FF69B4", "#FF1493", "#DB7093", "#FF69B4"},
	"gradient-blue":     {"#00BFFF", "#1E90FF", "#4169E1", "#0000FF", "#4169E1", "#1E90FF"},
	"gradient-green":    {"#00FF00", "#32CD32", "#228B22", "#006400", "#228B22", "#32CD32"},
	"gradient-gold":     {"#FFD700", "#FFA500", "#FF8C00", "#FFA500", "#FFD700"},
	"gradient-purple":   {"#9400D3", "#8A2BE2", "#9932CC", "#BA55D3", "#9932CC", "#8A2BE2"},
	"gradient-cyber":    {"#00FF00", "#00FFFF", "#FF00FF", "#00FFFF", "#00FF00"},
}

// Available colors for foreground/background
var colorOptions = []struct {
	Name  string
	Color string
}{
	{"none", ""},
	{"auto", "auto"},
	{"black", "#000000"},
	{"white", "#FFFFFF"},
	{"red", "#FF6B6B"},
	{"orange", "#FFA500"},
	{"yellow", "#FFD93D"},
	{"lime", "#ADFF2F"},
	{"green", "#6BCB77"},
	{"teal", "#20B2AA"},
	{"cyan", "#4DD0E1"},
	{"sky", "#87CEEB"},
	{"blue", "#6C9EFF"},
	{"indigo", "#7B68EE"},
	{"purple", "#B388FF"},
	{"magenta", "#FF00FF"},
	{"pink", "#FF8FAB"},
	{"rose", "#FF69B4"},
	{"coral", "#FF7F50"},
	{"gold", "#FFD700"},
	{"silver", "#C0C0C0"},
	{"gray", "#888888"},
	{"dark-red", "#8B0000"},
	{"dark-green", "#006400"},
	{"dark-blue", "#00008B"},
	{"dark-purple", "#4B0082"},
	// Gradients at the end (for foreground only)
	{"gradient-rainbow", "gradient-rainbow"},
	{"gradient-sunset", "gradient-sunset"},
	{"gradient-ocean", "gradient-ocean"},
	{"gradient-forest", "gradient-forest"},
	{"gradient-fire", "gradient-fire"},
	{"gradient-ice", "gradient-ice"},
	{"gradient-neon", "gradient-neon"},
	{"gradient-galaxy", "gradient-galaxy"},
	{"gradient-pastel", "gradient-pastel"},
	{"gradient-pink", "gradient-pink"},
	{"gradient-blue", "gradient-blue"},
	{"gradient-green", "gradient-green"},
	{"gradient-gold", "gradient-gold"},
	{"gradient-purple", "gradient-purple"},
	{"gradient-cyber", "gradient-cyber"},
}

// For backward compatibility
var sessionColors = colorOptions

// hexToRGB converts hex color to RGB values
func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 255, 255, 255
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	return int(r), int(g), int(b)
}

// interpolateColor interpolates between colors in a gradient
func interpolateColor(colors []string, position float64) string {
	if len(colors) == 0 {
		return "#FFFFFF"
	}
	if len(colors) == 1 {
		return colors[0]
	}

	// Clamp position
	if position <= 0 {
		return colors[0]
	}
	if position >= 1 {
		return colors[len(colors)-1]
	}

	// Find which segment we're in
	segment := position * float64(len(colors)-1)
	idx := int(segment)
	if idx >= len(colors)-1 {
		idx = len(colors) - 2
	}
	t := segment - float64(idx)

	r1, g1, b1 := hexToRGB(colors[idx])
	r2, g2, b2 := hexToRGB(colors[idx+1])

	r := int(float64(r1) + t*(float64(r2)-float64(r1)))
	g := int(float64(g1) + t*(float64(g2)-float64(g1)))
	b := int(float64(b1) + t*(float64(b2)-float64(b1)))

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// applyGradient applies a gradient to text, coloring each character
func applyGradient(text string, gradientName string) string {
	colors, ok := gradients[gradientName]
	if !ok || len(text) == 0 {
		return text
	}

	runes := []rune(text)
	var result strings.Builder

	for i, r := range runes {
		position := float64(i) / float64(len(runes)-1)
		if len(runes) == 1 {
			position = 0.5
		}
		color := interpolateColor(colors, position)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		result.WriteString(style.Render(string(r)))
	}

	return result.String()
}

// applyGradientWithBg applies a gradient with background color
func applyGradientWithBg(text string, gradientName string, bgColor string) string {
	colors, ok := gradients[gradientName]
	if !ok || len(text) == 0 {
		return text
	}

	runes := []rune(text)
	var result strings.Builder

	for i, r := range runes {
		position := float64(i) / float64(len(runes)-1)
		if len(runes) == 1 {
			position = 0.5
		}
		color := interpolateColor(colors, position)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Background(lipgloss.Color(bgColor))
		result.WriteString(style.Render(string(r)))
	}

	return result.String()
}

// applyGradientWithBgBold applies a gradient with background color and bold text
func applyGradientWithBgBold(text string, gradientName string, bgColor string) string {
	colors, ok := gradients[gradientName]
	if !ok || len(text) == 0 {
		return text
	}

	runes := []rune(text)
	var result strings.Builder

	for i, r := range runes {
		position := float64(i) / float64(len(runes)-1)
		if len(runes) == 1 {
			position = 0.5
		}
		color := interpolateColor(colors, position)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Background(lipgloss.Color(bgColor)).Bold(true)
		result.WriteString(style.Render(string(r)))
	}

	return result.String()
}

// getContrastColor returns black or white based on background luminance
func getContrastColor(bgColor string) string {
	r, g, b := hexToRGB(bgColor)
	// Calculate luminance
	luminance := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 255.0
	if luminance > 0.5 {
		return "#000000" // Dark text for light background
	}
	return "#FFFFFF" // Light text for dark background
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4"))

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")) // Orange for activity

	idleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")) // Grey for idle

	stoppedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87"))

	previewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	sessionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700"))

	listPaneStyle = lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.Border{Right: "│"}).
			BorderForeground(lipgloss.Color("#555555"))

	previewPaneStyle = lipgloss.NewStyle()

	listSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7D56F4")).
				Bold(true)
)

func (m Model) listView() string {
	// Fixed width for left panel (sessions list)
	listWidth := 45
	previewWidth := m.width - listWidth - 3 // -3 for borders/padding
	if previewWidth < 40 {
		previewWidth = 40
	}
	contentHeight := m.height - 1 // Reserve space for status bar
	if contentHeight < 10 {
		contentHeight = 10
	}

	// Build left pane (session list)
	var leftPane strings.Builder
	leftPane.WriteString("\n")
	leftPane.WriteString(titleStyle.Render(" Sessions "))
	leftPane.WriteString("\n\n")

	if len(m.instances) == 0 {
		leftPane.WriteString(" No sessions\n")
		leftPane.WriteString(dimStyle.Render(" Press 'n' to create"))
	} else {
		// Calculate visible range (each session takes 2 or 3 lines depending on compact mode)
		linesPerSession := 2
		if !m.compactList {
			linesPerSession = 3
		}
		maxVisible := (contentHeight - 4) / linesPerSession
		if maxVisible < 3 {
			maxVisible = 3
		}

		startIdx := 0
		if m.cursor >= maxVisible {
			startIdx = m.cursor - maxVisible + 1
		}
		endIdx := startIdx + maxVisible
		if endIdx > len(m.instances) {
			endIdx = len(m.instances)
		}

		// Show scroll indicator at top
		if startIdx > 0 {
			leftPane.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more\n", startIdx)))
		}

		for i := startIdx; i < endIdx; i++ {
			inst := m.instances[i]
			// Status indicator based on activity
			var status string
			if inst.Status == session.StatusRunning {
				if m.isActive[inst.ID] {
					status = activeStyle.Render("●") // Orange - active
				} else {
					status = idleStyle.Render("●") // Grey - idle/waiting
				}
			} else {
				status = stoppedStyle.Render("○") // Red outline - stopped
			}

			// Truncate name to fit
			name := inst.Name
			maxNameLen := listWidth - 6
			if maxNameLen < 10 {
				maxNameLen = 10
			}
			if len(name) > maxNameLen {
				name = name[:maxNameLen-2] + "…"
			}

			// Apply session colors
			styledName := name
			style := lipgloss.NewStyle()

			// Apply background color first
			if inst.BgColor != "" {
				style = style.Background(lipgloss.Color(inst.BgColor))
			}

			// Apply foreground color
			if inst.Color != "" {
				if inst.Color == "auto" && inst.BgColor != "" {
					// Auto mode: calculate contrast color
					autoColor := getContrastColor(inst.BgColor)
					style = style.Foreground(lipgloss.Color(autoColor))
					styledName = style.Render(name)
				} else if _, isGradient := gradients[inst.Color]; isGradient {
					// Gradient - apply to each character
					if inst.BgColor != "" {
						styledName = applyGradientWithBg(name, inst.Color, inst.BgColor)
					} else {
						styledName = applyGradient(name, inst.Color)
					}
				} else {
					style = style.Foreground(lipgloss.Color(inst.Color))
					styledName = style.Render(name)
				}
			} else if inst.BgColor != "" {
				// Only background, use auto text color
				autoColor := getContrastColor(inst.BgColor)
				style = style.Foreground(lipgloss.Color(autoColor))
				styledName = style.Render(name)
			}

			// Selected row is always bold
			if i == m.cursor {
				if inst.FullRowColor && inst.BgColor != "" {
					// Full row background
					if _, isGradient := gradients[inst.Color]; isGradient {
						// Gradient text on full row background
						padding := listWidth - 7 - len([]rune(name))
						paddingStr := ""
						if padding > 0 {
							paddingStr = lipgloss.NewStyle().Background(lipgloss.Color(inst.BgColor)).Render(strings.Repeat(" ", padding))
						}
						gradientText := applyGradientWithBgBold(name, inst.Color, inst.BgColor)
						leftPane.WriteString(fmt.Sprintf(" %s %s %s%s", listSelectedStyle.Render("▸"), status, gradientText, paddingStr))
					} else {
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
						leftPane.WriteString(fmt.Sprintf(" %s %s %s", listSelectedStyle.Render("▸"), status, rowStyle.Render(textPart)))
					}
				} else if inst.Color != "" || inst.BgColor != "" {
					leftPane.WriteString(fmt.Sprintf(" %s %s %s", listSelectedStyle.Render("▸"), status, lipgloss.NewStyle().Bold(true).Render(styledName)))
				} else {
					leftPane.WriteString(fmt.Sprintf(" %s %s %s", listSelectedStyle.Render("▸"), status, lipgloss.NewStyle().Bold(true).Render(name)))
				}
			} else {
				if inst.FullRowColor && inst.BgColor != "" {
					// Full row background
					if _, isGradient := gradients[inst.Color]; isGradient {
						// Gradient text on full row background
						padding := listWidth - 7 - len([]rune(name))
						paddingStr := ""
						if padding > 0 {
							paddingStr = lipgloss.NewStyle().Background(lipgloss.Color(inst.BgColor)).Render(strings.Repeat(" ", padding))
						}
						gradientText := applyGradientWithBg(name, inst.Color, inst.BgColor)
						leftPane.WriteString(fmt.Sprintf("   %s %s%s", status, gradientText, paddingStr))
					} else {
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
						leftPane.WriteString(fmt.Sprintf("   %s %s", status, rowStyle.Render(textPart)))
					}
				} else {
					leftPane.WriteString(fmt.Sprintf("   %s %s", status, styledName))
				}
			}
			leftPane.WriteString("\n")

			// Show last output line for all sessions
			lastLine := m.lastLines[inst.ID]
			if lastLine == "" {
				if inst.Status == session.StatusRunning {
					lastLine = "loading..."
				} else {
					lastLine = "stopped"
				}
			}
			// Truncate to prevent line wrap (strip ANSI for length check)
			cleanLine := strings.TrimSpace(stripANSI(lastLine))
			maxLen := listWidth - 10 // Account for "     └─ " prefix
			if maxLen < 10 {
				maxLen = 10
			}
			if len(cleanLine) > maxLen {
				// Truncate the original line (keeping colors is tricky, so use clean version)
				lastLine = cleanLine[:maxLen-3] + "..."
			} else {
				lastLine = cleanLine
			}
			leftPane.WriteString(fmt.Sprintf("     └─ %s", lastLine))
			leftPane.WriteString("\n")
			if !m.compactList {
				leftPane.WriteString("\n")
			}
		}

		// Show scroll indicator at bottom
		remaining := len(m.instances) - endIdx
		if remaining > 0 {
			leftPane.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more\n", remaining)))
		}
	}

	// Build right pane (preview)
	var rightPane strings.Builder
	rightPane.WriteString("\n")
	rightPane.WriteString(titleStyle.Render(" Preview "))
	rightPane.WriteString("\n\n")

	if len(m.instances) > 0 && m.cursor < len(m.instances) {
		inst := m.instances[m.cursor]
		// Instance info
		rightPane.WriteString(dimStyle.Render(fmt.Sprintf("  Path: %s", inst.Path)))
		rightPane.WriteString("\n")
		if inst.ResumeSessionID != "" {
			rightPane.WriteString(dimStyle.Render(fmt.Sprintf("  Resume: %s", inst.ResumeSessionID[:8])))
			rightPane.WriteString("\n")
		}
		rightPane.WriteString("\n")

		// Preview content
		if m.preview != "" {
			lines := strings.Split(m.preview, "\n")

			// Filter out Claude UI elements first
			var filteredLines []string
			for _, line := range lines {
				cleanLine := stripANSI(line)
				// Skip Claude UI separator lines
				if strings.Count(cleanLine, "─") > 20 {
					continue
				}
				// Skip status bar and prompt lines
				if strings.Contains(cleanLine, "? for") || strings.Contains(cleanLine, "Context left") || strings.Contains(cleanLine, "accept edits") {
					continue
				}
				trimmed := strings.TrimSpace(cleanLine)
				if trimmed == ">" || strings.HasPrefix(trimmed, "╭") || strings.HasPrefix(trimmed, "╰") {
					continue
				}
				filteredLines = append(filteredLines, line)
			}

			// Show last N lines - simple limit (account for header: title, path, spacing)
			maxLines := contentHeight - 10
			if maxLines < 5 {
				maxLines = 5
			}
			startIdx := len(filteredLines) - maxLines
			if startIdx < 0 {
				startIdx = 0
			}
			if startIdx > 0 {
				rightPane.WriteString(dimStyle.Render("   ..."))
				rightPane.WriteString("\n")
			}
			for i := startIdx; i < len(filteredLines); i++ {
				rightPane.WriteString("  " + filteredLines[i] + "\n")
			}
		} else {
			rightPane.WriteString(dimStyle.Render("  (no output yet)"))
		}
	}

	// Style the panes with borders
	leftStyled := listPaneStyle.
		Width(listWidth).
		Height(contentHeight).
		Render(leftPane.String())

	rightStyled := previewPaneStyle.
		Width(previewWidth).
		Height(contentHeight).
		Render(rightPane.String())

	// Join panes horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)

	// Build final view
	var b strings.Builder
	b.WriteString(content)

	// Error display
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf(" Error: %v\n", m.err)))
	}

	// Status bar
	autoYesIndicator := "OFF"
	if m.autoYes {
		autoYesIndicator = "ON"
	}
	compactIndicator := "OFF"
	if m.compactList {
		compactIndicator = "ON"
	}
	b.WriteString("\n")
	statusText := helpStyle.Render(fmt.Sprintf(
		"n:new  r:resume  p:prompt  e:rename  s:start  x:stop  d:delete  c:color  l:compact[%s]  y:autoyes[%s]  ?:help  q:quit",
		compactIndicator, autoYesIndicator,
	))
	b.WriteString(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, statusText))

	return b.String()
}

func (m Model) newInstanceView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" New Session "))
	b.WriteString("\n\n")

	if m.state == stateNewPath {
		b.WriteString("  Project Path:\n")
		b.WriteString("  " + m.pathInput.View() + "\n")
	} else {
		b.WriteString(fmt.Sprintf("  Project Path: %s\n", m.pathInput.Value()))
		b.WriteString("  Session Name:\n")
		b.WriteString("  " + m.nameInput.View() + "\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter: confirm  esc: cancel"))

	return b.String()
}

func (m Model) selectSessionView() string {
	var b strings.Builder

	// Header like Claude Code
	b.WriteString("Resume Session\n")

	// Search box (visual only for now)
	boxWidth := 80
	if m.width > 20 {
		boxWidth = m.width - 10
	}
	if boxWidth > 150 {
		boxWidth = 150
	}
	b.WriteString(searchBoxStyle.Width(boxWidth).Render("⌕ Search…"))
	b.WriteString("\n\n")

	// Calculate visible window (show max 8 items)
	maxVisible := 8
	startIdx := 0
	if m.sessionCursor > maxVisible-2 {
		startIdx = m.sessionCursor - maxVisible + 2
	}
	if startIdx < 0 {
		startIdx = 0
	}

	totalItems := len(m.claudeSessions) + 1 // +1 for "new session"

	// Option 0: Start new session
	if startIdx == 0 {
		otherCount := len(m.claudeSessions)
		suffix := ""
		if otherCount > 0 {
			suffix = fmt.Sprintf(" (+%d other sessions)", otherCount)
		}

		if m.sessionCursor == 0 {
			b.WriteString(fmt.Sprintf("❯ ▶ Start new session%s\n", suffix))
		} else {
			b.WriteString(fmt.Sprintf("  Start new session%s\n", suffix))
		}
		b.WriteString("\n")
	}

	// List existing sessions
	visibleCount := 1
	for i, cs := range m.claudeSessions {
		itemIdx := i + 1

		if itemIdx < startIdx {
			continue
		}
		if visibleCount >= maxVisible {
			break
		}

		// Use last prompt (like Claude Code does)
		prompt := cs.LastPrompt
		if prompt == "" {
			prompt = cs.FirstPrompt
		}
		maxPromptLen := 80
		if m.width > 40 {
			maxPromptLen = m.width - 40
		}
		if len(prompt) > maxPromptLen {
			prompt = prompt[:maxPromptLen-3] + "..."
		}

		timeAgo := formatTimeAgo(cs.UpdatedAt)
		msgText := "messages"
		if cs.MessageCount == 1 {
			msgText = "message"
		}

		// Format like Claude Code
		if itemIdx == m.sessionCursor {
			b.WriteString(selectedPromptStyle.Render(fmt.Sprintf("❯ ▶ %s", prompt)))
			b.WriteString("\n")
			b.WriteString(metaStyle.Render(fmt.Sprintf("  %s · %d %s", timeAgo, cs.MessageCount, msgText)))
			b.WriteString("\n\n")
		} else {
			b.WriteString(fmt.Sprintf("  %s\n", prompt))
			b.WriteString(dimStyle.Render(fmt.Sprintf("  %s · %d %s", timeAgo, cs.MessageCount, msgText)))
			b.WriteString("\n\n")
		}
		visibleCount++
	}

	// Show more indicator
	remaining := totalItems - startIdx - maxVisible
	if remaining > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more sessions\n", remaining)))
	}

	return b.String()
}

var (
	searchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555555")).
			Foreground(lipgloss.Color("#666666")).
			Padding(0, 1)

	selectedPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	metaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
)

func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		return fmt.Sprintf("%d min ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	}
}

func (m Model) renameView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Rename Session "))
	b.WriteString("\n\n")

	b.WriteString("  New Name:\n")
	b.WriteString("  " + m.nameInput.View() + "\n")

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter: confirm  esc: cancel"))

	return b.String()
}

func (m Model) confirmDeleteView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Confirm Delete "))
	b.WriteString("\n\n")

	if m.deleteTarget != nil {
		b.WriteString(fmt.Sprintf("  Delete session '%s'?\n\n", m.deleteTarget.Name))
	}

	b.WriteString(helpStyle.Render("y: yes  n: no"))

	return b.String()
}

func (m Model) promptView() string {
	// Render the list view as background
	background := m.listView()
	bgLines := strings.Split(background, "\n")

	// Build the prompt box content
	var boxContent strings.Builder
	boxContent.WriteString("\n")

	if len(m.instances) > 0 {
		inst := m.instances[m.cursor]
		boxContent.WriteString(fmt.Sprintf("  Session: %s\n\n", inst.Name))
	}

	boxContent.WriteString("  Message:\n")
	boxContent.WriteString("  " + m.promptInput.View() + "\n\n")
	boxContent.WriteString(helpStyle.Render("  enter: send  esc: cancel"))
	boxContent.WriteString("\n")

	// Create the box style
	boxWidth := 60
	if m.width > 80 {
		boxWidth = m.width / 2
	}
	if boxWidth > 80 {
		boxWidth = 80
	}

	promptBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Background(lipgloss.Color("#1a1a2e")).
		Padding(0, 1).
		Width(boxWidth)

	box := promptBoxStyle.Render(titleStyle.Render(" Send Message ") + boxContent.String())
	boxLines := strings.Split(box, "\n")

	// Calculate position to center the box
	boxHeight := len(boxLines)
	startY := (m.height - boxHeight) / 2
	startX := (m.width - boxWidth - 4) / 2 // -4 for border

	// Overlay the box on the background
	for i, boxLine := range boxLines {
		bgY := startY + i
		if bgY >= 0 && bgY < len(bgLines) {
			// Get the background line
			bgLine := bgLines[bgY]
			bgRunes := []rune(stripANSI(bgLine))

			// Build new line: left part of bg + box + right part of bg
			var newLine strings.Builder

			// Left part (before box)
			if startX > 0 {
				if len(bgRunes) >= startX {
					newLine.WriteString(string(bgRunes[:startX]))
				} else {
					newLine.WriteString(string(bgRunes))
					newLine.WriteString(strings.Repeat(" ", startX-len(bgRunes)))
				}
			}

			// Box line
			newLine.WriteString(boxLine)

			// Right part (after box) - usually not needed as box fills to edge
			bgLines[bgY] = newLine.String()
		}
	}

	return strings.Join(bgLines, "\n")
}

func (m Model) helpView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Help "))
	b.WriteString("\n\n")

	help := `  Navigation:
    j/↓         Move down
    k/↑         Move up
    J/Shift+↓   Move session down (reorder)
    K/Shift+↑   Move session up (reorder)

  Actions:
    enter    Start (if stopped) and attach to session
    s        Start session without attaching
    x        Stop session
    n        Create new session
    e        Rename session
    c        Change session color
    r        Resume: select previous Claude session to continue
    p        Send prompt/message to running session
    d        Delete session
    y        Toggle auto-yes mode (--dangerously-skip-permissions)

  Other:
    ?        Show this help
    q        Quit

  In attached session:
    Ctrl+q      Detach from session (quick)
    Ctrl+b d    Detach from session (tmux default)
`
	b.WriteString(help)
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press esc or ? to close"))

	return b.String()
}

func (m Model) colorPickerView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Session Color "))
	b.WriteString("\n\n")

	if len(m.instances) > 0 {
		inst := m.instances[m.cursor]

		// Get preview colors (current cursor selection for active mode)
		previewFg := m.previewFg
		previewBg := m.previewBg
		if m.colorCursor < len(colorOptions) {
			selected := colorOptions[m.colorCursor]
			if m.colorMode == 0 {
				previewFg = selected.Color
			} else {
				previewBg = selected.Color
			}
		}

		// Show session name with preview colors
		styledName := inst.Name
		nameStyle := lipgloss.NewStyle()
		if previewBg != "" {
			nameStyle = nameStyle.Background(lipgloss.Color(previewBg))
		}
		if previewFg != "" {
			if previewFg == "auto" && previewBg != "" {
				nameStyle = nameStyle.Foreground(lipgloss.Color(getContrastColor(previewBg)))
			} else if _, isGradient := gradients[previewFg]; isGradient {
				if previewBg != "" {
					styledName = applyGradientWithBg(inst.Name, previewFg, previewBg)
				} else {
					styledName = applyGradient(inst.Name, previewFg)
				}
			} else {
				nameStyle = nameStyle.Foreground(lipgloss.Color(previewFg))
			}
		} else if previewBg != "" {
			nameStyle = nameStyle.Foreground(lipgloss.Color(getContrastColor(previewBg)))
		}
		if styledName == inst.Name {
			styledName = nameStyle.Render(inst.Name)
		}

		b.WriteString(fmt.Sprintf("  Session: %s\n", styledName))

		// Show current colors
		fgDisplay := "none"
		if inst.Color != "" {
			fgDisplay = inst.Color
		}
		bgDisplay := "none"
		if inst.BgColor != "" {
			bgDisplay = inst.BgColor
		}
		fullRowDisplay := "OFF"
		if inst.FullRowColor {
			fullRowDisplay = "ON"
		}

		// Highlight active mode
		if m.colorMode == 0 {
			b.WriteString(fmt.Sprintf("  [Szöveg: %s]  Háttér: %s\n", fgDisplay, bgDisplay))
		} else {
			b.WriteString(fmt.Sprintf("   Szöveg: %s  [Háttér: %s]\n", fgDisplay, bgDisplay))
		}
		b.WriteString(fmt.Sprintf("  Teljes sor: %s (f)\n", fullRowDisplay))
		b.WriteString(dimStyle.Render("  TAB: váltás | f: teljes sor"))
		b.WriteString("\n\n")
	}

	// Calculate max items based on mode
	maxItems := len(colorOptions)
	if m.colorMode == 1 {
		maxItems = len(colorOptions) - 15 // No gradients for background
	}

	// Calculate visible window
	maxVisible := m.height - 12
	if maxVisible < 5 {
		maxVisible = 5
	}

	startIdx := 0
	if m.colorCursor >= maxVisible {
		startIdx = m.colorCursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > maxItems {
		endIdx = maxItems
	}

	// Show scroll indicator at top
	if startIdx > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more\n", startIdx)))
	}

	for i := startIdx; i < endIdx; i++ {
		c := colorOptions[i]

		// Skip "auto" for background mode
		if m.colorMode == 1 && c.Color == "auto" {
			continue
		}

		// Create color preview
		var colorPreview string
		if c.Color == "" {
			if m.colorMode == 0 {
				colorPreview = "      none"
			} else {
				colorPreview = "       none" // Extra space for background mode
			}
		} else if c.Color == "auto" {
			colorPreview = " ✨   auto"
		} else if _, isGradient := gradients[c.Color]; isGradient {
			// Show gradient preview
			colorPreview = " " + applyGradient("████", c.Color) + " " + c.Name
		} else {
			style := lipgloss.NewStyle()
			if m.colorMode == 0 {
				style = style.Foreground(lipgloss.Color(c.Color))
				colorPreview = style.Render(" ████ ") + c.Name
			} else {
				style = style.Background(lipgloss.Color(c.Color))
				// For background, show solid block with contrast text
				textColor := getContrastColor(c.Color)
				style = style.Foreground(lipgloss.Color(textColor))
				colorPreview = style.Render("      ") + " " + c.Name
			}
		}

		if i == m.colorCursor {
			b.WriteString(fmt.Sprintf("  ❯%s\n", colorPreview))
		} else {
			b.WriteString(fmt.Sprintf("   %s\n", colorPreview))
		}
	}

	// Show scroll indicator at bottom
	remaining := maxItems - endIdx
	if remaining > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more\n", remaining)))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  enter: select  esc: cancel"))

	return b.String()
}
