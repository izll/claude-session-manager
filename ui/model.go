package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/izll/agent-session-manager/session"
	"github.com/izll/agent-session-manager/updater"
)

// Update check messages
type updateCheckMsg string
type updateDoneMsg struct{ err error }
type debDownloadDoneMsg struct {
	err     error
	debPath string
}
type rpmDownloadDoneMsg struct {
	err     error
	rpmPath string
}

// History loading message (for global search)
type historyLoadedMsg struct {
	err error
}

// Version info
const (
	AppName    = "asmgr"
	AppVersion = "0.7.5"
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
	FavoritesGroupID     = "__favorites__"        // Virtual group ID for favorites
)

// state represents the current UI state
type state int

const (
	stateProjectSelect state = iota // Project selection at startup
	stateNewProject                 // Creating new project
	stateList
	stateNewName
	stateNewPath
	stateSelectAgentSession  // Selecting agent session to resume
	stateConfirmDelete
	stateConfirmStop          // Confirm session stop
	stateConfirmDeleteProject // Confirm project deletion
	stateConfirmImport        // Confirm import sessions
	stateSelectStartMode      // Select replace or parallel session start
	stateConfirmStart         // Confirm auto-start session
	stateRename
	stateRenameProject // Renaming a project
	stateHelp
	stateColorPicker
	statePrompt       // Send text to session
	stateNewGroup     // Creating new group
	stateRenameGroup  // Renaming a group
	stateSelectGroup  // Assigning session to group
	stateSelectAgent    // Selecting agent type for new session
	stateCustomCmd      // Entering custom command
	stateError          // Showing error overlay
	stateConfirmUpdate  // Confirming update action
	stateCheckingUpdate // Checking for updates
	stateUpdating       // Downloading update
	stateDownloadingDeb // Downloading .deb package for dpkg install
	stateDownloadingRpm // Downloading .rpm package for rpm install
	stateUpdateSuccess  // Showing successful update message
	stateNotes          // Editing session notes
	stateNewTabChoice   // Choosing between Agent or Terminal tab
	stateNewTabAgent    // Selecting agent type for new tab
	stateNewTab         // Creating new tmux tab/window with name
	stateRenameTab      // Renaming tmux tab/window
	stateDeleteChoice     // Choosing between deleting session or tab
	stateConfirmDeleteTab // Confirming tab deletion
	stateStopChoice       // Choosing between stopping session or tab
	stateConfirmStopTab   // Confirming tab stop
	stateConfirmYolo      // Confirming YOLO mode toggle
	stateSearch              // Searching/filtering sessions
	stateGlobalSearchLoading // Loading history for global search
	stateGlobalSearch        // Global history search across all agents
	stateForkDialog          // Fork session dialog (name + destination)
	stateGlobalSearchAction      // Action selection for global search result (open/new session/new tab)
	stateGlobalSearchConfirmJump // Confirm jump to existing session
	stateGlobalSearchNewName     // Entering name for new session from global search
	stateGlobalSearchSelectMatch // Selecting from multiple matching sessions/tabs
)

