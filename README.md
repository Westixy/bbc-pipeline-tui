# infra-pipeline-ui

A terminal-based TUI for browsing, inspecting, and triggering Bitbucket Pipelines, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- **Pipeline list** — browse pipelines across Bitbucket repositories with status/cache-aware loading.
- **Filtering** — filter by build number, branch name, pipeline type, or status (`/`).
- **Detail view** — inspect pipeline steps, parsed log variables, and run metadata.
- **Step logs** — view step output inline with search highlighting.
- **Trigger runs** — kick off pipeline runs with a custom branch, selector, and runtime variables (add/edit/delete).
- **Multi-project favourites** — save workspace/repo pairs (`p`) and switch between them (`tab`).
- **Project manager** (`p`) — two-pane browser to add repos from any workspace or remove favourites.
- **Config wizard** — interactive first-run setup for Bitbucket credentials and projects.
- **API cache** — pipelines and repositories are cached locally to reduce API calls.

## Quick start

### Nix

```bash
nix run .
```

Or build the binary:

```bash
nix build .
./result/bin/infra-pipeline-ui
```

### Go (dev shell)

```bash
nix develop   # or ensure Go + golangci-lint are on PATH
go build -o infra-pipeline-ui .
./infra-pipeline-ui
```

## Configuration

On first run, a setup wizard collects:

- Bitbucket username and **app password**
- One or more `workspace/repo_slug` pairs

Config is written to `~/.config/infra-pipeline-ui.yml`.

## Usage

| Key              | Action                                  |
| ---------------- | --------------------------------------- |
| `↑` `↓` / `j` `k` | Navigate list                          |
| `/`              | Filter pipelines                        |
| `enter`          | View pipeline detail                    |
| `r`              | Refresh pipeline list                   |
| `n`              | Load next page of pipelines             |
| `tab`            | Switch active project                   |
| `p`              | Open **project manager**                |
| `esc`            | Go back (detail → list / manager → list) |
| `q` / `ctrl+c`   | Quit                                    |

### Detail view extra keys

| Key     | Action                      |
| ------- | --------------------------- |
| `l`     | View step logs              |
| `enter` | Trigger a new pipeline run  |
| `esc`   | Back to pipeline list       |

### Project manager extra keys

| Key           | Action                                    |
| ------------- | ----------------------------------------- |
| `tab`         | Cycle panes: favourites → repos → input   |
| `enter`       | Select favourite / add repo to favourites |
| `d`           | Remove from favourites                    |
| `r`           | Refresh workspace repo list               |
| `esc`         | Step back (input → repos → favs → list)   |

## Project structure

```
.
├── main.go
├── bitbucket/
│   ├── client.go          # HTTP client, auth, shared types (PipelineVariable, Repository, …)
│   ├── pipelines.go       # Pipeline/step API methods
│   └── cache.go           # In-memory cache + rate-limit client wrapper
├── config/
│   ├── config.go          # Config file load/save + project helpers
│   └── wizard.go          # First-run setup wizard
├── ui/
│   ├── types.go           # Model struct, screen constants, messages
│   ├── styles.go          # Lipgloss style definitions
│   ├── helpers.go         # Utilities (filter, truncate, paginate, width)
│   ├── commands.go        # Async fetch/trigger tea.Cmd functions
│   ├── update.go          # NewModel, Init, Update router, global key handlers
│   ├── view.go            # View router, header, loading, error, content layout
│   ├── list.go            # Pipeline list update/view
│   ├── detail.go          # Pipeline detail update/view
│   ├── logs.go            # Step log view with search
│   ├── run.go             # Trigger pipeline form update/view
│   └── manage_projects.go # Two-pane project manager update/view
├── flake.nix              # Nix flake: dev shell, package, app
├── Makefile               # Convenience targets (build, test, lint, etc.)
├── bitbucket.openapi.json # Bitbucket API spec (documentation only)
└── README.md
```

## Nix flake

| Command          | Description                              |
| ---------------- | ---------------------------------------- |
| `nix develop`    | Enter dev shell with Go + golangci-lint  |
| `nix build`      | Build release binary                     |
| `nix run .`      | Build and run the TUI                    |
| `nix flake check`| Validate the flake                       |