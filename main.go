package main

import (
	"fmt"
	"os"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	"github.com/bbc/infra-pipeline-ui/config"
	"github.com/bbc/infra-pipeline-ui/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Check for config
	if !config.Exists() {
		fmt.Println("No configuration found. Running first-time setup wizard...")
		_, err := config.RunWizard()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\nSetup complete! Starting TUI...")
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config: %v\n", err)
		os.Exit(1)
	}

	// Create client with cache
	client := bitbucket.NewClient(cfg.Username, cfg.AppPass)
	cachedClient := bitbucket.NewCachedClient(client)

	// Create and run model
	model := ui.NewModel(cfg, cachedClient)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
