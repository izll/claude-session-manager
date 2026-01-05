package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/izll/agent-session-manager/session/filters"
	"github.com/mattn/go-runewidth"
)

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

type Status string

const (
	StatusRunning Status = "running"
	StatusPaused  Status = "paused"
	StatusStopped Status = "stopped"
)

// AgentType represents the type of AI agent
type AgentType string

const (
	AgentClaude   AgentType = "claude"
	AgentGemini   AgentType = "gemini"
	AgentAider    AgentType = "aider"
	AgentCodex    AgentType = "codex"
	AgentAmazonQ  AgentType = "amazonq"
	AgentOpenCode AgentType = "opencode"
	AgentCustom   AgentType = "custom"
)

// AgentConfig contains configuration for each agent type
type AgentConfig struct {
	Command            string // Base command to run
	SupportsResume     bool   // Whether agent supports session resume
	SupportsAutoYes    bool   // Whether agent has auto-approve flag
	AutoYesFlag        string // The flag for auto-approve (e.g., "--dangerously-skip-permissions")
	ResumeFlag         string // The flag for resume (e.g., "--resume")
	ResumeIsSubcommand bool   // If true, resume is a subcommand (e.g., "codex resume") not a flag
}

// AgentConfigs maps agent types to their configurations
var AgentConfigs = map[AgentType]AgentConfig{
	AgentClaude: {
		Command:         "claude",
		SupportsResume:  true,
		SupportsAutoYes: true,
		AutoYesFlag:     "--dangerously-skip-permissions",
		ResumeFlag:      "--resume",
	},
	AgentGemini: {
		Command:         "gemini",
		SupportsResume:  true,
		SupportsAutoYes: false,
		ResumeFlag:      "--resume",
	},
	AgentAider: {
		Command:         "aider",
		SupportsResume:  false,
		SupportsAutoYes: true,
		AutoYesFlag:     "--yes",
	},
	AgentCodex: {
		Command:            "codex",
		SupportsResume:     true,
		SupportsAutoYes:    true,
		AutoYesFlag:        "--full-auto",
		ResumeFlag:         "resume",
		ResumeIsSubcommand: true,
	},
	AgentAmazonQ: {
		Command:            "q",
		SupportsResume:     true,
		SupportsAutoYes:    true,
		AutoYesFlag:        "--trust-all-tools",
		ResumeFlag:         "chat --resume",
		ResumeIsSubcommand: true,
	},
	AgentOpenCode: {
		Command:         "opencode",
		SupportsResume:  true,
		SupportsAutoYes: false,
		ResumeFlag:      "--session",
	},
	AgentCustom: {
		Command:         "",
		SupportsResume:  false,
		SupportsAutoYes: false,
	},
}

type Instance struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Path            string    `json:"path"`
	Status          Status    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	AutoYes         bool      `json:"auto_yes"`
	ResumeSessionID string    `json:"resume_session_id,omitempty"` // Claude session ID to resume
	Color           string    `json:"color,omitempty"`             // Foreground color
	BgColor         string    `json:"bg_color,omitempty"`          // Background color
	FullRowColor    bool      `json:"full_row_color,omitempty"`    // Extend background to full row
	GroupID         string    `json:"group_id,omitempty"`          // Session group ID
	Agent           AgentType `json:"agent,omitempty"`             // Agent type (claude, gemini, aider, custom)
	CustomCommand   string    `json:"custom_command,omitempty"`    // Custom command for AgentCustom

}

// GetAgentConfig returns the agent configuration for this instance
func (i *Instance) GetAgentConfig() AgentConfig {
	agent := i.Agent
	if agent == "" {
		agent = AgentClaude // Default to Claude for backward compatibility
	}
	if config, ok := AgentConfigs[agent]; ok {
		return config
	}
	return AgentConfigs[AgentClaude]
}

// expandTilde expands ~ to user's home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[2:])
		}
	} else if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return homeDir
		}
	}
	return path
}

func NewInstance(name, path string, autoYes bool, agent AgentType) (*Instance, error) {
	// Expand ~ to home directory
	path = expandTilde(path)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", absPath)
	}

	id := generateID(name, agent)
	now := time.Now()

	return &Instance{
		ID:        id,
		Name:      name,
		Path:      absPath,
		Status:    StatusStopped,
		CreatedAt: now,
		UpdatedAt: now,
		AutoYes:   autoYes,
		Agent:     agent,
	}, nil
}