// Model represents the main TUI application state for Agent Session Manager.
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
	promptInput     textarea.Model            // Textarea for sending multi-line text to session
	promptSuggestion string                    // Autocomplete suggestion from agent
	autoYes         bool
	deleteTarget    *session.Instance
	stopTarget      *session.Instance
	yoloTarget      *session.Instance // Instance for YOLO confirmation
	yoloWindowIndex int               // Window index for YOLO toggle (0 = main, >0 = tab)
	yoloNewState    bool              // New YOLO state (true = enable, false = disable)
	preview         string
	err             error
	successMsg      string                        // Success message to display
	agentSessions       []session.AgentSession    // Agent sessions for current instance
	resumeAgentType     session.AgentType         // Agent type for resume (active tab's agent)
	resumeWindowIndex   int                       // Window index for resume (active tab's index)
	sessionCursor       int                       // Cursor for Claude session selection
	pendingInstance     *session.Instance         // Instance being created
	isParallelSession   bool                      // True if creating parallel session (don't show resume)
	parallelOriginalID  string                    // Original instance ID when creating parallel session
	lastLines           map[string]string                   // Last output line for each instance (by ID)
	prevContent        map[string]string                            // Previous content hash to detect activity
	isActive           map[string]bool                              // Whether instance has recent activity
	activityState      map[string]session.SessionActivity           // Activity state (idle/busy/waiting)
	windowActivityState map[string]map[int]session.SessionActivity  // Window-level activity (session ID -> window index -> activity)
	colorCursor     int                       // Cursor for color picker
	colorMode       int                       // 0 = foreground, 1 = background
	previewFg       string                    // Preview foreground color
	previewBg       string                    // Preview background color
	compactList     bool                      // No extra line between sessions
	hideStatusLines bool                      // Hide last output line under sessions
	showAgentIcons  bool                      // Show agent type icons in session list
	splitView          bool                      // Split preview mode
	markedSessionID    string                    // Session ID marked for split view
	markedVisibleIndex int                       // Visual index for pinned navigation (handles duplicates)
	splitFocus         int                       // 0 = selected (bottom), 1 = pinned (top)
	groups             []*session.Group          // Session groups
	favoritesCollapsed bool                      // Whether favorites group is collapsed
	groupInput         textinput.Model           // Input for group name
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
	helpScroll      int                       // Help view scroll offset (0 = top, positive = scroll down)
	projects        []*session.Project        // Available projects
	projectCursor   int                       // Cursor for project selection
	activeProject   *session.Project          // Currently active project (nil = default)
	projectInput    textinput.Model           // Input for project name
	deleteProjectTarget *session.Project      // Project being deleted
	importTarget        *session.Project      // Project to import sessions into
	previousState       state                 // Previous state to return to from error dialog
	notesInput          textarea.Model        // Textarea for editing session notes
	notesWindowIndex    int                   // Window index for notes editing (-1 = session, >=0 = tab)
	newTabIsAgent       bool                  // Whether new tab should run agent (true) or shell (false)
	newTabAgent         session.AgentType     // Agent type for new tab
	newTabAgentCursor   int                   // Cursor for agent selection in new tab dialog

	// Diff pane
	diffPane       *DiffPane // Diff display component
	showDiff       bool      // Show diff tab instead of preview

	// Fork dialog
	forkNameInput textinput.Model    // Input for fork name
	forkToTab     bool               // true = fork to new tab, false = fork to new session
	forkTarget    *session.Instance  // Session being forked

	// Search
	searchInput  textinput.Model // Search input field
	searchQuery  string          // Active search query (for filtering)
	searchActive bool            // Whether search filter is active

	// Global Search (multi-agent history)
	globalSearchInput          textinput.Model                  // Search input field
	globalSearchResults        []session.HistoryEntry           // Search results
	globalSearchCursor         int                              // Cursor in results list
	globalSearchExpanded       int                              // Expanded result index (-1 = none)
	historyIndex               *session.HistoryIndex            // History index for all agents
	globalSearchConversation   []session.ConversationMessage    // Cached conversation for preview
	globalSearchScroll         int                              // Scroll position in conversation preview
	globalSearchLastCursor     int                              // Last cursor position (to detect changes)
	globalSearchLastQuery      string                           // Last search query (for debounce)
	globalSearchPendingQuery   string                           // Query waiting to be searched
	globalSearchDebounceActive bool                             // Whether debounce timer is active
	globalSearchConvLoading    bool                             // Whether conversation is loading

	// Global search action dialog
	globalSearchActionCursor    int                              // Cursor for action selection (0=new session, 1=to group, 2=as tab)
	globalSearchSelectedEntry   *session.HistoryEntry            // Selected history entry for action
	globalSearchMatchedSession  *session.Instance                // Matched session for confirm jump dialog
	globalSearchMatchedTabIndex int                              // Matched tab index (-1 = main session, >=0 = tab index)
	globalSearchMatches         []globalSearchMatch              // All matching sessions/tabs for selection
	globalSearchMatchCursor     int                              // Cursor for match selection
}

