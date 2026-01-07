package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/izll/agent-session-manager/session"
)

// handleMoveSessionUp moves the selected session or group up in the list
func (m *Model) handleMoveSessionUp() {
	// If groups exist, handle grouped reordering
	if len(m.groups) > 0 {
		m.buildVisibleItems()
		if m.cursor <= 0 || m.cursor >= len(m.visibleItems) {
			return
		}
		currentItem := m.visibleItems[m.cursor]
		prevItem := m.visibleItems[m.cursor-1]

		// Moving a group up
		if currentItem.isGroup {
			groupIdx := m.findGroupIndex(currentItem.group.ID)
			if groupIdx > 0 {
				m.groups[groupIdx], m.groups[groupIdx-1] = m.groups[groupIdx-1], m.groups[groupIdx]
				m.storage.SaveWithGroups(m.instances, m.groups)
				m.buildVisibleItems()
				// Find new cursor position
				for i, item := range m.visibleItems {
					if item.isGroup && item.group.ID == currentItem.group.ID {
						m.cursor = i
						break
					}
				}
			}
			return
		}

		// If previous is a group header, can't move further up
		if prevItem.isGroup {
			return
		}

		// Both are sessions - swap in the instances array (keep original groups)
		currentIdx := m.findInstanceIndex(currentItem.instance.ID)
		prevIdx := m.findInstanceIndex(prevItem.instance.ID)
		if currentIdx >= 0 && prevIdx >= 0 {
			m.instances[currentIdx], m.instances[prevIdx] = m.instances[prevIdx], m.instances[currentIdx]
			m.cursor--
			m.storage.Save(m.instances)
		}
		return
	}

	// Original behavior for non-grouped view
	if m.cursor > 0 && len(m.instances) > 1 {
		m.instances[m.cursor], m.instances[m.cursor-1] = m.instances[m.cursor-1], m.instances[m.cursor]
		m.cursor--
		m.storage.Save(m.instances)
	}
}

// handleMoveSessionDown moves the selected session or group down in the list
func (m *Model) handleMoveSessionDown() {
	// If groups exist, handle grouped reordering
	if len(m.groups) > 0 {
		m.buildVisibleItems()
		if m.cursor < 0 || m.cursor >= len(m.visibleItems)-1 {
			return
		}
		currentItem := m.visibleItems[m.cursor]
		nextItem := m.visibleItems[m.cursor+1]

		// Moving a group down
		if currentItem.isGroup {
			groupIdx := m.findGroupIndex(currentItem.group.ID)
			if groupIdx >= 0 && groupIdx < len(m.groups)-1 {
				m.groups[groupIdx], m.groups[groupIdx+1] = m.groups[groupIdx+1], m.groups[groupIdx]
				m.storage.SaveWithGroups(m.instances, m.groups)
				m.buildVisibleItems()
				// Find new cursor position
				for i, item := range m.visibleItems {
					if item.isGroup && item.group.ID == currentItem.group.ID {
						m.cursor = i
						break
					}
				}
			}
			return
		}

		// If next is a group header, can't move further down
		if nextItem.isGroup {
			return
		}

		// Both are sessions - swap in the instances array (keep original groups)
		currentIdx := m.findInstanceIndex(currentItem.instance.ID)
		nextIdx := m.findInstanceIndex(nextItem.instance.ID)
		if currentIdx >= 0 && nextIdx >= 0 {
			m.instances[currentIdx], m.instances[nextIdx] = m.instances[nextIdx], m.instances[currentIdx]
			m.cursor++
			m.storage.Save(m.instances)
		}
		return
	}

	// Original behavior for non-grouped view
	if m.cursor < len(m.instances)-1 {
		m.instances[m.cursor], m.instances[m.cursor+1] = m.instances[m.cursor+1], m.instances[m.cursor]
		m.cursor++
		m.storage.Save(m.instances)
	}
}

// findInstanceIndex finds the index of an instance in the instances array by ID
func (m *Model) findInstanceIndex(id string) int {
	for i, inst := range m.instances {
		if inst.ID == id {
			return i
		}
	}
	return -1
}

// findGroupIndex finds the index of a group in the groups array by ID
func (m *Model) findGroupIndex(id string) int {
	for i, g := range m.groups {
		if g.ID == id {
			return i
		}
	}
	return -1
}

