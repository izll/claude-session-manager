package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// handleProjectSelectKeys handles keyboard input in the project selection view
func (m Model) handleProjectSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Total items: projects + "New Project" + "Continue without project"
	totalItems := len(m.projects) + 2

	switch msg.String() {
	case "q", "ctrl+c":
		m.storage.UnlockProject()
		return m, tea.Quit

	case "up", "k":
		if m.projectCursor > 0 {
			m.projectCursor--
		}

	case "down", "j":
		if m.projectCursor < totalItems-1 {
			m.projectCursor++
		}

	case "enter":
		if m.projectCursor < len(m.projects) {
			// Selected a project
			project := m.projects[m.projectCursor]
			if err := m.switchToProject(project); err != nil {
				m.previousState = stateProjectSelect
				m.err = err
				m.state = stateError
				return m, nil
			}
			m.state = stateList
		} else if m.projectCursor == len(m.projects) {
			// "Continue without project"
			if err := m.switchToProject(nil); err != nil {
				m.previousState = stateProjectSelect
				m.err = err
				m.state = stateError
				return m, nil
			}
			m.state = stateList
		} else {
			// "New Project" (last option)
			m.projectInput.Reset()
			m.projectInput.Focus()
			m.state = stateNewProject
			return m, textinput.Blink
		}

	case "n":
		// Shortcut for new project
		m.projectInput.Reset()
		m.projectInput.Focus()
		m.state = stateNewProject
		return m, textinput.Blink

	case "e":
		// Rename project
		if m.projectCursor < len(m.projects) {
			project := m.projects[m.projectCursor]
			m.projectInput.SetValue(project.Name)
			m.projectInput.Focus()
			m.state = stateRenameProject
			return m, textinput.Blink
		}

	case "d":
		// Delete project
		if m.projectCursor < len(m.projects) {
			m.deleteProjectTarget = m.projects[m.projectCursor]
			m.state = stateConfirmDeleteProject
		}

	case "i":
		// Import sessions from default (no project) into selected project
		if m.projectCursor < len(m.projects) {
			// Check if there are sessions to import
			defaultCount := m.storage.GetProjectSessionCount("")
			if defaultCount == 0 {
				m.previousState = stateProjectSelect
				m.err = fmt.Errorf("no sessions to import (default is empty)")
				m.state = stateError
				return m, nil
			}
			m.importTarget = m.projects[m.projectCursor]
			m.state = stateConfirmImport
		} else {
			m.previousState = stateProjectSelect
			m.err = fmt.Errorf("select a project first to import sessions into")
			m.state = stateError
		}

	case "U":
		// Check for updates
		m.previousState = stateProjectSelect
		m.state = stateConfirmUpdate
		return m, nil
	}

	return m, nil
}

// handleNewProjectKeys handles keyboard input when creating a new project
func (m Model) handleNewProjectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateProjectSelect
		return m, nil

	case "enter":
		name := strings.TrimSpace(m.projectInput.Value())
		if name == "" {
			m.previousState = stateNewProject
			m.err = fmt.Errorf("project name cannot be empty")
			m.state = stateError
			return m, nil
		}

		project, err := m.storage.AddProject(name)
		if err != nil {
			m.previousState = stateNewProject
			m.err = err
			m.state = stateError
			return m, nil
		}

		// Reload projects list
		projectsData, _ := m.storage.LoadProjects()
		m.projects = projectsData.Projects

		// Switch to the new project
		if err := m.switchToProject(project); err != nil {
			m.previousState = stateProjectSelect
			m.err = err
			m.state = stateError
			return m, nil
		}

		m.state = stateList
		return m, nil
	}

	var cmd tea.Cmd
	m.projectInput, cmd = m.projectInput.Update(msg)
	return m, cmd
}

// handleRenameProjectKeys handles keyboard input when renaming a project
func (m Model) handleRenameProjectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateProjectSelect
		return m, nil

	case "enter":
		name := strings.TrimSpace(m.projectInput.Value())
		if name == "" {
			m.previousState = stateRenameProject
			m.err = fmt.Errorf("project name cannot be empty")
			m.state = stateError
			return m, nil
		}

		if m.projectCursor < len(m.projects) {
			project := m.projects[m.projectCursor]
			if err := m.storage.RenameProject(project.ID, name); err != nil {
				m.previousState = stateRenameProject
				m.err = err
				m.state = stateError
				return m, nil
			}
			project.Name = name
		}

		m.state = stateProjectSelect
		return m, nil
	}

	var cmd tea.Cmd
	m.projectInput, cmd = m.projectInput.Update(msg)
	return m, cmd
}

// handleConfirmDeleteProjectKeys handles keyboard input in the project deletion confirmation
func (m Model) handleConfirmDeleteProjectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.deleteProjectTarget != nil {
			if err := m.storage.RemoveProject(m.deleteProjectTarget.ID); err != nil {
				m.previousState = stateProjectSelect
				m.err = err
				m.state = stateError
				return m, nil
			}

			// Reload projects list
			projectsData, _ := m.storage.LoadProjects()
			m.projects = projectsData.Projects

			// Adjust cursor if needed
			if m.projectCursor >= len(m.projects) && m.projectCursor > 0 {
				m.projectCursor--
			}
		}
		m.deleteProjectTarget = nil
		m.state = stateProjectSelect

	case "n", "N", "esc":
		m.deleteProjectTarget = nil
		m.state = stateProjectSelect
	}

	return m, nil
}

// handleConfirmImportKeys handles keyboard input in the import confirmation
func (m Model) handleConfirmImportKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.importTarget != nil {
			count, err := m.storage.ImportDefaultSessions(m.importTarget.ID)
			m.previousState = stateProjectSelect
			if err != nil {
				m.err = err
				m.state = stateError
			} else {
				m.err = fmt.Errorf("successfully imported %d sessions into '%s'", count, m.importTarget.Name)
				m.state = stateError // Use dialog for success message too
			}
		}
		m.importTarget = nil

	case "n", "N", "esc":
		m.importTarget = nil
		m.state = stateProjectSelect
	}

	return m, nil
}