// globalSearchMatch represents a matched session/tab for selection
type globalSearchMatch struct {
	Session  *session.Instance
	TabIndex int    // -1 = main session, >=0 = tab index
	TabName  string // Display name for the tab
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

// globalSearchDebounceMsg triggers delayed search after typing stops
type globalSearchDebounceMsg struct{}

// globalSearchConvLoadedMsg indicates conversation finished loading
type globalSearchConvLoadedMsg struct {
	conversation []session.ConversationMessage
	cursorPos    int // cursor position when loading started
}

// NewModel creates and initializes a new TUI Model.
// It loads existing sessions from storage, sets up input fields, and
// prepares the initial state for the Bubble Tea program.
func NewModel() (Model, error) {
	storage, err := session.NewStorage()
	if err != nil {
		return Model{}, err
	}

	nameInput := textinput.New()
	nameInput.Placeholder = "Session name"
	nameInput.CharLimit = 50

	pathInput := textinput.New()
	pathInput.Placeholder = "/path/to/project"
	pathInput.CharLimit = 256

	promptInput := textarea.New()
	promptInput.Placeholder = "Enter message to send..."
	promptInput.CharLimit = 5000
	promptInput.ShowLineNumbers = false
	promptInput.Prompt = ""
	promptInput.SetHeight(10)
	promptInput.SetWidth(60)

	groupInput := textinput.New()
	groupInput.Placeholder = "Group name"
	groupInput.CharLimit = 50

	customCmdInput := textinput.New()
	customCmdInput.Placeholder = "command --flags"
	customCmdInput.CharLimit = 500

	projectInput := textinput.New()
	projectInput.Placeholder = "Project name"
	projectInput.CharLimit = 50

	notesInput := textarea.New()
	notesInput.Placeholder = "Add notes about this session..."
	notesInput.CharLimit = 5000
	notesInput.ShowLineNumbers = false
	notesInput.Prompt = ""
	notesInput.SetWidth(70)
	notesInput.SetHeight(9)

	searchInput := textinput.New()
	searchInput.Placeholder = "Search..."
	searchInput.CharLimit = 100
	searchInput.Prompt = "/ "
	searchInput.Width = 200 // Will be adjusted on resize

	globalSearchInput := textinput.New()
	globalSearchInput.Placeholder = "Search all agents..."
	globalSearchInput.CharLimit = 200
	globalSearchInput.Prompt = ""
	globalSearchInput.Width = 200 // Will be adjusted on resize

	forkNameInput := textinput.New()
	forkNameInput.Placeholder = "Fork name"
	forkNameInput.CharLimit = 50

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
		notesInput:      notesInput,
		searchInput:     searchInput,
		globalSearchInput:   globalSearchInput,
		globalSearchExpanded: -1,
		historyIndex:        session.NewHistoryIndex(),
		forkNameInput:       forkNameInput,
		projects:        projectsData.Projects,
		projectCursor:   0,
		groups:          []*session.Group{},
		lastLines:           make(map[string]string),
		prevContent:         make(map[string]string),
		isActive:            make(map[string]bool),
		activityState:       make(map[string]session.SessionActivity),
		windowActivityState: make(map[string]map[int]session.SessionActivity),
		diffPane:            NewDiffPane(),
		updateAvailable:     updater.GetCachedAvailableUpdate(), // Load cached update
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
		tea.SetWindowTitle("Agent Session Manager"),
		tea.EnableMouseCellMotion,
		checkForUpdateCmd(),
	)
}

// loadHistoryCmd loads history index for global search asynchronously
func (m *Model) loadHistoryCmd() tea.Cmd {
	// Pass instances for terminal search
	m.historyIndex.SetInstances(m.instances)
	return func() tea.Msg {
		err := m.historyIndex.Load()
		return historyLoadedMsg{err: err}
	}
}

// checkForUpdateCmd checks for updates in the background (once per day, after 30s delay)
func checkForUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		// Wait 30 seconds after startup before checking
		time.Sleep(30 * time.Second)
		// Only check if 24 hours have passed since last check
		if !updater.ShouldCheckForUpdate() {
			return updateCheckMsg("")
		}
		newVersion := updater.CheckForUpdate(AppVersion)
		updater.SaveLastCheckTime()
		return updateCheckMsg(newVersion)
	}
}

