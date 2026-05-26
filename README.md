# Bitbucket Pipeline TUI

A terminal-based UI for browsing and triggering Bitbucket Pipelines, built with TView

## Features

- Browse pipelines across multiple Bitbucket repositories
- Filter pipelines by build number, branch name, type, or status
- View detailed pipeline information including variables
- Inspect step logs with search functionality
- Trigger new pipeline runs with custom branch, selector, and variables
- Multi-project support with tab switching

## Installation

```bash
go build -o bbc-pipeline-tui .
```

## Configuration

On first run, a setup wizard will guide you through configuring:

- Bitbucket username and app password
- One or more workspace/repo_slug pairs

The config is stored at `~/.config/bbc-pipeline-tui.yml`.

## Usage

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Navigate pipeline list |
| `/` | Filter pipelines |
| `enter` | View pipeline details |
| `r` | Refresh pipeline list |
| `n` | Load next page |
| `tab` | Switch project |
| `l` (in detail view) | View step logs |
| `enter` (in detail view) | Trigger new pipeline run |
| `esc` | Go back |
| `q` or `ctrl+c` | Quit |

## Project Structure

```
.
├── main.go              # Entry point
├── bitbucket/
│   ├── client.go        # HTTP client, auth, error handling
│   └── pipelines.go     # Pipeline/step types and API methods
├── config/
│   ├── config.go        # Config loading/saving
│   └── wizard.go        # First-run setup wizard
└── ui/
    ├── types.go         # Model struct, screen constants, messages
    ├── styles.go        # Lipgloss style definitions
    ├── helpers.go       # Utility functions (filteredPipelines, truncate, etc.)
    ├── commands.go      # Async fetch/trigger tea.Cmd functions
    ├── update.go        # NewModel, Init, Update router, handleLoading/Error/Esc
    ├── view.go          # View router, viewHeader, viewLoading, viewError, renderPage
    ├── list.go          # Pipeline list update/view
    ├── detail.go        # Pipeline detail update/view
    ├── logs.go          # Step log list & view (with search)
    └── run.go           # Trigger pipeline form update/view