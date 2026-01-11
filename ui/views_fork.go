package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// forkDialogView renders the fork dialog overlay
func (m Model) forkDialogView() string {
	var content strings.Builder

	// Show what we're forking
	if m.forkTarget != nil {
		sourceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
		content.WriteString("\n\n")
		content.WriteString(sourceStyle.Render(fmt.Sprintf("Forking from: %s", m.forkTarget.Name)))
		content.WriteString("\n\n")
	}

	// Name input
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWhite))
	content.WriteString(labelStyle.Render("Fork name:"))
	content.WriteString("\n")
	content.WriteString(m.forkNameInput.View())
	content.WriteString("\n\n")

	// Destination choice
	content.WriteString(labelStyle.Render("Fork to:"))
	content.WriteString("\n\n")

	// Tab option
	tabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	sessionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	if m.forkToTab {
		tabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWhite)).Bold(true)
	} else {
		sessionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWhite)).Bold(true)
	}

	tabIndicator := "  "
	sessionIndicator := "  "
	if m.forkToTab {
		tabIndicator = "► "
	} else {
		sessionIndicator = "► "
	}

	content.WriteString(tabStyle.Render(tabIndicator + "New Tab"))
	content.WriteString("\n")
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	content.WriteString(descStyle.Render("    Fork as a new tab in this session"))
	content.WriteString("\n\n")

	content.WriteString(sessionStyle.Render(sessionIndicator + "New Session"))
	content.WriteString("\n")
	content.WriteString(descStyle.Render("    Fork as a separate session"))
	content.WriteString("\n\n")

	// Footer
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))
	content.WriteString(footerStyle.Render("Tab: switch • Enter: fork • ESC: cancel"))

	// Render as overlay dialog
	boxWidth := 50
	if m.width > 80 {
		boxWidth = 55
	}

	return m.renderOverlayDialog(" Fork Session ", content.String(), boxWidth, ColorPurple)
}
