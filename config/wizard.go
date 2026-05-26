package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// RunWizard runs the first-run setup wizard interactively.
func RunWizard() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	cfg := &Config{}

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║   Bitbucket Pipeline TUI - First Setup   ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("This wizard will create your configuration at ~/.config/bbc-pipeline-tui.yml")
	fmt.Println()

	// Username
	fmt.Print("Enter your Bitbucket username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read username: %w", err)
	}
	cfg.Username = strings.TrimSpace(username)
	if cfg.Username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	// App Password
	fmt.Print("Enter your Bitbucket app password: ")
	appPass, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read app password: %w", err)
	}
	cfg.AppPass = strings.TrimSpace(appPass)
	if cfg.AppPass == "" {
		return nil, fmt.Errorf("app password cannot be empty")
	}

	// Projects
	fmt.Println()
	fmt.Println("Now add one or more Bitbucket projects (workspace/repo_slug pairs).")
	for {
		fmt.Println()
		fmt.Print("Enter workspace slug (or press Enter to finish): ")
		workspace, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read workspace: %w", err)
		}
		workspace = strings.TrimSpace(workspace)
		if workspace == "" {
			if len(cfg.Projects) == 0 {
				fmt.Println("At least one project is required.")
				continue
			}
			break
		}

		fmt.Print("Enter repository slug: ")
		repoSlug, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read repo_slug: %w", err)
		}
		repoSlug = strings.TrimSpace(repoSlug)
		if repoSlug == "" {
			fmt.Println("Repository slug cannot be empty.")
			continue
		}

		cfg.Projects = append(cfg.Projects, Project{
			Workspace: workspace,
			RepoSlug:  repoSlug,
		})
		fmt.Printf("Added: %s/%s\n", workspace, repoSlug)
	}

	// Save
	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	path, _ := DefaultPath()
	fmt.Println()
	fmt.Printf("Configuration saved to %s\n", path)

	// Verify can parse
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func init() {
	// Prevent unused import error for strconv
	_ = strconv.Itoa
}