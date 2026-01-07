package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/izll/agent-session-manager/session"
)

// projectSelectView renders the project selection screen
func (m Model) projectSelectView() string {
	// Calculate box width first (needed for centering)
	boxWidth := 50
	if m.width > 60 {
		boxWidth = m.width / 2
	}
	if boxWidth > 80 {
		boxWidth = 80
	}

	var content strings.Builder

	// Title - with ice gradient
	title := " " + applyLipglossGradient("Agent Session Manager", gradients["gradient-ice"]) + " "

	content.WriteString("\n")
	content.WriteString(lipgloss.PlaceHorizontal(boxWidth, lipgloss.Center, title))
	content.WriteString("\n\n")

	// Build the project list
	var listContent strings.Builder
	listContent.WriteString("\n") // Extra empty line after top border

	// Projects first
	projectNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple)).Bold(true)
	for i, project := range m.projects {
		sessionCount := m.storage.GetProjectSessionCount(project.ID)
		countStr := fmt.Sprintf("[%d]", sessionCount)

		// Pad to align counts
		padding := boxWidth - len(project.Name) - len(countStr) - 6
		if padding < 1 {
			padding = 1
		}

		if i == m.projectCursor {
			listContent.WriteString(listSelectedStyle.Render(fmt.Sprintf("> %s%s%s", project.Name, strings.Repeat(" ", padding), countStr)))
		} else {
			listContent.WriteString(fmt.Sprintf("  %s%s%s", projectNameStyle.Render(project.Name), strings.Repeat(" ", padding), dimStyle.Render(countStr)))
		}
		listContent.WriteString("\n")
	}

	// Separator after projects
	if len(m.projects) > 0 {
		listContent.WriteString(dimStyle.Render("  " + strings.Repeat("─", boxWidth-4)))
		listContent.WriteString("\n")
	}

	// Continue without project option (after projects)
	continueIdx := len(m.projects)
	defaultCount := m.storage.GetProjectSessionCount("")
	defaultCountStr := fmt.Sprintf("[%d]", defaultCount)
	defaultText := "No project"
	defaultPadding := boxWidth - len(defaultText) - len(defaultCountStr) - 10
	if defaultPadding < 1 {
		defaultPadding = 1
	}
	if m.projectCursor == continueIdx {
		listContent.WriteString(listSelectedStyle.Render(fmt.Sprintf("> [ ] %s%s%s", defaultText, strings.Repeat(" ", defaultPadding), defaultCountStr)))
	} else {
		listContent.WriteString(fmt.Sprintf("  [ ] %s%s%s", defaultText, strings.Repeat(" ", defaultPadding), dimStyle.Render(defaultCountStr)))
	}
	listContent.WriteString("\n")

	// New Project option (always last)
	newProjectIdx := len(m.projects) + 1
	if m.projectCursor == newProjectIdx {
		listContent.WriteString(listSelectedStyle.Render("> [+] New Project"))
	} else {
		listContent.WriteString("  [+] New Project")
	}
	listContent.WriteString("\n")

	// Wrap in a box (without bottom border - we'll add it manually with version)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorPurple)).
		Padding(1, 2).
		Width(boxWidth)

	boxRendered := boxStyle.Render(listContent.String())

	// Replace the last line (bottom border) with version embedded
	lines := strings.Split(boxRendered, "\n")
	if len(lines) > 1 {
		// Get actual box width from the first line (top border)
		actualWidth := lipgloss.Width(lines[0])

		// Build custom bottom border with version
		version := fmt.Sprintf(" v%s ", AppVersion)
		borderColor := lipgloss.Color(ColorPurple)
		borderStyle := lipgloss.NewStyle().Foreground(borderColor)
		versionStyle := dimStyle

		// Calculate dashes: actualWidth = 1 (╰) + dashes + versionLen + 1 (╯)
		versionLen := len(version)
		leftDashes := actualWidth - versionLen - 2
		if leftDashes < 1 {
			leftDashes = 1
		}

		bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", leftDashes)) +
			versionStyle.Render(version) +
			borderStyle.Render("╯")

		lines[len(lines)-1] = bottomBorder
	}
	content.WriteString(strings.Join(lines, "\n"))
	content.WriteString("\n\n")

	// Help text with styled keys (same as status bar)
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(lipgloss.Color(ColorPurple)).
		Bold(true).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorLightGray))

	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#444444"))

	sep := separatorStyle.Render(" │ ")

	helpItems := []string{
		keyStyle.Render("↑/↓") + descStyle.Render(" navigate"),
		keyStyle.Render("enter") + descStyle.Render(" select"),
		keyStyle.Render("n") + descStyle.Render(" new"),
		keyStyle.Render("e") + descStyle.Render(" rename"),
		keyStyle.Render("d") + descStyle.Render(" delete"),
		keyStyle.Render("i") + descStyle.Render(" import"),
		keyStyle.Render("q") + descStyle.Render(" quit"),
	}

	helpText := strings.Join(helpItems, sep)
	content.WriteString(helpText)

	// Center vertically and horizontally
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content.String())
}

