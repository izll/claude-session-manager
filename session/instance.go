package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/creack/pty"
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

	// ptmx is a PTY running tmux attach - used to control pane size while detached
	ptmx *os.File `json:"-"`
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

func NewInstance(name, path string, autoYes bool) (*Instance, error) {
	// Expand ~ to home directory
	path = expandTilde(path)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", absPath)
	}

	id := generateID(name)
	now := time.Now()

	return &Instance{
		ID:        id,
		Name:      name,
		Path:      absPath,
		Status:    StatusStopped,
		CreatedAt: now,
		UpdatedAt: now,
		AutoYes:   autoYes,
	}, nil
}

func generateID(name string) string {
	sanitized := strings.ToLower(name)
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("csm_%s_%d", sanitized, timestamp)
}

func (i *Instance) TmuxSessionName() string {
	return fmt.Sprintf("csm_%s", i.ID)
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
		// Build claude command
		claudeArgs := []string{}
		if i.AutoYes {
			claudeArgs = append(claudeArgs, "--dangerously-skip-permissions")
		}

		// Add resume flag if specified
		if resumeID != "" {
			claudeArgs = append(claudeArgs, "--resume", resumeID)
			i.ResumeSessionID = resumeID
		} else if i.ResumeSessionID != "" {
			claudeArgs = append(claudeArgs, "--resume", i.ResumeSessionID)
		}

		claudeCmd := "claude " + strings.Join(claudeArgs, " ")

		// Create new tmux session
		cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", i.Path, claudeCmd)
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
	}

	// Attach through PTY and keep the handle for size control (like Claude Squad)
	if err := i.attachPty(); err != nil {
		return fmt.Errorf("failed to attach PTY: %w", err)
	}

	i.Status = StatusRunning
	i.UpdatedAt = time.Now()

	return nil
}

// attachPty attaches to the tmux session through a PTY and keeps the handle
func (i *Instance) attachPty() error {
	sessionName := i.TmuxSessionName()
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	i.ptmx = ptmx
	return nil
}

func (i *Instance) Stop() error {
	if i.Status != StatusRunning {
		return nil
	}

	// Close PTY handle
	if i.ptmx != nil {
		i.ptmx.Close()
		i.ptmx = nil
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
	// Bind Ctrl+Q to: resize-window, then detach - all in one shell command
	shellCmd := fmt.Sprintf("tmux resize-window -t %s -x %d -y %d; tmux detach-client", sessionName, previewWidth, previewHeight)
	exec.Command("tmux", "bind-key", "-n", "C-q", "run-shell", shellCmd).Run()
}

// EnsurePty ensures we have a PTY connection (for restored instances)
func (i *Instance) EnsurePty() error {
	if i.ptmx != nil {
		return nil
	}
	if !i.IsAlive() {
		return nil
	}
	return i.attachPty()
}

// ClosePty closes the PTY connection
func (i *Instance) ClosePty() {
	if i.ptmx != nil {
		i.ptmx.Close()
		i.ptmx = nil
	}
}

func (i *Instance) GetPreview(lines int) (string, error) {
	if !i.IsAlive() {
		return "(session not running)", nil
	}

	sessionName := i.TmuxSessionName()
	// Capture visible pane with colors (-e flag)
	cmd := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p", "-e")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane: %w", err)
	}

	// Get all lines
	allLines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")

	// Take last N lines
	startIdx := len(allLines) - lines
	if startIdx < 0 {
		startIdx = 0
	}

	return strings.Join(allLines[startIdx:], "\n"), nil
}

// GetLastLine returns the last non-empty line of output (for status display)
func (i *Instance) GetLastLine() string {
	if !i.IsAlive() {
		return "stopped"
	}

	sessionName := i.TmuxSessionName()
	// Just capture the visible pane with colors (-e flag preserves ANSI escape sequences)
	cmd := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p", "-e")
	output, err := cmd.Output()
	if err != nil {
		return "..."
	}

	lines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")

	// Find last meaningful line (skip Claude UI elements)
	for j := len(lines) - 1; j >= 0; j-- {
		line := lines[j]
		// Strip ANSI codes for checking
		cleanLine := strings.TrimSpace(stripANSI(line))
		// Skip empty lines
		if cleanLine == "" {
			continue
		}
		// Skip status bar
		if strings.Contains(cleanLine, "? for") || strings.Contains(cleanLine, "Context left") || strings.Contains(cleanLine, "accept edits") {
			continue
		}
		// Skip separator lines (more than 20 dash chars)
		if strings.Count(cleanLine, "─") > 20 {
			continue
		}
		// Skip empty prompt
		if cleanLine == ">" || strings.HasPrefix(cleanLine, "╭") || strings.HasPrefix(cleanLine, "╰") {
			continue
		}
		// Found actual content - return with colors but truncate by visible length
		if len(cleanLine) > 50 {
			// Truncate based on clean length, but we need to be careful with ANSI codes
			// For simplicity, just return the line with colors
			return line
		}
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

func (i *Instance) UpdateStatus() {
	if i.IsAlive() {
		i.Status = StatusRunning
	} else {
		i.Status = StatusStopped
	}
}
