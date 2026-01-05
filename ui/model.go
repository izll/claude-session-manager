package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/izll/agent-session-manager/session"
	"github.com/izll/agent-session-manager/updater"
)

// Update check messages
type updateCheckMsg string
type updateDoneMsg struct{ err error }

// Version info
const (
	AppName    = "asmgr"
	AppVersion = "0.3.5"
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
	PreviewLineCount     = 100  // Number of lines to capture for preview
	ScrollbackLines      = 1000 // Number of lines for scroll history
	GradientColorCount   = 15  // Number of gradient options (for background exclusion)
	PromptMinWidth       = 50  // Minimum prompt input width
	PromptMaxWidth       = 70  // Maximum prompt input width
	TickInterval         = 100 * time.Millisecond // UI refresh interval for selected
	SlowTickInterval     = 500 * time.Millisecond // UI refresh interval for others
)

// state represents the current UI state
type state int

const (
	stateProjectSelect state = iota // Project selection at startup
	stateNewProject                  // Creating new project
	stateList
	stateNewName
	stateNewPath
	stateSelectAgentSession  // Selecting agent session to resume
	stateConfirmDelete
	stateConfirmDeleteProject // Confirm project deletion
	stateConfirmImport        // Confirm import sessions
	stateConfirmStart         // Confirm auto-start session
	stateRename
	stateRenameProject // Renaming a project
	stateHelp
	stateColorPicker
	statePrompt       // Send text to session
	stateNewGroup     // Creating new group
	stateRenameGroup  // Renaming a group
	stateSelectGroup  // Assigning session to group
	stateSelectAgent  // Selecting agent type for new session
	stateCustomCmd    // Entering custom command
	stateError        // Showing error overlay
	stateUpdating     // Downloading update
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
	agentSessions   []session.AgentSession    // Agent sessions for current instance
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
	hideStatusLines bool                      // Hide last output line under sessions
	splitView       bool                      // Split preview mode
	markedSessionID string                    // Session ID marked for split view
	splitFocus      int                       // 0 = selected (bottom), 1 = pinned (top)
	groups          []*session.Group          // Session groups
	groupInput      textinput.Model           // Input for group name
	groupCursor     int                       // Cursor for group selection
	visibleItems    []visibleItem             // Flattened list of visible items (groups + sessions)
	pendingGroupID  string                    // Group ID for new session creation
	editingGroup    *session.Group            // Group being edited in color picker (nil = editing session)
	agentCursor     int                       // Cursor for agent selection
	pendingAgent    session.AgentType         // Agent type for new session
	customCmdInput  textinput.Model           // Input for custom command
	tickCount       int                       // Counter for slow tick (update others every 5th tick)
	updateAvailable string                    // New version available (empty if up to date)
	previewScroll   int                       // Preview scroll offset (0 = bottom, positive = scroll up)
	scrollContent   string                    // Extended content for scrolling (fetched on demand)
	projects        []*session.Project        // Available projects
	projectCursor   int                       // Cursor for project selection
	activeProject   *session.Project          // Currently active project (nil = default)
	projectInput    textinput.Model           // Input for project name
	deleteProjectTarget *session.Project      // Project being deleted
	importTarget        *session.Project      // Project to import sessions into
	previousState       state                 // Previous state to return to from error dialog
}

