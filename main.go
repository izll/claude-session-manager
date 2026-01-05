package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/izll/agent-session-manager/session"
	"github.com/izll/agent-session-manager/ui"
	"github.com/izll/agent-session-manager/updater"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("%s version %s\n", ui.AppName, ui.AppVersion)
			return
		case "--update", "-u":
			if err := runUpdate(); err != nil {
				fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
				os.Exit(1)
			}
			return
		case "--help", "-h":
			printHelp()
			return
		case "yolo":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: %s yolo <tmux-session-name>\n", os.Args[0])
				os.Exit(1)
			}
			if err := toggleYolo(os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	model, err := ui.NewModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`%s - Agent Session Manager

Usage: %s [options]

Options:
  -v, --version    Show version
  -u, --update     Update to latest version
  -h, --help       Show this help

Run without arguments to start the TUI.
`, ui.AppName, ui.AppName)
}

func runUpdate() error {
	fmt.Printf("Current version: %s\n", ui.AppVersion)
	fmt.Println("Checking for updates...")

	latest := updater.CheckForUpdate(ui.AppVersion)
	if latest == "" {
		fmt.Printf("Already up to date (%s)\n", ui.AppVersion)
		return nil
	}

	fmt.Printf("New version available: %s\n", latest)
	fmt.Print("Update now? [Y/n] ")

	var answer string
	fmt.Scanln(&answer)
	if strings.ToLower(answer) == "n" {
		fmt.Println("Update cancelled")
		return nil
	}

	fmt.Println("Downloading...")
	if err := updater.DownloadAndInstall(latest); err != nil {
		return err
	}

	fmt.Printf("Updated to %s\n", latest)
	return nil
}

// toggleYolo toggles yolo mode for a session by tmux session name
// Called from tmux binding when user presses Ctrl+Y inside a session
func toggleYolo(tmuxSessionName string) error {
	// Load storage
	storage, err := session.NewStorage()
	if err != nil {
		return fmt.Errorf("failed to load storage: %w", err)
	}

	// Search in all projects for the session
	var inst *session.Instance

	// First check default project
	instances, _, _ := storage.LoadAll()
	for _, i := range instances {
		if i.TmuxSessionName() == tmuxSessionName {
			inst = i
			break
		}
	}

	// If not found, search in other projects
	if inst == nil {
		projectsData, err := storage.LoadProjects()
		if err == nil && projectsData != nil {
			for _, project := range projectsData.Projects {
				storage.SetActiveProject(project.ID)
				instances, _, _ := storage.LoadAll()
				for _, i := range instances {
					if i.TmuxSessionName() == tmuxSessionName {
						inst = i
						break
					}
				}
				if inst != nil {
					break
				}
			}
		}
	}

	if inst == nil {
		return fmt.Errorf("session not found: %s", tmuxSessionName)
	}

	// Get agent type
	agentType := inst.Agent
	if agentType == "" {
		agentType = session.AgentClaude
	}

	// Special handling for Gemini - just send Ctrl+Y
	if agentType == session.AgentGemini {
		return inst.SendKeys("C-y")
	}

	// Check if agent supports AutoYes
	config := session.AgentConfigs[agentType]
	if !config.SupportsAutoYes {
		return fmt.Errorf("yolo mode not supported for %s", agentType)
	}

	// Toggle AutoYes
	inst.AutoYes = !inst.AutoYes
	if err := storage.UpdateInstance(inst); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	// Build restart command
	var args []string
	args = append(args, config.Command)
	if inst.AutoYes && config.AutoYesFlag != "" {
		args = append(args, config.AutoYesFlag)
	}
	// Add --resume with session ID to continue the specific conversation
	if config.SupportsResume && config.ResumeFlag != "" && inst.ResumeSessionID != "" {
		args = append(args, config.ResumeFlag, inst.ResumeSessionID)
	}

	// Update tmux window name and status bar to show yolo status
	windowName := inst.Name
	if inst.AutoYes {
		windowName = " ! " + inst.Name
		// Orange/yellow status bar for yolo mode
		exec.Command("tmux", "set-option", "-t", tmuxSessionName, "status-style", "bg=colour208,fg=black").Run()
	} else {
		// Reset to default status bar
		exec.Command("tmux", "set-option", "-t", tmuxSessionName, "status-style", "bg=green,fg=black").Run()
	}
	exec.Command("tmux", "rename-window", "-t", tmuxSessionName, windowName).Run()

	// Kill current process and respawn pane with new command
	cmdStr := strings.Join(args, " ")
	exec.Command("tmux", "respawn-pane", "-t", tmuxSessionName, "-k", cmdStr).Run()

	return nil
}