func generateID(name string, agent AgentType) string {
	sanitized := strings.ToLower(name)
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	timestamp := time.Now().UnixNano()
	agentStr := string(agent)
	if agentStr == "" {
		agentStr = "claude"
	}
	return fmt.Sprintf("asm_%s_%s_%d", agentStr, sanitized, timestamp)
}

func (i *Instance) TmuxSessionName() string {
	return i.ID
}

// CheckAgentCommand verifies that the agent command exists in PATH
func CheckAgentCommand(inst *Instance) error {
	var cmdToCheck string

	if inst.Agent == AgentCustom {
		// Extract the base command (first word) from custom command
		parts := strings.Fields(inst.CustomCommand)
		if len(parts) > 0 {
			cmdToCheck = parts[0]
		}
	} else {
		config := inst.GetAgentConfig()
		cmdToCheck = config.Command
	}

	if cmdToCheck == "" {
		return fmt.Errorf("no command specified")
	}

	if _, err := exec.LookPath(cmdToCheck); err != nil {
		return fmt.Errorf("command '%s' not found - is it installed?", cmdToCheck)
	}

	return nil
}

func (i *Instance) Start() error {
	return i.StartWithResume("")
}

func (i *Instance) StartWithResume(resumeID string) error {
	if i.Status == StatusRunning {
		return fmt.Errorf("instance already running")
	}

	sessionName := i.TmuxSessionName()

	// Check if tmux session already exists
	checkCmd := exec.Command("tmux", "has-session", "-t", sessionName)
	sessionExists := checkCmd.Run() == nil

	if !sessionExists {
		// Build command based on agent type
		config := i.GetAgentConfig()
		var agentCmd string
		var cmdToCheck string

		if i.Agent == AgentCustom {
			// Use custom command directly
			agentCmd = i.CustomCommand
			// Extract the base command (first word) to check
			parts := strings.Fields(i.CustomCommand)
			if len(parts) > 0 {
				cmdToCheck = parts[0]
			}
		} else {
			cmdToCheck = config.Command
			args := []string{}

			// Handle resume subcommands (codex resume, q chat --resume) vs flags (claude --resume)
			if config.SupportsResume && config.ResumeIsSubcommand {
				// Resume is a subcommand - put it first, then flags, then session ID
				if resumeID != "" || i.ResumeSessionID != "" {
					// Add resume subcommand
					args = append(args, config.ResumeFlag)

					// Add auto-yes flag after subcommand if supported
					if i.AutoYes && config.SupportsAutoYes && config.AutoYesFlag != "" {
						args = append(args, config.AutoYesFlag)
					}

					// Add session ID
					if resumeID != "" {
						args = append(args, resumeID)
						i.ResumeSessionID = resumeID
					} else if i.ResumeSessionID != "" {
						args = append(args, i.ResumeSessionID)
					}
				} else {
					// No resume - just add auto-yes flag if needed
					if i.AutoYes && config.SupportsAutoYes && config.AutoYesFlag != "" {
						args = append(args, config.AutoYesFlag)
					}
				}
			} else {
				// Resume is a flag - add auto-yes first, then resume flag
				// Add auto-yes flag if supported and enabled
				if i.AutoYes && config.SupportsAutoYes && config.AutoYesFlag != "" {
					args = append(args, config.AutoYesFlag)
				}

				// Add resume flag if supported and specified
				if config.SupportsResume && config.ResumeFlag != "" {
					if resumeID != "" {
						args = append(args, config.ResumeFlag, resumeID)
						i.ResumeSessionID = resumeID
					} else if i.ResumeSessionID != "" {
						args = append(args, config.ResumeFlag, i.ResumeSessionID)
					}
				}
			}

			agentCmd = config.Command + " " + strings.Join(args, " ")
		}

		// Check if the command exists
		if cmdToCheck != "" {
			if _, err := exec.LookPath(cmdToCheck); err != nil {
				return fmt.Errorf("command '%s' not found - is it installed?", cmdToCheck)
			}
		}

		// Create new tmux session
		cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", i.Path, agentCmd)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create tmux session: %w", err)
		}

		// Wait for session to be ready
		for j := 0; j < 20; j++ {
			checkCmd := exec.Command("tmux", "has-session", "-t", sessionName)
			if checkCmd.Run() == nil {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		// Configure tmux session for better scrolling
		exec.Command("tmux", "set-option", "-t", sessionName, "history-limit", "50000").Run()
		exec.Command("tmux", "set-option", "-t", sessionName, "mouse", "on").Run()

		// Use latest client size and aggressive resize for proper terminal following
		exec.Command("tmux", "set-option", "-t", sessionName, "window-size", "latest").Run()
		exec.Command("tmux", "set-option", "-t", sessionName, "aggressive-resize", "on").Run()

		// Enable xterm keys for Shift+PageUp/Down support
		exec.Command("tmux", "set-option", "-t", sessionName, "-g", "xterm-keys", "on").Run()

		// Set terminal overrides for better key support
		exec.Command("tmux", "set-option", "-t", sessionName, "-ga", "terminal-overrides", ",xterm*:smcup@:rmcup@").Run()

		// Bind Shift+PageUp/Down for scrolling in copy mode
		exec.Command("tmux", "bind-key", "-T", "root", "S-PageUp", "copy-mode", "-eu").Run()
		exec.Command("tmux", "bind-key", "-T", "root", "S-PageDown", "send-keys", "PageDown").Run()
		exec.Command("tmux", "bind-key", "-T", "copy-mode-vi", "S-PageUp", "send-keys", "-X", "page-up").Run()
		exec.Command("tmux", "bind-key", "-T", "copy-mode-vi", "S-PageDown", "send-keys", "-X", "page-down").Run()

		// Ctrl+q will be set up with resize in UpdateDetachBinding

		// Check if session is still alive after a short delay (detect immediate exit)
		time.Sleep(300 * time.Millisecond)
		if !i.IsAlive() {
			// Session died immediately - try to get output for error message
			return fmt.Errorf("session exited immediately - check if login or API key is required")
		}
	}

	i.Status = StatusRunning
	i.UpdatedAt = time.Now()

	return nil
}