// visibleItem represents an item in the flattened list view (group header or session)
type visibleItem struct {
	isGroup  bool              // true if this is a group header
	group    *session.Group    // The group (if isGroup is true)
	instance *session.Instance // The session instance (if isGroup is false)
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

	// Don't load sessions yet - wait until project is selected
	// instances, groups, settings will be loaded in switchToProject

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

	groupInput := textinput.New()
	groupInput.Placeholder = "Group name"
	groupInput.CharLimit = 50

	customCmdInput := textinput.New()
	customCmdInput.Placeholder = "command --flags"
	customCmdInput.CharLimit = 500

	projectInput := textinput.New()
	projectInput.Placeholder = "Project name"
	projectInput.CharLimit = 50

	// Load projects
	projectsData, err := storage.LoadProjects()
	if err != nil {
		return Model{}, err
	}

	m := Model{
		instances:       []*session.Instance{}, // Empty until project selected
		storage:         storage,
		state:           stateProjectSelect, // Start with project selection
		nameInput:       nameInput,
		pathInput:       pathInput,
		promptInput:     promptInput,
		groupInput:      groupInput,
		customCmdInput:  customCmdInput,
		projectInput:    projectInput,
		projects:        projectsData.Projects,
		projectCursor:   0, // Default to first project or "Continue without project"
		groups:          []*session.Group{}, // Empty until project selected
		lastLines:       make(map[string]string),
		prevContent:     make(map[string]string),
		isActive:        make(map[string]bool),
	}

	// Sessions will be loaded when user selects a project via switchToProject

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
		tea.SetWindowTitle("Agent Session Manager"),
		tea.EnableMouseCellMotion,
		checkForUpdateCmd(),
	)
}

// checkForUpdateCmd checks for updates in the background
func checkForUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		newVersion := updater.CheckForUpdate(AppVersion)
		return updateCheckMsg(newVersion)
	}
}

