package ui

import "github.com/charmbracelet/lipgloss"

// UI styles for the TUI components
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4"))

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")) // Orange for activity

	idleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")) // Grey for idle

	stoppedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87"))

	previewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	sessionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700"))

	listPaneStyle = lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.Border{Right: "│"}).
			BorderForeground(lipgloss.Color("#555555"))

	previewPaneStyle = lipgloss.NewStyle()

	listSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7D56F4")).
				Bold(true)

	searchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555555")).
			Foreground(lipgloss.Color("#666666")).
			Padding(0, 1)

	selectedPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	metaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
)
