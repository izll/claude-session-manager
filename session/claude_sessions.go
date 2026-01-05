package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type sessionLine struct {
	Type        string `json:"type"`
	SessionID   string `json:"sessionId"`
	Timestamp   string `json:"timestamp"`
	IsSidechain bool   `json:"isSidechain"`
	AgentID     string `json:"agentId"`
	Message     *struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"`
	} `json:"message"`
}

// isValidUUID checks if a string looks like a UUID (basic check)
func isValidUUID(s string) bool {
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

func GetClaudeProjectDir(projectPath string) string {
	homeDir, _ := os.UserHomeDir()

	// Resolve symlinks to get the real path (Claude does this)
	realPath, err := filepath.EvalSymlinks(projectPath)
	if err == nil {
		projectPath = realPath
	}

	// Convert path to Claude's format: /home/user/project_name -> -home-user-project-name
	// Claude replaces: / -> -, _ -> -, space -> -, and accented chars -> -
	var result strings.Builder
	for _, r := range projectPath {
		if r == '/' || r == '_' || r == ' ' {
			result.WriteRune('-')
		} else if r > 127 {
			// Non-ASCII characters (accented letters, etc.) -> -
			result.WriteRune('-')
		} else {
			result.WriteRune(r)
		}
	}
	sanitized := result.String()
	if strings.HasPrefix(sanitized, "-") {
		sanitized = sanitized[1:] // Remove leading dash
	}
	sanitized = "-" + sanitized // Add back the leading dash Claude uses
	return filepath.Join(homeDir, ".claude", "projects", sanitized)
}

func ListAgentSessions(projectPath string) ([]AgentSession, error) {
	claudeDir := GetClaudeProjectDir(projectPath)

	entries, err := os.ReadDir(claudeDir)
	if os.IsNotExist(err) {
		return []AgentSession{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read claude directory: %w", err)
	}

	var sessions []AgentSession

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), ".jsonl")

		// Skip non-UUID files (like agent-* files which are subagent sessions)
		if !isValidUUID(sessionID) {
			continue
		}

		sessionPath := filepath.Join(claudeDir, entry.Name())

		session, err := parseSessionFile(sessionPath, sessionID)
		if err != nil {
			continue // Skip invalid files
		}
		// Only include sessions with at least one real user message
		if session.MessageCount > 0 && session.FirstPrompt != "" {
			sessions = append(sessions, *session)
		}
	}

	// Sort by UpdatedAt descending (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

func parseSessionFile(path string, sessionID string) (*AgentSession, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	session := &AgentSession{
		SessionID: sessionID,
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var firstUserMessage string
	var lastUserMessage string
	var firstTimestamp time.Time
	var lastTimestamp time.Time
	messageCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var sl sessionLine
		if err := json.Unmarshal([]byte(line), &sl); err != nil {
			continue
		}

		// Only count main conversation messages (not sidechain/agent messages)
		if sl.Type == "user" && sl.Message != nil && sl.Message.Role == "user" && !sl.IsSidechain && sl.AgentID == "" {
			// Extract content as string
			content := extractContent(sl.Message.Content)

			// Skip empty content (tool results, notifications, etc.)
			if content == "" {
				continue
			}

			messageCount++
			ts, _ := time.Parse(time.RFC3339, sl.Timestamp)

			if firstUserMessage == "" {
				firstUserMessage = content
				firstTimestamp = ts
			}
			lastUserMessage = content
			lastTimestamp = ts
		}
	}

	session.FirstPrompt = truncateString(firstUserMessage, 80)
	session.LastPrompt = truncateString(lastUserMessage, 80)
	session.MessageCount = messageCount
	session.CreatedAt = firstTimestamp
	session.UpdatedAt = lastTimestamp

	return session, nil
}

func extractContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		// Skip tool results and notifications
		if strings.HasPrefix(v, "<bash-notification>") ||
			strings.HasPrefix(v, "<tool_result>") ||
			strings.HasPrefix(v, "{\"tool_use_id\":") {
			return ""
		}
		return v
	case []interface{}:
		// Content can be array of content blocks
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				// Skip tool_result type blocks
				if t, ok := m["type"].(string); ok && t == "tool_result" {
					continue
				}
				if text, ok := m["text"].(string); ok {
					return text
				}
			}
		}
	}
	return ""
}

func truncateString(s string, maxLen int) string {
	// Remove newlines
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// GetClaudeStatusLine handles Claude Code's special UI with horizontal separator lines.
// If the input area (between two horizontal lines) has only 1 line (the prompt),
// it returns the content above the top separator instead of the prompt line.
func GetClaudeStatusLine(lines []string, stripANSIFunc func(string) string) string {
	// Find horizontal line positions (lines with many ─ or ━ characters)
	var separatorIndices []int
	for idx, line := range lines {
		cleanLine := strings.TrimSpace(stripANSIFunc(line))
		sepCount := strings.Count(cleanLine, "─") + strings.Count(cleanLine, "━")
		if sepCount > 20 {
			separatorIndices = append(separatorIndices, idx)
		}
	}

	// Need at least 2 separators to detect input area
	if len(separatorIndices) < 2 {
		return ""
	}

	// Get the last two separators (they form the input area boundary)
	topSepIdx := separatorIndices[len(separatorIndices)-2]
	bottomSepIdx := separatorIndices[len(separatorIndices)-1]

	// Count non-empty lines between separators
	contentLinesBetween := 0
	for idx := topSepIdx + 1; idx < bottomSepIdx; idx++ {
		cleanLine := strings.TrimSpace(stripANSIFunc(lines[idx]))
		if cleanLine != "" {
			contentLinesBetween++
		}
	}

	// If only 1 line between separators (the prompt line), get content above top separator
	if contentLinesBetween <= 1 {
		// Search upward from top separator for meaningful content
		for j := topSepIdx - 1; j >= 0; j-- {
			line := lines[j]
			cleanLine := strings.TrimSpace(stripANSIFunc(line))
			if cleanLine == "" {
				continue
			}

			// Skip separator lines
			sepCount := strings.Count(cleanLine, "─") + strings.Count(cleanLine, "━")
			if sepCount > 20 {
				continue
			}

			// Skip UI elements
			if strings.HasPrefix(cleanLine, "╭") || strings.HasPrefix(cleanLine, "╰") {
				continue
			}

			// Skip tip lines
			if strings.HasPrefix(cleanLine, "└") || strings.HasPrefix(cleanLine, "Tip:") {
				continue
			}

			// Found actual content above input area
			return line
		}
	} else {
		// Multiple lines between separators - get content from input area (bottom to top)
		for j := bottomSepIdx - 1; j > topSepIdx; j-- {
			line := lines[j]
			cleanLine := strings.TrimSpace(stripANSIFunc(line))
			if cleanLine == "" {
				continue
			}

			// Skip prompt-only lines (just ">")
			if cleanLine == ">" || strings.HasPrefix(cleanLine, "> ") {
				continue
			}

			// Found actual content in input area
			return line
		}
	}

	// No content found - return empty for fallback processing
	return ""
}