// saveSettings saves UI settings to storage
func (m *Model) saveSettings() {
	m.storage.SaveSettings(&session.Settings{
		CompactList:     m.compactList,
		HideStatusLines: m.hideStatusLines,
		ShowAgentIcons:  m.showAgentIcons,
		SplitView:       m.splitView,
		MarkedSessionID: m.markedSessionID,
		Cursor:          m.cursor,
		SplitFocus:      m.splitFocus,
	})
}

// getScrollableContent returns the content to use for scrolling
func (m *Model) getScrollableContent() string {
	if m.scrollContent != "" {
		return m.scrollContent
	}
	return m.preview
}

// scrollPreviewUp scrolls the preview up by the given number of lines
func (m *Model) scrollPreviewUp(lines int) {
	// Fetch extended content on first scroll
	if m.scrollContent == "" {
		inst := m.getSelectedInstance()
		if inst != nil && inst.Status == session.StatusRunning {
			m.scrollContent, _ = inst.GetPreview(ScrollbackLines)
		}
	}
	content := m.getScrollableContent()
	if content == "" {
		return
	}
	contentLines := strings.Split(content, "\n")
	maxLines := m.getPreviewMaxLines()
	maxScroll := len(contentLines) - maxLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.previewScroll += lines
	if m.previewScroll > maxScroll {
		m.previewScroll = maxScroll
	}
}

// scrollPreviewDown scrolls the preview down by the given number of lines
func (m *Model) scrollPreviewDown(lines int) {
	m.previewScroll -= lines
	if m.previewScroll < 0 {
		m.previewScroll = 0
	}
}

// resetScroll resets scroll state when changing sessions
func (m *Model) resetScroll() {
	m.previewScroll = 0
	m.scrollContent = ""
}

// getPreviewMaxLines returns the maximum number of lines visible in the preview pane
func (m *Model) getPreviewMaxLines() int {
	contentHeight := m.height - 1
	if m.splitView {
		// In split view, each pane gets half the height
		halfHeight := (contentHeight - 1) / 2
		maxLines := halfHeight - 2 // -2 for header and margin in buildMiniPreview
		if maxLines < 2 {
			maxLines = 2
		}
		return maxLines
	}
	// Normal view
	maxLines := contentHeight - PreviewHeaderHeight
	if maxLines < MinPreviewLines {
		maxLines = MinPreviewLines
	}
	return maxLines
}

