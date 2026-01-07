package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/izll/agent-session-manager/session"
)

// Diff color styles
var (
	diffAdditionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")) // Green
	diffDeletionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")) // Red
	diffHunkStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#0ea5e9")) // Cyan
	diffMetaStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")) // Gray
	diffFileStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#f59e0b")).Bold(true) // Orange bold
)

// DiffMode represents the type of diff to display
type DiffMode int

const (
	DiffModeSession DiffMode = iota // Changes since session start
	DiffModeFull                    // All uncommitted changes
)

// DiffPane manages the diff display with scrolling support
type DiffPane struct {
	viewport viewport.Model
	stats    *session.DiffStats
	mode     DiffMode
	width    int
	height   int
}

// NewDiffPane creates a new diff pane
func NewDiffPane() *DiffPane {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()
	return &DiffPane{
		viewport: vp,
		mode:     DiffModeFull,
	}
}

// SetSize updates the diff pane dimensions
func (d *DiffPane) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.viewport.Width = width
	d.viewport.Height = height
	d.updateContent()
}

// SetDiff updates the diff content from an instance
func (d *DiffPane) SetDiff(inst *session.Instance) {
	if inst == nil {
		d.stats = nil
		d.updateContent()
		return
	}

	switch d.mode {
	case DiffModeSession:
		d.stats = inst.GetSessionDiff()
	case DiffModeFull:
		d.stats = inst.GetFullDiff()
	}
	d.updateContent()
}

// ToggleMode switches between session diff and full diff
func (d *DiffPane) ToggleMode() {
	if d.mode == DiffModeSession {
		d.mode = DiffModeFull
	} else {
		d.mode = DiffModeSession
	}
}

// GetMode returns the current diff mode
func (d *DiffPane) GetMode() DiffMode {
	return d.mode
}

// GetModeLabel returns a human-readable label for current mode
func (d *DiffPane) GetModeLabel() string {
	if d.mode == DiffModeSession {
		return "Session"
	}
	return "Full"
}

// ScrollUp scrolls the viewport up
func (d *DiffPane) ScrollUp() {
	d.viewport.LineUp(1)
}

// ScrollDown scrolls the viewport down
func (d *DiffPane) ScrollDown() {
	d.viewport.LineDown(1)
}

// PageUp scrolls the viewport up by a page
func (d *DiffPane) PageUp() {
	d.viewport.ViewUp()
}

// PageDown scrolls the viewport down by a page
func (d *DiffPane) PageDown() {
	d.viewport.ViewDown()
}

// GotoTop scrolls to the beginning of the content
func (d *DiffPane) GotoTop() {
	d.viewport.GotoTop()
}

// GotoBottom scrolls to the end of the content
func (d *DiffPane) GotoBottom() {
	d.viewport.GotoBottom()
}

// View renders the diff pane
func (d *DiffPane) View() string {
	return d.viewport.View()
}

// updateContent refreshes the viewport content
func (d *DiffPane) updateContent() {
	if d.stats == nil {
		d.viewport.SetContent(dimStyle.Render("No diff available"))
		return
	}

	if d.stats.Error != nil {
		d.viewport.SetContent(errorStyle.Render(fmt.Sprintf("Error: %v", d.stats.Error)))
		return
	}

	if d.stats.IsEmpty() {
		d.viewport.SetContent(dimStyle.Render("No changes"))
		return
	}

	// Stats header - horizontal join like claude-squad
	additions := diffAdditionStyle.Render(fmt.Sprintf("+%d", d.stats.Added))
	deletions := diffDeletionStyle.Render(fmt.Sprintf("-%d", d.stats.Removed))
	statsLine := " " + lipgloss.JoinHorizontal(lipgloss.Center, additions, "  ", deletions)

	// Colorized diff content
	diffContent := colorizeDiff(d.stats.Content)

	// Join stats and diff vertically
	d.viewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, statsLine, "", diffContent))
}

// colorizeDiff applies syntax highlighting to diff content
func colorizeDiff(diff string) string {
	if diff == "" {
		return ""
	}

	var result strings.Builder
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		if len(line) == 0 {
			result.WriteString("\n")
			continue
		}

		// Add space padding for a more spacious look
		coloredLine := " " + colorDiffLine(line) + " "
		result.WriteString(coloredLine)
		result.WriteString("\n")
	}

	return result.String()
}

// colorDiffLine applies color to a single diff line
func colorDiffLine(line string) string {
	if len(line) == 0 {
		return ""
	}

	switch {
	case strings.HasPrefix(line, "@@"):
		// Hunk header
		return diffHunkStyle.Render(line)

	case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
		// File markers
		return diffFileStyle.Render(line)

	case strings.HasPrefix(line, "diff --git"):
		// Diff header
		return diffFileStyle.Render(line)

	case strings.HasPrefix(line, "index ") ||
		strings.HasPrefix(line, "new file") ||
		strings.HasPrefix(line, "deleted file"):
		// Meta info
		return diffMetaStyle.Render(line)

	case line[0] == '+':
		// Addition
		return diffAdditionStyle.Render(line)

	case line[0] == '-':
		// Deletion
		return diffDeletionStyle.Render(line)

	default:
		// Context or other lines
		return line
	}
}
