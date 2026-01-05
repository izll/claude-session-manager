package session

import (
	"os/exec"
	"strings"
)

// SessionActivity represents the activity state of a session
type SessionActivity int

const (
	ActivityIdle    SessionActivity = iota // No activity, no prompt
	ActivityBusy                           // Agent is working
	ActivityWaiting                        // Agent needs user input/permission
)

// Busy patterns (case sensitive)
var busyPatterns = []string{
	"esc to interrupt",
	"tokens",
	"Generating",
}

// Waiting patterns (case insensitive) - common for all agents
var waitingPatterns = []string{
	"allow once",
	"allow always",
	"yes, allow",
	"no, and tell",
	"esc to cancel",
	"do you want to proceed",
	"waiting for user",
	"waiting for tool",
	"apply this change",
}

// Claude-specific waiting patterns
var claudeWaitingPatterns = []string{
	"? for shortcuts",
}

// Spinner characters (braille dots)
var spinners = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}


// DetectActivity analyzes tmux pane content to determine session activity
func (i *Instance) DetectActivity() SessionActivity {
	if !i.IsAlive() {
		return ActivityIdle
	}

	sessionName := i.TmuxSessionName()
	cmd := exec.Command("tmux", "capture-pane", "-t", sessionName, "-p", "-S", "-50")
	output, err := cmd.Output()
	if err != nil {
		return ActivityIdle
	}

	lines := strings.Split(string(output), "\n")

	// For Claude: use the area between horizontal separator lines
	if i.Agent == AgentClaude || i.Agent == "" {
		return detectClaudeActivity(lines)
	}

	// For other agents: simple pattern check on last lines
	return detectGenericActivity(lines)
}

// detectClaudeActivity uses Claude Code's UI structure (horizontal separators)
func detectClaudeActivity(lines []string) SessionActivity {
	// Find separator line positions
	var separatorIndices []int
	for idx, line := range lines {
		cleanLine := strings.TrimSpace(stripANSIForDetect(line))
		sepCount := strings.Count(cleanLine, "─") + strings.Count(cleanLine, "━")
		if sepCount > 20 {
			separatorIndices = append(separatorIndices, idx)
		}
	}

	var inputAreaLines []string
	var aboveSeparatorLines []string // Lines above top separator (for thinking state)

	if len(separatorIndices) >= 2 {
		// Normal mode: 2 separators, check between them
		topSepIdx := separatorIndices[len(separatorIndices)-2]
		bottomSepIdx := separatorIndices[len(separatorIndices)-1]

		// Count non-empty lines between separators
		contentCount := 0
		for idx := topSepIdx + 1; idx < bottomSepIdx; idx++ {
			cleanLine := strings.TrimSpace(stripANSIForDetect(lines[idx]))
			if cleanLine != "" {
				inputAreaLines = append(inputAreaLines, cleanLine)
				contentCount++
			}
		}

		// If only prompt line (or empty), check content ABOVE top separator
		// This is where Claude shows spinner and "esc to interrupt" during thinking
		if contentCount <= 1 {
			for j := topSepIdx - 1; j >= 0 && j >= topSepIdx-15; j-- {
				cleanLine := strings.TrimSpace(stripANSIForDetect(lines[j]))
				if cleanLine != "" {
					// Skip UI elements and tips
					if strings.HasPrefix(cleanLine, "╭") || strings.HasPrefix(cleanLine, "╰") ||
						strings.HasPrefix(cleanLine, "└") || strings.HasPrefix(cleanLine, "Tip:") {
						continue
					}
					aboveSeparatorLines = append(aboveSeparatorLines, cleanLine)
				}
			}
		}
	} else if len(separatorIndices) == 1 {
		// Permission dialog: only 1 separator, check lines below it
		sepIdx := separatorIndices[0]
		for idx := sepIdx + 1; idx < len(lines); idx++ {
			cleanLine := strings.TrimSpace(stripANSIForDetect(lines[idx]))
			if cleanLine != "" {
				inputAreaLines = append(inputAreaLines, cleanLine)
			}
		}
	} else {
		// No separators - check last lines
		for j := len(lines) - 1; j >= 0 && j >= len(lines)-10; j-- {
			cleanLine := strings.TrimSpace(stripANSIForDetect(lines[j]))
			if cleanLine != "" {
				inputAreaLines = append(inputAreaLines, cleanLine)
			}
		}
	}

	// Combine lines to check - input area has priority, then above separator
	allLinesToCheck := append(inputAreaLines, aboveSeparatorLines...)

	// Check for patterns
	// First pass: check for waiting patterns (higher priority)
	for _, line := range allLinesToCheck {
		lineLower := strings.ToLower(line)
		// Common waiting patterns
		for _, pattern := range waitingPatterns {
			if strings.Contains(lineLower, pattern) {
				return ActivityWaiting
			}
		}
		// Claude-specific waiting patterns
		for _, pattern := range claudeWaitingPatterns {
			if strings.Contains(lineLower, pattern) {
				return ActivityWaiting
			}
		}
	}

	// Second pass: check for busy patterns
	for _, line := range allLinesToCheck {
		for _, pattern := range busyPatterns {
			if strings.Contains(line, pattern) {
				return ActivityBusy
			}
		}
		for _, s := range spinners {
			if strings.Contains(line, s) {
				return ActivityBusy
			}
		}
	}

	return ActivityIdle
}

// detectGenericActivity checks last lines for other agents
func detectGenericActivity(lines []string) SessionActivity {
	// First pass: check for waiting patterns (higher priority)
	for j := len(lines) - 1; j >= 0 && j >= len(lines)-15; j-- {
		line := strings.TrimSpace(stripANSIForDetect(lines[j]))
		if line == "" {
			continue
		}
		lineLower := strings.ToLower(line)
		for _, pattern := range waitingPatterns {
			if strings.Contains(lineLower, pattern) {
				return ActivityWaiting
			}
		}
	}

	// Second pass: check for busy patterns
	for j := len(lines) - 1; j >= 0 && j >= len(lines)-15; j-- {
		line := strings.TrimSpace(stripANSIForDetect(lines[j]))
		if line == "" {
			continue
		}
		for _, pattern := range busyPatterns {
			if strings.Contains(line, pattern) {
				return ActivityBusy
			}
		}
		for _, s := range spinners {
			if strings.Contains(line, s) {
				return ActivityBusy
			}
		}
	}

	return ActivityIdle
}

// stripANSIForDetect removes ANSI escape sequences (uses stripANSI from instance.go)
func stripANSIForDetect(s string) string {
	return stripANSI(s)
}