func (i *Instance) Stop() error {
	if i.Status != StatusRunning {
		return nil
	}

	sessionName := i.TmuxSessionName()
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kill tmux session: %w", err)
	}

	i.Status = StatusStopped
	i.UpdatedAt = time.Now()

	return nil
}

func (i *Instance) Attach() error {
	if i.Status != StatusRunning {
		return fmt.Errorf("instance not running")
	}

	sessionName := i.TmuxSessionName()
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (i *Instance) IsAlive() bool {
	sessionName := i.TmuxSessionName()
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	return cmd.Run() == nil
}

// ResizePane resizes the tmux pane to the specified dimensions
func (i *Instance) ResizePane(width, height int) error {
	if !i.IsAlive() {
		return nil
	}
	sessionName := i.TmuxSessionName()
	return exec.Command("tmux", "resize-window", "-t", sessionName, "-x", fmt.Sprintf("%d", width), "-y", fmt.Sprintf("%d", height)).Run()
}

// UpdateDetachBinding updates Ctrl+Q to resize to preview size before detaching
func (i *Instance) UpdateDetachBinding(previewWidth, previewHeight int) {
	if !i.IsAlive() {
		return
	}
	sessionName := i.TmuxSessionName()
	// Bind Ctrl+Q: ASM sessions get resize+detach, all others get plain detach
	shellScript := fmt.Sprintf(`
SESSION=$(tmux display-message -p '#{session_name}')
if echo "$SESSION" | grep -q '^asm_'; then
  tmux resize-window -t %s -x %d -y %d
fi
tmux detach-client
`, sessionName, previewWidth, previewHeight)
	exec.Command("tmux", "bind-key", "-n", "C-q", "run-shell", shellScript).Run()
}


func (i *Instance) GetPreview(lines int) (string, error) {
	if !i.IsAlive() {
		return "(session not running)", nil
	}

	sessionName := i.TmuxSessionName()
	// Capture pane with scrollback history (-S for start line, -E for end)
	// -S -lines means start from 'lines' back in history
	// -e preserves colors, -J joins wrapped lines
	startLine := fmt.Sprintf("-%d", lines)
	cmd := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p", "-e", "-J", "-S", startLine)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane: %w", err)
	}

	// Post-process to remove extra spaces after wide characters (emojis)
	// This is needed because tmux -J flag adds padding after wide chars
	result := removeWideCharPadding(string(output))
	return strings.TrimRight(result, "\n"), nil
}

