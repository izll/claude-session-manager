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
			if len(os.Args) < 4 {
				fmt.Fprintf(os.Stderr, "Usage: %s yolo <tmux-session-name> <window-index>\n", os.Args[0])
				os.Exit(1)
			}
			if err := toggleYolo(os.Args[2], os.Args[3]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "refresh-status":
			if len(os.Args) < 3 {
				os.Exit(1)
			}
			refreshStatusBar(os.Args[2])
			return
		case "yolo-confirm":
			if len(os.Args) < 5 {
				fmt.Fprintf(os.Stderr, "Usage: %s yolo-confirm <tmux-session> <window-index> <on|off>\n", os.Args[0])
				os.Exit(1)
			}
			enable := os.Args[4] == "on"
			if err := confirmYolo(os.Args[2], os.Args[3], enable); err != nil {
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

// refreshStatusBar updates the tmux status bar for a session
// Called from tmux hook when window changes
func refreshStatusBar(tmuxSessionName string) {
	// Load storage
	storage, err := session.NewStorage()
	if err != nil {
		return
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
				projInstances, _, _ := storage.LoadAll()
				for _, i := range projInstances {
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
		return
	}

	// Update status bar with full instance for per-window YOLO support
	ui.RefreshTmuxStatusBarFull(tmuxSessionName, inst.Name, inst.Color, inst.BgColor, inst)
}

// toggleYolo shows confirmation menu for toggling yolo mode on the active window
// Called from tmux binding when user presses Ctrl+Y inside a session
func toggleYolo(tmuxSessionName, windowIndex string) error {
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

	// Determine agent type for the active window
	var agentType session.AgentType
	var currentYolo bool

	if windowIndex == "0" {
		// Main window
		agentType = inst.Agent
		if agentType == "" {
			agentType = session.AgentClaude
		}
		currentYolo = inst.AutoYes
	} else {
		// Check if it's a followed window
		for _, fw := range inst.FollowedWindows {
			if fmt.Sprintf("%d", fw.Index) == windowIndex {
				agentType = fw.Agent
				currentYolo = fw.AutoYes
				break
			}
		}
		if agentType == "" {
			return fmt.Errorf("window %s is not a tracked agent", windowIndex)
		}
	}

	// Special handling for Gemini - just send Ctrl+Y (Gemini has its own confirmation)
	if agentType == session.AgentGemini {
		return inst.SendKeys("C-y")
	}

	// Terminal windows don't support YOLO
	if agentType == session.AgentTerminal {
		exec.Command("tmux", "display-message", "-t", tmuxSessionName, "Terminal windows don't support YOLO mode").Run()
		return nil
	}

	// Check if agent supports AutoYes
	config := session.AgentConfigs[agentType]
	if !config.SupportsAutoYes {
		exec.Command("tmux", "display-message", "-t", tmuxSessionName, fmt.Sprintf("YOLO mode not supported for %s", agentType)).Run()
		return nil
	}

	// Build confirmation menu
	var menuTitle, menuAction, newState string
	if currentYolo {
		menuTitle = "Disable YOLO mode?"
		menuAction = "Disable YOLO"
		newState = "off"
	} else {
		menuTitle = "Enable YOLO mode?"
		menuAction = "Enable YOLO"
		newState = "on"
	}

	// Show tmux menu for confirmation
	confirmCmd := fmt.Sprintf("asmgr yolo-confirm %s %s %s", tmuxSessionName, windowIndex, newState)
	exec.Command("tmux", "display-menu", "-t", tmuxSessionName,
		"-T", fmt.Sprintf(" %s ", menuTitle),
		fmt.Sprintf(" %s ", menuAction), "", fmt.Sprintf("run-shell '%s'", confirmCmd),
		" Cancel ", "", "",
	).Run()

	return nil
}

// confirmYolo performs the actual YOLO toggle after user confirmation
func confirmYolo(tmuxSessionName, windowIndex string, enableYolo bool) error {
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

	// Determine agent type and update YOLO state
	var agentType session.AgentType
	var isFollowedWindow bool
	var followedWindowIdx int

	if windowIndex == "0" {
		agentType = inst.Agent
		if agentType == "" {
			agentType = session.AgentClaude
		}
		inst.AutoYes = enableYolo
	} else {
		for idx, fw := range inst.FollowedWindows {
			if fmt.Sprintf("%d", fw.Index) == windowIndex {
				agentType = fw.Agent
				isFollowedWindow = true
				followedWindowIdx = idx
				inst.FollowedWindows[idx].AutoYes = enableYolo
				break
			}
		}
	}

	// Save changes
	if err := storage.UpdateInstance(inst); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	// Get agent config
	config := session.AgentConfigs[agentType]

	// Build restart command
	var args []string
	args = append(args, config.Command)
	if enableYolo && config.AutoYesFlag != "" {
		args = append(args, config.AutoYesFlag)
	}

	// Add resume flag if supported
	if config.SupportsResume && config.ResumeFlag != "" {
		if !isFollowedWindow && inst.ResumeSessionID != "" {
			args = append(args, config.ResumeFlag, inst.ResumeSessionID)
		} else if isFollowedWindow && inst.FollowedWindows[followedWindowIdx].ResumeSessionID != "" {
			args = append(args, config.ResumeFlag, inst.FollowedWindows[followedWindowIdx].ResumeSessionID)
		}
	}

	// Respawn the pane with new command
	target := fmt.Sprintf("%s:%s", tmuxSessionName, windowIndex)
	cmdStr := strings.Join(args, " ")
	exec.Command("tmux", "respawn-pane", "-t", target, "-k", cmdStr).Run()

	// Refresh status bar
	refreshStatusBar(tmuxSessionName)

	// Show confirmation message
	status := "OFF"
	if enableYolo {
		status = "ON"
	}
	exec.Command("tmux", "display-message", "-t", tmuxSessionName, fmt.Sprintf("YOLO mode %s for window %s", status, windowIndex)).Run()

	return nil
}