// navigatePinned changes the pinned session in split view
func (m *Model) navigatePinned(direction int) {
	if len(m.instances) == 0 {
		return
	}

	// If groups exist, use visible items
	if len(m.groups) > 0 {
		m.buildVisibleItems()

		// Find current pinned index in visible items (sessions only)
		currentIdx := -1
		for i, item := range m.visibleItems {
			if !item.isGroup && item.instance.ID == m.markedSessionID {
				currentIdx = i
				break
			}
		}

		if currentIdx == -1 {
			return
		}

		// Find next/previous session (skip groups)
		newIdx := currentIdx + direction
		for newIdx >= 0 && newIdx < len(m.visibleItems) {
			if !m.visibleItems[newIdx].isGroup {
				m.markedSessionID = m.visibleItems[newIdx].instance.ID
				return
			}
			newIdx += direction
		}
		return
	}

	// Original behavior for non-grouped view
	currentIdx := -1
	for i, inst := range m.instances {
		if inst.ID == m.markedSessionID {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		return
	}

	newIdx := currentIdx + direction
	if newIdx < 0 {
		newIdx = 0
	} else if newIdx >= len(m.instances) {
		newIdx = len(m.instances) - 1
	}

	m.markedSessionID = m.instances[newIdx].ID
}

// handleListKeys handles keyboard input in the main list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear error on any key press
	m.err = nil

	switch msg.String() {
	case "ctrl+c":
		m.saveSettings() // Save cursor position on quit
		m.storage.UnlockProject()
		return m, tea.Quit

	case "q":
		// Go back to project selector
		m.saveSettings()
		currentProjectID := ""
		if m.activeProject != nil {
			currentProjectID = m.activeProject.ID
		}
		m.storage.UnlockProject()
		// Reload projects to refresh session counts
		projectsData, _ := m.storage.LoadProjects()
		m.projects = projectsData.Projects
		// Find and select the project we just left
		// List structure: projects (0..n-1), "No project" (n), "New Project" (n+1)
		if currentProjectID == "" {
			// Default/no project
			m.projectCursor = len(m.projects)
		} else {
			// Find project index
			m.projectCursor = len(m.projects) // fallback to "No project"
			for i, p := range m.projects {
				if p.ID == currentProjectID {
					m.projectCursor = i
					break
				}
			}
		}
		m.state = stateProjectSelect
		return m, nil

	case "up", "k":
		// In split view with focus on pinned: change pinned session
		if m.splitView && m.splitFocus == 1 && m.markedSessionID != "" {
			m.navigatePinned(-1)
			m.saveSettings()
		} else if len(m.groups) > 0 {
			m.buildVisibleItems()
			if m.cursor > 0 {
				m.cursor--
				m.resetScroll()
				m.resizeSelectedPane()
			}
		} else if m.cursor > 0 {
			m.cursor--
			m.resetScroll()
			m.resizeSelectedPane()
		}

	case "down", "j":
		// In split view with focus on pinned: change pinned session
		if m.splitView && m.splitFocus == 1 && m.markedSessionID != "" {
			m.navigatePinned(1)
			m.saveSettings()
		} else if len(m.groups) > 0 {
			m.buildVisibleItems()
			if m.cursor < len(m.visibleItems)-1 {
				m.cursor++
				m.resetScroll()
				m.resizeSelectedPane()
			}
		} else if m.cursor < len(m.instances)-1 {
			m.cursor++
			m.resetScroll()
			m.resizeSelectedPane()
		}

	case "ctrl+up":
		m.handleMoveSessionUp()

	case "ctrl+down":
		m.handleMoveSessionDown()

	case "ctrl+left", "alt+left", "[":
		// Switch to previous tmux window/tab
		if inst := m.getSelectedInstance(); inst != nil {
			if inst.Status == session.StatusRunning {
				inst.PrevWindow()
			}
		}

	case "ctrl+right", "alt+right", "]":
		// Switch to next tmux window/tab
		if inst := m.getSelectedInstance(); inst != nil {
			if inst.Status == session.StatusRunning {
				inst.NextWindow()
			}
		}

	case "alt+up":
		// Scroll diff pane or preview up (1 line)
		if m.showDiff {
			m.diffPane.ScrollUp()
			break
		}
		m.scrollPreviewUp(1)

	case "alt+down":
		// Scroll diff pane or preview down (1 line)
		if m.showDiff {
			m.diffPane.ScrollDown()
			break
		}
		m.scrollPreviewDown(1)

	case "pgup", "alt+pgup":
		// Scroll diff pane or preview up (half page)
		if m.showDiff {
			m.diffPane.PageUp()
			break
		}
		halfPage := m.getPreviewMaxLines() / 2
		if halfPage < 5 {
			halfPage = 5
		}
		m.scrollPreviewUp(halfPage)

	case "pgdown", "alt+pgdown":
		// Scroll diff pane or preview down (half page)
		if m.showDiff {
			m.diffPane.PageDown()
			break
		}
		halfPage := m.getPreviewMaxLines() / 2
		if halfPage < 5 {
			halfPage = 5
		}
		m.scrollPreviewDown(halfPage)

	case "home":
		// Scroll to top
		if m.showDiff {
			m.diffPane.GotoTop()
			break
		}
		m.scrollPreviewUp(10000) // Large number to go to top

	case "end":
		// Scroll to bottom
		if m.showDiff {
			m.diffPane.GotoBottom()
			break
		}
		m.previewScroll = 0

	case "enter":
		// Check if a group is selected
		if len(m.groups) > 0 {
			m.buildVisibleItems()
			if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
				item := m.visibleItems[m.cursor]
				if item.isGroup {
					// Toggle collapse
					m.storage.ToggleGroupCollapsed(item.group.ID)
					groups, _ := m.storage.GetGroups()
					m.groups = groups
					m.buildVisibleItems()
					return m, nil
				}
			}
		}
		if cmd := m.handleEnterSession(); cmd != nil {
			return m, cmd
		}

	case "n":
		// Start new session flow: agent selection -> path -> name
		m.agentCursor = 0
		m.pendingAgent = session.AgentClaude
		m.pendingGroupID = m.getCurrentGroupID()
		m.state = stateSelectAgent
		return m, nil

	case "N":
		// Open notes editor for selected session or tab
		if inst := m.getSelectedInstance(); inst != nil {
			// Get current window index (0 = main, >0 = tab)
			windowIdx := 0
			if inst.Status == session.StatusRunning {
				windowIdx = inst.GetCurrentWindowIndex()
			}
			m.notesWindowIndex = windowIdx

			// Load notes for the current window
			if windowIdx == 0 {
				// Main session notes
				m.notesInput.SetValue(inst.Notes)
			} else {
				// Tab notes
				if fw := inst.GetFollowedWindow(windowIdx); fw != nil {
					m.notesInput.SetValue(fw.Notes)
				} else {
					m.notesInput.SetValue("")
				}
			}
			m.notesInput.Focus()
			m.state = stateNotes
			return m, nil
		}

	case "r":
		// Resume only works for agents that support it
		if inst := m.getSelectedInstance(); inst != nil {
			config := inst.GetAgentConfig()
			if !config.SupportsResume {
				m.err = fmt.Errorf("resume not supported for %s agent", inst.Agent)
				m.previousState = m.state // Save current state to return after error
				m.state = stateError
				return m, nil
			}
		}
		if err := m.handleResumeSession(); err != nil {
			m.err = err
			m.previousState = m.state // Save current state to return after error
			m.state = stateError
		}

	case "s":
		m.handleStartSession()

	case "a":
		// Show start mode selection (replace or parallel)
		if inst := m.getSelectedInstance(); inst != nil {
			m.state = stateSelectStartMode
		}

	case "x":
		// Stop session or tab
		if inst := m.getSelectedInstance(); inst != nil {
			if inst.Status == session.StatusRunning {
				m.stopTarget = inst
				// If multiple tabs, ask what to stop
				windows := inst.GetWindowList()
				if len(windows) > 1 {
					m.state = stateStopChoice
					return m, nil
				}
			}
			// Otherwise just confirm session stop
			m.handleStopSession()
		}

	case "d":
		// Check if a group is selected
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.isGroup {
				// Delete group
				if err := m.storage.RemoveGroup(item.group.ID); err != nil {
					m.err = err
				} else {
					// Reload groups
					groups, _ := m.storage.GetGroups()
					m.groups = groups
					m.buildVisibleItems()
					if m.cursor >= len(m.visibleItems) && m.cursor > 0 {
						m.cursor--
					}
				}
				return m, nil
			}
		}
		// Delete session or tab
		if inst := m.getSelectedInstance(); inst != nil {
			m.deleteTarget = inst
			// If running and has multiple tabs, ask what to delete
			if inst.Status == session.StatusRunning {
				windows := inst.GetWindowList()
				if len(windows) > 1 {
					m.state = stateDeleteChoice
					return m, nil
				}
			}
			// Otherwise just confirm session delete
			m.state = stateConfirmDelete
		}

	case "ctrl+y":
		if cmd := m.handleToggleAutoYes(); cmd != nil {
			return m, cmd
		}

	case "e":
		// Check if a group is selected
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.isGroup {
				// Rename group
				m.groupInput.SetValue(item.group.Name)
				m.groupInput.Focus()
				m.state = stateRenameGroup
				return m, textinput.Blink
			}
		}
		// Rename session
		if cmd := m.handleRenameSession(); cmd != nil {
			return m, cmd
		}

	case "?", "f1":
		m.state = stateHelp

	case "c":
		// Check if a group is selected
		if len(m.groups) > 0 {
			m.buildVisibleItems()
			if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
				item := m.visibleItems[m.cursor]
				if item.isGroup {
					m.handleGroupColorPicker(item.group)
					return m, nil
				}
			}
		}
		m.handleColorPicker()

	case "l":
		m.compactList = !m.compactList
		m.saveSettings()

	case "o":
		m.hideStatusLines = !m.hideStatusLines
		m.saveSettings()

	case "t":
		// Open new tmux tab/window - ask Agent or Terminal
		if inst := m.getSelectedInstance(); inst != nil {
			if inst.Status == session.StatusRunning {
				m.state = stateNewTabChoice
				return m, nil
			}
		}

	case "ctrl+f":
		// Toggle follow on current tab
		if inst := m.getSelectedInstance(); inst != nil {
			if inst.Status == session.StatusRunning {
				currentIdx := inst.GetCurrentWindowIndex()
				if currentIdx > 0 { // Can't unfollow window 0
					followed := inst.ToggleWindowFollow(currentIdx)
					m.storage.UpdateInstance(inst)
					if followed {
						m.err = fmt.Errorf("Tab is now tracked as agent")
					} else {
						m.err = fmt.Errorf("Tab is no longer tracked")
					}
					m.previousState = stateList
					m.state = stateError
					return m, nil
				}
			}
		}

	case "T":
		// Rename current tmux tab/window (only if multiple windows)
		if inst := m.getSelectedInstance(); inst != nil {
			if inst.Status == session.StatusRunning {
				windows := inst.GetWindowList()
				if len(windows) > 1 {
					for _, w := range windows {
						if w.Active {
							m.nameInput.SetValue(w.Name)
							m.nameInput.CursorEnd()
							m.nameInput.Focus()
							break
						}
					}
					m.state = stateRenameTab
					return m, textinput.Blink
				}
			}
		}

	case "W":
		// Close current tmux tab/window (not window 0)
		if inst := m.getSelectedInstance(); inst != nil {
			if inst.Status == session.StatusRunning {
				windows := inst.GetWindowList()
				for _, w := range windows {
					if w.Active && w.Index != 0 {
						if err := inst.CloseWindow(w.Index); err != nil {
							m.err = err
							m.previousState = stateList
							m.state = stateError
							return m, nil
						}
						// Refresh status bar (may hide tabs if only 1 window left)
						configureTmuxStatusBar(inst.TmuxSessionName(), inst.Name, inst.Color, inst.BgColor, inst.AutoYes)
						m.storage.UpdateInstance(inst)
						break
					}
				}
			}
		}

	case "D":
		// Toggle diff view in preview pane
		m.showDiff = !m.showDiff
		if m.showDiff {
			if inst := m.getSelectedInstance(); inst != nil {
				m.diffPane.SetDiff(inst)
			}
		}

	case "F":
		// Toggle diff mode (Session/Full) when in diff view
		if m.showDiff {
			m.diffPane.ToggleMode()
			if inst := m.getSelectedInstance(); inst != nil {
				m.diffPane.SetDiff(inst)
			}
		}

	case "I":
		m.showAgentIcons = !m.showAgentIcons
		m.saveSettings()

	case "v":
		m.splitView = !m.splitView
		m.splitFocus = 0 // Reset focus when toggling
		m.saveSettings()

	case "tab":
		// In split view: switch focus between panels
		if m.splitView && m.markedSessionID != "" {
			m.splitFocus = 1 - m.splitFocus // Toggle between 0 and 1
			m.saveSettings()
		} else {
			// In normal view: toggle group collapse
			m.buildVisibleItems()
			if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
				item := m.visibleItems[m.cursor]
				if item.isGroup {
					m.storage.ToggleGroupCollapsed(item.group.ID)
					// Reload groups
					groups, _ := m.storage.GetGroups()
					m.groups = groups
					m.buildVisibleItems()
				}
			}
		}

	case "m":
		// Mark current session for split view
		inst := m.getSelectedInstance()
		if inst != nil {
			if m.markedSessionID == inst.ID {
				m.markedSessionID = "" // Unmark if already marked
			} else {
				m.markedSessionID = inst.ID
			}
			m.saveSettings()
		}

	case "p":
		m.handleSendPrompt()

	case "R":
		m.handleForceResize()

	case "U":
		// Show update confirmation
		m.state = stateConfirmUpdate
		return m, nil

	case "g":
		// Create new group
		m.groupInput.SetValue("")
		m.groupInput.Focus()
		m.state = stateNewGroup
		return m, textinput.Blink

	case "G":
		// Assign session to group
		if len(m.instances) > 0 {
			// Find current session
			var inst *session.Instance
			if len(m.groups) > 0 {
				m.buildVisibleItems()
				if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
					item := m.visibleItems[m.cursor]
					if !item.isGroup {
						inst = item.instance
					}
				}
			} else if m.cursor < len(m.instances) {
				inst = m.instances[m.cursor]
			}

			// Pre-select current group
			m.groupCursor = 0 // Default to "No Group"
			if inst != nil && inst.GroupID != "" {
				for i, g := range m.groups {
					if g.ID == inst.GroupID {
						m.groupCursor = i + 1 // +1 because 0 is "No Group"
						break
					}
				}
			}
			m.state = stateSelectGroup
		}

	case "right":
		// Expand group
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.isGroup && item.group.Collapsed {
				m.storage.ToggleGroupCollapsed(item.group.ID)
				groups, _ := m.storage.GetGroups()
				m.groups = groups
				m.buildVisibleItems()
			}
		}

	case "left":
		// Collapse group
		m.buildVisibleItems()
		if m.cursor >= 0 && m.cursor < len(m.visibleItems) {
			item := m.visibleItems[m.cursor]
			if item.isGroup && !item.group.Collapsed {
				m.storage.ToggleGroupCollapsed(item.group.ID)
				groups, _ := m.storage.GetGroups()
				m.groups = groups
				m.buildVisibleItems()
			}
		}
	}

	return m, nil
}
