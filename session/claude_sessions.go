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

type ClaudeSession struct {
	SessionID   string    `json:"session_id"`
	FirstPrompt string    `json:"first_prompt"`
	LastPrompt  string    `json:"last_prompt"`
	MessageCount int      `json:"message_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

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

func ListClaudeSessions(projectPath string) ([]ClaudeSession, error) {
	claudeDir := GetClaudeProjectDir(projectPath)

	entries, err := os.ReadDir(claudeDir)
	if os.IsNotExist(err) {
		return []ClaudeSession{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read claude directory: %w", err)
	}

	var sessions []ClaudeSession

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

func parseSessionFile(path string, sessionID string) (*ClaudeSession, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	session := &ClaudeSession{
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
