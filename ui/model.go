package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/izll/claude-session-manager/session"
)

// Layout constants
const (
	ListPaneWidth        = 45  // Fixed width for session list panel
	BorderPadding        = 3   // Border and padding offset
	MinPreviewWidth      = 40  // Minimum preview panel width
	TmuxWidthOffset      = 2   // Offset to prevent line wrapping in tmux
	HeightOffset         = 8   // Height offset for UI elements
	MinContentHeight     = 10  // Minimum content height
	MinPreviewLines      = 5   // Minimum preview lines to show
	PreviewHeaderHeight  = 6   // Height of preview header area
	ColorPickerHeader    = 12  // Height of color picker header
	MinColorPickerRows   = 5   // Minimum visible color options
	SessionListMaxItems  = 8   // Max visible items in session selector
	PreviewLineCount     = 100 // Number of lines to capture for preview
	GradientColorCount   = 15  // Number of gradient options (for background exclusion)
	PromptMinWidth       = 50  // Minimum prompt input width
	PromptMaxWidth       = 70  // Maximum prompt input width
	TickInterval         = 100 * time.Millisecond // UI refresh interval
)

// state represents the current UI state
type state int

const (
	stateList state = iota
	stateNewName
	stateNewPath
	stateSelectClaudeSession // Selecting Claude session to resume
	stateConfirmDelete
	stateRename
	stateHelp
	stateColorPicker
	statePrompt // Send text to session
)

// Model represents the main TUI application state for Claude Session Manager.
// It manages multiple Claude Code instances, handles user input, and renders
// the split-pane interface with session list and preview.
type Model struct {
	instances       []*session.Instance
	storage         *session.Storage
	cursor          int
	state           state
	width           int
	height          int
	nameInput       textinput.Model
	pathInput       textinput.Model
	promptInput     textinput.Model           // Input for sending text to session
	autoYes         bool
	deleteTarget    *session.Instance
	preview         string
	err             error
	claudeSessions  []session.ClaudeSession   // Claude sessions for current instance
	sessionCursor   int                       // Cursor for Claude session selection
	pendingInstance *session.Instance         // Instance being created
	lastLines       map[string]string         // Last output line for each instance (by ID)
	prevContent     map[string]string         // Previous content hash to detect activity
	isActive        map[string]bool           // Whether instance has recent activity
	colorCursor     int                       // Cursor for color picker
	colorMode       int                       // 0 = foreground, 1 = background
	previewFg       string                    // Preview foreground color
	previewBg       string                    // Preview background color
	compactList     bool                      // No extra line between sessions
}

// tickMsg is sent periodically to update the UI
type tickMsg time.Time

// reattachMsg is sent when returning from an attached session
type reattachMsg struct{}

// NewModel creates and initializes a new TUI Model.
// It loads existing sessions from storage, sets up input fields, and
// prepares the initial state for the Bubble Tea program.
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
	promptInput.Prompt = "" // Remove the default "> " prompt
	promptInput.Cursor.SetMode(cursor.CursorStatic) // No blinking

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
		// Ensure PTY connection for running instances (for size control)
		if inst.Status == session.StatusRunning {
			inst.EnsurePty()
		}
		m.lastLines[inst.ID] = inst.GetLastLine()
	}

	// Initialize preview for first instance
	if len(instances) > 0 {
		preview, err := instances[0].GetPreview(PreviewLineCount)
		if err != nil {
			m.preview = "(error loading preview)"
		} else {
			m.preview = preview
		}
	}

	return m, nil
}

// Init implements tea.Model and returns the initial command for the program.
// It sets up the terminal appearance and starts the tick timer.
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

// tickCmd returns a command that sends a tick message after TickInterval
func tickCmd() tea.Cmd {
	return tea.Tick(TickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update implements tea.Model and handles all incoming messages.
// It delegates to specialized handlers based on the current state.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Resize selected instance's tmux pane to match preview width
		if len(m.instances) > 0 && m.cursor < len(m.instances) {
			inst := m.instances[m.cursor]
			if inst.Status == session.StatusRunning {
				tmuxWidth, tmuxHeight := m.calculateTmuxDimensions()
				inst.ResizePane(tmuxWidth, tmuxHeight)
				inst.UpdateDetachBinding(tmuxWidth, tmuxHeight)
			}
		}
		return m, nil

	case reattachMsg:
		// Request window size to refresh dimensions after reattach
		return m, tea.Batch(tea.ClearScreen, tea.EnableMouseCellMotion, tea.WindowSize())

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
		return m.handleTick()

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

// handleTick processes tick messages for periodic UI updates
func (m Model) handleTick() (tea.Model, tea.Cmd) {
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

	// Update preview for selected instance
	if len(m.instances) > 0 && m.cursor < len(m.instances) {
		preview, err := m.instances[m.cursor].GetPreview(PreviewLineCount)
		if err != nil {
			m.preview = "(error loading preview)"
		} else {
			m.preview = preview
		}
	}
	return m, tickCmd()
}

// calculatePreviewWidth returns the width for the preview panel
func (m *Model) calculatePreviewWidth() int {
	previewWidth := m.width - ListPaneWidth - BorderPadding
	if previewWidth < MinPreviewWidth {
		previewWidth = MinPreviewWidth
	}
	return previewWidth
}

// calculateTmuxDimensions returns the width and height for the tmux pane
func (m *Model) calculateTmuxDimensions() (width, height int) {
	return m.calculatePreviewWidth() - TmuxWidthOffset, m.height - HeightOffset
}

// resizeSelectedPane resizes the currently selected instance's tmux pane
func (m *Model) resizeSelectedPane() {
	if len(m.instances) > 0 && m.cursor < len(m.instances) {
		inst := m.instances[m.cursor]
		tmuxWidth, tmuxHeight := m.calculateTmuxDimensions()
		inst.ResizePane(tmuxWidth, tmuxHeight)
	}
}

// getMaxColorItems returns the maximum number of color options based on current mode
func (m *Model) getMaxColorItems() int {
	if m.colorMode == 1 {
		return len(colorOptions) - GradientColorCount
	}
	return len(colorOptions)
}