// runUpdateCmd downloads and installs the update
func runUpdateCmd(version string) tea.Cmd {
	return func() tea.Msg {
		err := updater.DownloadAndInstall(version)
		return updateDoneMsg{err: err}
	}
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
		if inst := m.getSelectedInstance(); inst != nil && inst.Status == session.StatusRunning {
			tmuxWidth, tmuxHeight := m.calculateTmuxDimensions()
			inst.ResizePane(tmuxWidth, tmuxHeight)
			inst.UpdateDetachBinding(tmuxWidth, tmuxHeight)
		}
		return m, nil

	case reattachMsg:
		// Request window size to refresh dimensions after reattach
		return m, tea.Batch(tea.ClearScreen, tea.EnableMouseCellMotion, tea.WindowSize())

	case updateCheckMsg:
		if string(msg) != "" {
			m.updateAvailable = string(msg)
		}
		return m, nil

	case updateDoneMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Update successful - show message and quit so user can restart
			m.err = fmt.Errorf("updated to %s - please restart", m.updateAvailable)
		}
		m.state = stateList
		return m, nil

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
		case stateProjectSelect:
			return m.handleProjectSelectKeys(msg)
		case stateNewProject:
			return m.handleNewProjectKeys(msg)
		case stateConfirmDeleteProject:
			return m.handleConfirmDeleteProjectKeys(msg)
		case stateConfirmImport:
			return m.handleConfirmImportKeys(msg)
		case stateRenameProject:
			return m.handleRenameProjectKeys(msg)
		case stateList:
			return m.handleListKeys(msg)
		case stateNewName:
			return m.handleNewNameKeys(msg)
		case stateNewPath:
			return m.handleNewPathKeys(msg)
		case stateSelectAgentSession:
			return m.handleSelectSessionKeys(msg)
		case stateConfirmDelete:
			return m.handleConfirmDeleteKeys(msg)
		case stateConfirmStart:
			return m.handleConfirmStartKeys(msg)
		case stateRename:
			return m.handleRenameKeys(msg)
		case stateHelp:
			return m.handleHelpKeys(msg)
		case stateColorPicker:
			return m.handleColorPickerKeys(msg)
		case statePrompt:
			return m.handlePromptKeys(msg)
		case stateNewGroup:
			return m.handleNewGroupKeys(msg)
		case stateRenameGroup:
			return m.handleRenameGroupKeys(msg)
		case stateSelectGroup:
			return m.handleSelectGroupKeys(msg)
		case stateSelectAgent:
			return m.handleSelectAgentKeys(msg)
		case stateCustomCmd:
			return m.handleCustomCmdKeys(msg)
		case stateError:
			return m.handleErrorKeys(msg)
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
	if m.state == stateNewGroup || m.state == stateRenameGroup {
		m.groupInput, cmd = m.groupInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.state == stateCustomCmd {
		m.customCmdInput, cmd = m.customCmdInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.state == stateNewProject || m.state == stateRenameProject {
		m.projectInput, cmd = m.projectInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleTick processes tick messages for periodic UI updates
func (m Model) handleTick() (tea.Model, tea.Cmd) {
	// Skip heavy processing during dialogs - only update in list view
	if m.state != stateList {
		return m, tickCmd()
	}

	m.tickCount++
	slowTick := m.tickCount%5 == 0 // Every 5th tick (500ms) for non-selected

	selectedInst := m.getSelectedInstance()

	// Update instance statuses and last lines
	for _, inst := range m.instances {
		// Only update non-selected instances on slow tick
		isSelected := selectedInst != nil && inst.ID == selectedInst.ID
		if !isSelected && !slowTick {
			continue
		}

		inst.UpdateStatus()
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

	// Update preview for selected instance
	if selectedInst != nil {
		preview, err := selectedInst.GetPreview(PreviewLineCount)
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
	if inst := m.getSelectedInstance(); inst != nil {
		tmuxWidth, tmuxHeight := m.calculateTmuxDimensions()
		inst.ResizePane(tmuxWidth, tmuxHeight)
	}
}

// getFilteredColorOptions returns color options filtered for current mode
func (m *Model) getFilteredColorOptions() []ColorOption {
	var filtered []ColorOption
	for _, c := range colorOptions {
		if m.colorMode == 1 {
			// Skip gradients for background mode
			if _, isGradient := gradients[c.Color]; isGradient {
				continue
			}
			// Skip "auto" for background mode
			if c.Color == "auto" {
				continue
			}
		}
		filtered = append(filtered, c)
	}
	return filtered
}

// getMaxColorItems returns the maximum number of color options based on current mode
func (m *Model) getMaxColorItems() int {
	return len(m.getFilteredColorOptions())
}

// buildVisibleItems builds the flattened list of visible items (groups + sessions)
func (m *Model) buildVisibleItems() {
	m.visibleItems = []visibleItem{}

	// Get sessions by group
	groupedSessions := make(map[string][]*session.Instance)
	var ungroupedSessions []*session.Instance

	for _, inst := range m.instances {
		if inst.GroupID == "" {
			ungroupedSessions = append(ungroupedSessions, inst)
		} else {
			groupedSessions[inst.GroupID] = append(groupedSessions[inst.GroupID], inst)
		}
	}

	// Add groups and their sessions
	for _, group := range m.groups {
		m.visibleItems = append(m.visibleItems, visibleItem{
			isGroup: true,
			group:   group,
		})
		if !group.Collapsed {
			for _, inst := range groupedSessions[group.ID] {
				m.visibleItems = append(m.visibleItems, visibleItem{
					isGroup:  false,
					instance: inst,
				})
			}
		}
	}

	// Add ungrouped sessions
	for _, inst := range ungroupedSessions {
		m.visibleItems = append(m.visibleItems, visibleItem{
			isGroup:  false,
			instance: inst,
		})
	}
}

// getSelectedInstance returns the currently selected instance, or nil if a group is selected
// Works in both grouped and non-grouped modes
func (m *Model) getSelectedInstance() *session.Instance {
	if len(m.groups) > 0 {
		m.buildVisibleItems()
		if m.cursor < 0 || m.cursor >= len(m.visibleItems) {
			return nil
		}
		item := m.visibleItems[m.cursor]
		if item.isGroup {
			return nil
		}
		return item.instance
	}
	// Non-grouped mode
	if m.cursor < 0 || m.cursor >= len(m.instances) {
		return nil
	}
	return m.instances[m.cursor]
}

// getSelectedGroup returns the currently selected group, or nil if a session is selected
func (m *Model) getSelectedGroup() *session.Group {
	if m.cursor < 0 || m.cursor >= len(m.visibleItems) {
		return nil
	}
	item := m.visibleItems[m.cursor]
	if !item.isGroup {
		return nil
	}
	return item.group
}

// getSessionsInGroup returns all sessions that belong to a group
func (m *Model) getSessionsInGroup(groupID string) []*session.Instance {
	var sessions []*session.Instance
	for _, inst := range m.instances {
		if inst.GroupID == groupID {
			sessions = append(sessions, inst)
		}
	}
	return sessions
}

// getCurrentGroupID returns the group ID of the currently selected item
// Returns the group ID if a group is selected, or the group ID of the selected session
func (m *Model) getCurrentGroupID() string {
	if len(m.groups) == 0 {
		return ""
	}
	m.buildVisibleItems()
	if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
		item := m.visibleItems[m.cursor]
		if item.isGroup {
			return item.group.ID
		}
		if item.instance != nil {
			return item.instance.GroupID
		}
	}
	return ""
}

// isLastInGroup checks if the session at the given visibleItems index is the last one in its group
func (m *Model) isLastInGroup(index int) bool {
	if index < 0 || index >= len(m.visibleItems) {
		return true
	}
	item := m.visibleItems[index]
	if item.isGroup || item.instance == nil {
		return true
	}
	groupID := item.instance.GroupID
	if groupID == "" {
		// Ungrouped session - check if next is also ungrouped or end of list
		if index+1 >= len(m.visibleItems) {
			return true
		}
		nextItem := m.visibleItems[index+1]
		return nextItem.isGroup || nextItem.instance.GroupID != ""
	}
	// Grouped session - check if next is in same group
	if index+1 >= len(m.visibleItems) {
		return true
	}
	nextItem := m.visibleItems[index+1]
	return nextItem.isGroup || nextItem.instance.GroupID != groupID
}

// switchToProject switches to a different project and reloads data
func (m *Model) switchToProject(project *session.Project) error {
	var projectID string
	projectName := "default"
	if project != nil {
		projectID = project.ID
		projectName = project.Name
	}

	// Check if project is already locked by another instance
	locked, pid := m.storage.IsProjectLocked(projectID)
	if locked {
		return fmt.Errorf("project '%s' is already open (PID: %d)", projectName, pid)
	}

	// Release current lock before switching
	m.storage.UnlockProject()

	// Switch storage to new project
	if err := m.storage.SetActiveProject(projectID); err != nil {
		return err
	}

	// Lock the new project
	if err := m.storage.LockProject(projectID); err != nil {
		return err
	}

	// Load the new project's sessions
	instances, groups, settings, err := m.storage.LoadAllWithSettings()
	if err != nil {
		return err
	}

	m.activeProject = project
	m.instances = instances
	m.groups = groups
	m.cursor = settings.Cursor
	m.compactList = settings.CompactList
	m.hideStatusLines = settings.HideStatusLines
	m.splitView = settings.SplitView
	m.markedSessionID = settings.MarkedSessionID
	m.splitFocus = settings.SplitFocus

	// Reset maps
	m.lastLines = make(map[string]string)
	m.prevContent = make(map[string]string)
	m.isActive = make(map[string]bool)

	// Initialize status and last lines for all instances
	for _, inst := range m.instances {
		inst.UpdateStatus()
		m.lastLines[inst.ID] = inst.GetLastLine()
	}

	// Initialize preview
	if len(m.instances) > 0 {
		if m.cursor >= len(m.instances) {
			m.cursor = 0
		}
		preview, err := m.instances[m.cursor].GetPreview(PreviewLineCount)
		if err != nil {
			m.preview = "(error loading preview)"
		} else {
			m.preview = preview
		}
	} else {
		m.preview = ""
	}

	return nil
}