// forceCheckForUpdateCmd always checks for updates (user requested)
func forceCheckForUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		newVersion := updater.CheckForUpdate(AppVersion)
		updater.SaveLastCheckTime()
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

// runDebDownload downloads the .deb package
func runDebDownload(version string) tea.Cmd {
	return func() tea.Msg {
		debPath, err := updater.DownloadDeb(version)
		return debDownloadDoneMsg{err: err, debPath: debPath}
	}
}

// runRpmDownload downloads the .rpm package
func runRpmDownload(version string) tea.Cmd {
	return func() tea.Msg {
		rpmPath, err := updater.DownloadRpm(version)
		return rpmDownloadDoneMsg{err: err, rpmPath: rpmPath}
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
		// Update search input widths
		m.searchInput.Width = msg.Width - 6 // Account for prompt and padding
		m.globalSearchInput.Width = ListPaneWidth - 6
		// Resize selected instance's tmux pane to match preview width
		if inst := m.getSelectedInstance(); inst != nil && inst.Status == session.StatusRunning {
			tmuxWidth, tmuxHeight := m.calculateTmuxDimensions()
			inst.ResizePane(tmuxWidth, tmuxHeight)
			inst.UpdateDetachBinding(tmuxWidth, tmuxHeight)
		}
		return m, nil

	case historyLoadedMsg:
		// History loaded - transition to global search
		if msg.err != nil {
			m.err = msg.err
			m.previousState = stateList
			m.state = stateError
			return m, nil
		}
		m.state = stateGlobalSearch
		m.globalSearchInput.Focus()

		// Re-run search if there was a query (after Ctrl+R reload)
		query := strings.TrimSpace(m.globalSearchInput.Value())
		if query != "" {
			m.globalSearchResults = m.historyIndex.Search(query)
			m.globalSearchCursor = 0
			m.globalSearchLastQuery = query
			// Load conversation for first result
			return m, m.loadConversationAsync()
		}
		return m, nil

	case reattachMsg:
		// Request window size to refresh dimensions after reattach
		return m, tea.Batch(tea.ClearScreen, tea.EnableMouseCellMotion, tea.WindowSize())

	case updateCheckMsg:
		newVersion := string(msg)
		if newVersion != "" {
			m.updateAvailable = newVersion
			updater.SaveAvailableUpdate(newVersion) // Cache for next startup
		}

		// If we're actively checking for updates (user pressed U)
		if m.state == stateCheckingUpdate {
			if newVersion != "" {
				// Update available - start download based on package type
				if updater.IsPackageManaged() {
					// Check if deb
					if _, err := os.Stat("/var/lib/dpkg/info/asmgr.list"); err == nil {
						m.state = stateDownloadingDeb
						return m, runDebDownload(newVersion)
					}
					// Otherwise rpm
					m.state = stateDownloadingRpm
					return m, runRpmDownload(newVersion)
				}
				// Tarball update
				m.state = stateUpdating
				return m, runUpdateCmd(newVersion)
			} else {
				// Already up to date
				m.err = fmt.Errorf("already up to date (v%s)", AppVersion)
				m.previousState = stateList
				m.state = stateError
				return m, nil
			}
		}
		return m, nil

	case debDownloadDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.previousState = m.state
			m.state = stateError
			return m, nil
		}
		// Deb downloaded - now run sudo dpkg -i via tea.ExecProcess
		cmd := exec.Command("sudo", "dpkg", "-i", msg.debPath)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			// Clean up temp file
			os.Remove(msg.debPath)
			if err != nil {
				return updateDoneMsg{err: fmt.Errorf("dpkg installation failed: %w", err)}
			}
			return updateDoneMsg{err: nil}
		})

	case rpmDownloadDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.previousState = m.state
			m.state = stateError
			return m, nil
		}
		// Rpm downloaded - now run sudo rpm -Uvh via tea.ExecProcess
		cmd := exec.Command("sudo", "rpm", "-Uvh", msg.rpmPath)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			// Clean up temp file
			os.Remove(msg.rpmPath)
			if err != nil {
				return updateDoneMsg{err: fmt.Errorf("rpm installation failed: %w", err)}
			}
			return updateDoneMsg{err: nil}
		})

	case updateDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.previousState = m.state
			m.state = stateError
		} else {
			// Update successful - show success message and clear cache
			m.successMsg = fmt.Sprintf("Updated to %s - please restart", m.updateAvailable)
			updater.ClearAvailableUpdate()
			m.updateAvailable = ""
			m.previousState = m.state
			m.state = stateUpdateSuccess
		}
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

	case globalSearchDebounceMsg:
		return m.handleGlobalSearchDebounce()

	case globalSearchConvLoadedMsg:
		return m.handleGlobalSearchConvLoaded(msg)

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
		case stateConfirmStop:
			return m.handleConfirmStopKeys(msg)
		case stateSelectStartMode:
			return m.handleSelectStartModeKeys(msg)
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
		case stateConfirmUpdate:
			return m.handleConfirmUpdateKeys(msg)
		case stateUpdateSuccess:
			return m.handleUpdateSuccessKeys(msg)
		case stateNotes:
			return m.handleNotesKeys(msg)
		case stateNewTabChoice:
			return m.handleNewTabChoiceKeys(msg)
		case stateNewTabAgent:
			return m.handleNewTabAgentKeys(msg)
		case stateNewTab:
			return m.handleNewTabKeys(msg)
		case stateRenameTab:
			return m.handleRenameTabKeys(msg)
		case stateDeleteChoice:
			return m.handleDeleteChoiceKeys(msg)
		case stateConfirmDeleteTab:
			return m.handleConfirmDeleteTabKeys(msg)
		case stateStopChoice:
			return m.handleStopChoiceKeys(msg)
		case stateConfirmStopTab:
			return m.handleConfirmStopTabKeys(msg)
		case stateConfirmYolo:
			return m.handleConfirmYoloKeys(msg)
		case stateSearch:
			return m.handleSearchKeys(msg)
		case stateGlobalSearchLoading:
			// Only ESC to cancel loading
			if msg.String() == "esc" {
				m.state = stateList
				return m, nil
			}
			return m, nil
		case stateGlobalSearch:
			return m.handleGlobalSearchKeys(msg)
		case stateForkDialog:
			return m.handleForkDialogKeys(msg)
		case stateGlobalSearchAction:
			return m.handleGlobalSearchActionKeys(msg)
		case stateGlobalSearchConfirmJump:
			return m.handleGlobalSearchConfirmJumpKeys(msg)
		case stateGlobalSearchNewName:
			return m.handleGlobalSearchNewNameKeys(msg)
		case stateGlobalSearchSelectMatch:
			return m.handleGlobalSearchSelectMatchKeys(msg)
		}
	}

	if m.state == stateNewName || m.state == stateRename || m.state == stateNewTab || m.state == stateRenameTab {
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
	if m.state == stateNotes {
		m.notesInput, cmd = m.notesInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.state == stateSearch {
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.state == stateGlobalSearch {
		m.globalSearchInput, cmd = m.globalSearchInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.state == stateForkDialog {
		m.forkNameInput, cmd = m.forkNameInput.Update(msg)
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

			// Detect detailed activity state (busy/waiting/idle) across all followed windows
			m.activityState[inst.ID] = inst.DetectAggregatedActivity()

			// Detect per-window activity for status line coloring
			if m.windowActivityState[inst.ID] == nil {
				m.windowActivityState[inst.ID] = make(map[int]session.SessionActivity)
			}
			// Main window (0)
			m.windowActivityState[inst.ID][0] = inst.DetectActivityForWindow(0)
			// Followed windows
			for _, fw := range inst.FollowedWindows {
				m.windowActivityState[inst.ID][fw.Index] = inst.DetectActivityForWindow(fw.Index)
			}
		} else {
			m.isActive[inst.ID] = false
			m.activityState[inst.ID] = session.ActivityIdle
			m.windowActivityState[inst.ID] = nil
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

		// Update diff content if showing diff tab (only on slow tick to avoid git overload)
		if m.showDiff && slowTick {
			m.diffPane.SetDiff(selectedInst)
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

	// Collect favorite sessions first
	var favoriteSessions []*session.Instance

	for _, inst := range m.instances {
		if inst.Favorite {
			// Skip if doesn't match search
			if m.searchActive && !m.matchesSearch(inst) {
				continue
			}
			favoriteSessions = append(favoriteSessions, inst)
		}
	}

	// Add favorites group at top if there are favorites
	if len(favoriteSessions) > 0 {
		favGroup := &session.Group{
			ID:        FavoritesGroupID,
			Name:      "Favorites",
			Collapsed: m.favoritesCollapsed,
		}
		m.visibleItems = append(m.visibleItems, visibleItem{
			isGroup: true,
			group:   favGroup,
		})
		if !m.favoritesCollapsed {
			for _, inst := range favoriteSessions {
				m.visibleItems = append(m.visibleItems, visibleItem{
					isGroup:  false,
					instance: inst,
				})
			}
		}
		// Add separator after favorites group
		m.visibleItems = append(m.visibleItems, visibleItem{
			isGroup:  false,
			instance: nil, // nil instance = separator
		})
	}

	// Get sessions by group (filtered if search is active)
	// Favorites also appear in their original groups
	groupedSessions := make(map[string][]*session.Instance)
	var ungroupedSessions []*session.Instance

	for _, inst := range m.instances {
		// Skip if doesn't match search
		if m.searchActive && !m.matchesSearch(inst) {
			continue
		}
		if inst.GroupID == "" {
			ungroupedSessions = append(ungroupedSessions, inst)
		} else {
			groupedSessions[inst.GroupID] = append(groupedSessions[inst.GroupID], inst)
		}
	}

	// Add groups and their sessions
	for _, group := range m.groups {
		// Skip empty groups when search is active
		if m.searchActive && len(groupedSessions[group.ID]) == 0 {
			continue
		}
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

// hasFavorites returns true if there are any favorite sessions
func (m *Model) hasFavorites() bool {
	for _, inst := range m.instances {
		if inst.Favorite {
			return true
		}
	}
	return false
}

// getSelectedInstance returns the currently selected instance, or nil if a group is selected
// Works in both grouped and non-grouped modes
func (m *Model) getSelectedInstance() *session.Instance {
	if len(m.groups) > 0 || m.hasFavorites() {
		m.buildVisibleItems()
		if m.cursor < 0 || m.cursor >= len(m.visibleItems) {
			return nil
		}
		item := m.visibleItems[m.cursor]
		if item.isGroup || item.instance == nil {
			return nil
		}
		return item.instance
	}
	// Non-grouped mode - use filtered list when search is active
	if m.searchActive {
		filtered := m.getFilteredInstances()
		if m.cursor < 0 || m.cursor >= len(filtered) {
			return nil
		}
		return filtered[m.cursor]
	}
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

	// Check if next item exists
	if index+1 >= len(m.visibleItems) {
		return true
	}
	nextItem := m.visibleItems[index+1]

	// Separator or group = last in current context
	if nextItem.isGroup || nextItem.instance == nil {
		return true
	}

	// For favorites: check if both current and next are favorites
	if item.instance.Favorite {
		// Last in favorites group if next is not a favorite
		return !nextItem.instance.Favorite
	}

	// For regular groups
	groupID := item.instance.GroupID
	if groupID == "" {
		// Ungrouped session - last if next has a group or is a favorite
		return nextItem.instance.GroupID != "" || nextItem.instance.Favorite
	}

	// Grouped session - last if next is in different group or is a favorite shown separately
	return nextItem.instance.GroupID != groupID
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
	m.showAgentIcons = settings.ShowAgentIcons
	m.splitView = settings.SplitView
	m.markedSessionID = settings.MarkedSessionID
	m.splitFocus = settings.SplitFocus
	m.markedVisibleIndex = -1 // Will be found after buildVisibleItems

	// Reset maps
	m.lastLines = make(map[string]string)
	m.prevContent = make(map[string]string)
	m.isActive = make(map[string]bool)
	m.activityState = make(map[string]session.SessionActivity)
	m.windowActivityState = make(map[string]map[int]session.SessionActivity)

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