// newProjectView renders the new project creation dialog
func (m Model) newProjectView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	boxContent.WriteString("  Project Name:\n")
	boxContent.WriteString("  " + m.projectInput.View() + "\n\n")
	boxContent.WriteString(helpStyle.Render("  enter: create  esc: cancel"))
	boxContent.WriteString("\n")

	// Use project select view as background
	background := m.projectSelectView()
	return m.renderOverlayDialogWithBackground(" New Project ", boxContent.String(), 50, ColorPurple, background)
}

// renameProjectView renders the project rename dialog
func (m Model) renameProjectView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	boxContent.WriteString("  New Name:\n")
	boxContent.WriteString("  " + m.projectInput.View() + "\n\n")
	boxContent.WriteString(helpStyle.Render("  enter: save  esc: cancel"))
	boxContent.WriteString("\n")

	// Use project select view as background
	background := m.projectSelectView()
	return m.renderOverlayDialogWithBackground(" Rename Project ", boxContent.String(), 50, ColorPurple, background)
}

// confirmDeleteProjectView renders the project deletion confirmation
func (m Model) confirmDeleteProjectView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")
	if m.deleteProjectTarget != nil {
		boxContent.WriteString(fmt.Sprintf("  Delete project '%s'?\n", m.deleteProjectTarget.Name))
		boxContent.WriteString("  All sessions in this project will be deleted.\n\n")
	}
	boxContent.WriteString(helpStyle.Render("  y: yes  n: no"))
	boxContent.WriteString("\n")

	// Use project select view as background
	background := m.projectSelectView()
	return m.renderOverlayDialogWithBackground(" Confirm Delete ", boxContent.String(), 50, ColorRed, background)
}

// confirmImportView renders the import confirmation dialog with default sessions as background
func (m Model) confirmImportView() string {
	var boxContent strings.Builder
	boxContent.WriteString("\n\n")

	// Load default sessions
	originalProject := m.storage.GetActiveProjectID()
	m.storage.SetActiveProject("")
	defaultInstances, defaultGroups, _ := m.storage.LoadAll()
	m.storage.SetActiveProject(originalProject)

	if m.importTarget != nil {
		boxContent.WriteString(fmt.Sprintf("  Import %d sessions into '%s'?\n\n", len(defaultInstances), m.importTarget.Name))
	}

	boxContent.WriteString(helpStyle.Render("  y: yes  n: no"))
	boxContent.WriteString("\n")

	// Create background showing default sessions
	background := m.renderDefaultSessionsBackground(defaultInstances, defaultGroups)

	return m.renderOverlayDialogWithBackground(" Confirm Import ", boxContent.String(), 50, ColorPurple, background)
}

// renderDefaultSessionsBackground renders a view of the default (no project) sessions
func (m Model) renderDefaultSessionsBackground(instances []*session.Instance, groups []*session.Group) string {
	listWidth := ListPaneWidth
	previewWidth := m.calculatePreviewWidth()
	contentHeight := m.height - 1
	if contentHeight < MinContentHeight {
		contentHeight = MinContentHeight
	}

	// Build left pane with "Sessions to Import" header
	var leftPane strings.Builder
	leftPane.WriteString("\n")

	header := titleStyle.Render(" Sessions to Import ")
	leftPane.WriteString(header)
	leftPane.WriteString("\n\n")

	if len(instances) == 0 {
		leftPane.WriteString(" No sessions\n")
	} else {
		for i, inst := range instances {
			if i >= contentHeight-5 {
				leftPane.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more\n", len(instances)-i)))
				break
			}
			var status string
			if inst.Status == session.StatusRunning {
				status = activeStyle.Render("●")
			} else {
				status = stoppedStyle.Render("○")
			}
			leftPane.WriteString(fmt.Sprintf("   %s %s\n", status, inst.Name))
		}
	}

	// Build right pane (empty preview)
	var rightPane strings.Builder
	rightPane.WriteString("\n")
	rightPane.WriteString(titleStyle.Render(" Preview "))
	rightPane.WriteString(dimStyle.Render(fmt.Sprintf(" %s v%s ", AppName, AppVersion)))
	rightPane.WriteString("\n\n")
	rightPane.WriteString(dimStyle.Render("  Select a project to import these sessions"))

	// Style the panes
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

	var b strings.Builder
	b.WriteString(content)
	b.WriteString("\n")

	return b.String()
}
