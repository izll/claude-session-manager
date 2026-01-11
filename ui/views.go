package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// truncateRunes truncates a string to maxLen runes and adds ellipsis if needed
func truncateRunes(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

// View implements tea.Model and renders the current UI state.
// It returns different views based on the current application state.
func (m Model) View() string {
	switch m.state {
	case stateProjectSelect:
		return m.projectSelectView()
	case stateNewProject:
		return m.newProjectView()
	case stateConfirmDeleteProject:
		return m.confirmDeleteProjectView()
	case stateConfirmImport:
		return m.confirmImportView()
	case stateRenameProject:
		return m.renameProjectView()
	case stateHelp:
		return m.helpView()
	case stateConfirmDelete:
		return m.confirmDeleteView()
	case stateConfirmStop:
		return m.confirmStopView()
	case stateSelectStartMode:
		return m.selectStartModeView()
	case stateConfirmStart:
		return m.confirmStartView()
	case stateNewName, stateNewPath:
		return m.newInstanceView()
	case stateRename:
		return m.renameView()
	case stateSelectAgentSession:
		return m.selectSessionView()
	case stateColorPicker:
		return m.colorPickerView()
	case statePrompt:
		return m.promptView()
	case stateNewGroup:
		return m.newGroupView()
	case stateRenameGroup:
		return m.renameGroupView()
	case stateSelectGroup:
		return m.selectGroupView()
	case stateSelectAgent:
		return m.selectAgentView()
	case stateCustomCmd:
		return m.customCmdView()
	case stateError:
		return m.errorView()
	case stateConfirmUpdate:
		return m.confirmUpdateView()
	case stateCheckingUpdate:
		return m.checkingUpdateView()
	case stateUpdating:
		return m.updatingView()
	case stateUpdateSuccess:
		return m.updateSuccessView()
	case stateNotes:
		return m.notesView()
	case stateNewTabChoice:
		return m.newTabChoiceView()
	case stateNewTabAgent:
		return m.newTabAgentView()
	case stateNewTab:
		return m.newTabView()
	case stateRenameTab:
		return m.renameTabView()
	case stateDeleteChoice:
		return m.deleteChoiceView()
	case stateConfirmDeleteTab:
		return m.confirmDeleteTabView()
	case stateStopChoice:
		return m.stopChoiceView()
	case stateConfirmStopTab:
		return m.confirmStopTabView()
	case stateConfirmYolo:
		return m.confirmYoloView()
	case stateSearch:
		return m.searchView()
	case stateGlobalSearchLoading:
		return m.globalSearchLoadingView()
	case stateGlobalSearch:
		return m.globalSearchView()
	case stateForkDialog:
		return m.forkDialogView()
	case stateGlobalSearchAction:
		return m.globalSearchActionView()
	case stateGlobalSearchConfirmJump:
		return m.globalSearchConfirmJumpView()
	case stateGlobalSearchNewName:
		return m.globalSearchNewNameView()
	case stateGlobalSearchSelectMatch:
		return m.globalSearchSelectMatchView()
	default:
		return m.listView()
	}
}

// searchView renders the search input overlay on top of the list view
func (m Model) searchView() string {
	// Render normal list view first
	listWidth := ListPaneWidth
	previewWidth := m.calculatePreviewWidth()
	contentHeight := m.height - 2 // Leave room for search bar

	if contentHeight < MinContentHeight {
		contentHeight = MinContentHeight
	}

	// Build panes
	leftPane := m.buildSessionListPane(listWidth, contentHeight)

	var rightPane string
	if m.splitView {
		rightPane = m.buildSplitPreviewPane(contentHeight)
	} else {
		rightPane = m.buildPreviewPane(contentHeight)
	}

	// Style the panes
	leftStyled := listPaneStyle.
		Width(listWidth).
		Height(contentHeight).
		Render(leftPane)

	rightStyled := previewPaneStyle.
		Width(previewWidth).
		Height(contentHeight).
		Render(rightPane)

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)

	// Build search bar
	searchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorWhite)).
		Background(lipgloss.Color("#333333")).
		Width(m.width).
		Padding(0, 1)

	searchBar := searchStyle.Render(m.searchInput.View())

	// Combine
	var b strings.Builder
	b.WriteString(content)
	b.WriteString("\n")
	b.WriteString(searchBar)

	return b.String()
}

// listView renders the main split-pane view with session list and preview
func (m Model) listView() string {
	listWidth := ListPaneWidth
	previewWidth := m.calculatePreviewWidth()
	contentHeight := m.height - 1
	if contentHeight < MinContentHeight {
		contentHeight = MinContentHeight
	}

	// Build panes using helper methods
	leftPane := m.buildSessionListPane(listWidth, contentHeight)

	var rightPane string
	if m.splitView {
		rightPane = m.buildSplitPreviewPane(contentHeight)
	} else {
		rightPane = m.buildPreviewPane(contentHeight)
	}

	// Style the panes with borders
	leftStyled := listPaneStyle.
		Width(listWidth).
		Height(contentHeight).
		Render(leftPane)

	rightStyled := previewPaneStyle.
		Width(previewWidth).
		Height(contentHeight).
		Render(rightPane)

	// Join panes horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)

	// Build final view
	var b strings.Builder
	b.WriteString(content)

	// Status bar
	b.WriteString(m.buildStatusBar())

	return b.String()
}

