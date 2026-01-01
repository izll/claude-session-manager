package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
