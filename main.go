package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	"github.com/bbc/infra-pipeline-ui/config"
	"github.com/bbc/infra-pipeline-ui/server"
	"github.com/bbc/infra-pipeline-ui/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Parse flags
	webappAddr := flag.String("webapp", "", "Run as web application API with embedded Svelte frontend (e.g. localhost:8080)")
	flag.Parse()

	// Check for config
	if !config.Exists() {
		fmt.Println("No configuration found. Running first-time setup wizard...")
		_, err := config.RunWizard()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\nSetup complete!")
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

	// --- WebApp mode ---
	if *webappAddr != "" {
		srv := server.New(cfg, cachedClient)

		fmt.Printf("Starting webapp on http://%s\n", *webappAddr)
		if err := http.ListenAndServe(*webappAddr, srv.Handler()); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// --- TUI mode (default) ---
	fmt.Println("Starting TUI...")

	// Create and run model
	model := ui.NewModel(cfg, cachedClient)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