// removeWideCharPadding removes extra spaces after wide characters (emojis)
// that tmux -J flag adds when capturing panes
func removeWideCharPadding(s string) string {
	runes := []rune(s)
	var result []rune
	i := 0

	for i < len(runes) {
		// Check for ANSI escape sequence - preserve them
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			start := i
			i += 2
			// Find end of ANSI sequence
			for i < len(runes) && !((runes[i] >= 'A' && runes[i] <= 'Z') || (runes[i] >= 'a' && runes[i] <= 'z')) {
				i++
			}
			if i < len(runes) {
				i++ // include final letter
			}
			// Copy ANSI sequence
			result = append(result, runes[start:i]...)
			continue
		}

		// Normal character
		currentRune := runes[i]
		result = append(result, currentRune)
		i++

		// If this is a wide character (width 2) and next char is space, skip the space
		if i < len(runes) && runes[i] == ' ' {
			// Check if previous character was wide using runewidth
			if runewidth.RuneWidth(currentRune) == 2 {
				i++ // Skip the space after wide character
			}
		}
	}

	return string(result)
}

// GetLastLine returns the last non-empty line of output (for status display)
func (i *Instance) GetLastLine() string {
	if !i.IsAlive() {
		return "stopped"
	}

	sessionName := i.TmuxSessionName()
	// Capture last 50 lines with colors (-e flag preserves ANSI escape sequences)
	// -J flag joins wrapped lines (prevents terminal width wrapping issues)
	cmd := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p", "-e", "-J", "-S", "-50")
	output, err := cmd.Output()
	if err != nil {
		return "..."
	}

	lines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")

	// Find last meaningful line
	for j := len(lines) - 1; j >= 0; j-- {
		line := lines[j]
		// Strip ANSI codes for checking
		cleanLine := strings.TrimSpace(stripANSI(line))
		// Skip empty lines
		if cleanLine == "" {
			continue
		}

		// Agent-specific filtering using configurable filters
		agentFilters := filters.LoadFilters()
		agentName := string(i.Agent)
		if agentName == "" {
			agentName = "claude"
		}

		if config, ok := agentFilters[agentName]; ok {
			skip, content := filters.ApplyFilter(config, cleanLine)
			if skip {
				continue
			}
			if content != "" {
				return content
			}
		}

		// Found actual content - return with colors
		return line
	}

	return "..."
}

func (i *Instance) SendKeys(keys string) error {
	if !i.IsAlive() {
		return fmt.Errorf("session not running")
	}

	sessionName := i.TmuxSessionName()
	cmd := exec.Command("tmux", "send-keys", "-t", sessionName, keys)
	return cmd.Run()
}

// SendText sends text literally (not interpreted as key names)
func (i *Instance) SendText(text string) error {
	if !i.IsAlive() {
		return fmt.Errorf("session not running")
	}

	sessionName := i.TmuxSessionName()
	// Use -l flag to send text literally without interpreting key names
	cmd := exec.Command("tmux", "send-keys", "-l", "-t", sessionName, text)
	return cmd.Run()
}

// SendPrompt sends a prompt text followed by Enter key
func (i *Instance) SendPrompt(text string) error {
	if !i.IsAlive() {
		return fmt.Errorf("session not running")
	}

	sessionName := i.TmuxSessionName()

	// First send text literally with -l flag to avoid key interpretation
	cmd := exec.Command("tmux", "send-keys", "-l", "-t", sessionName, text)
	if err := cmd.Run(); err != nil {
		return err
	}

	// Small delay to ensure text is processed before Enter
	time.Sleep(50 * time.Millisecond)

	// Then send Enter separately
	cmd = exec.Command("tmux", "send-keys", "-t", sessionName, "Enter")
	return cmd.Run()
}

func (i *Instance) UpdateStatus() {
	if i.IsAlive() {
		i.Status = StatusRunning
	} else {
		i.Status = StatusStopped
	}
}
