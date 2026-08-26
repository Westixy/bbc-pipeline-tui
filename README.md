# infra-pipeline-ui

A terminal-based TUI for browsing, inspecting, and triggering Bitbucket Pipelines, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Why

I built this because the original Bitbucket Pipelines UI was missing some
features I consider mandatory. Filtering pipeline history is clunky, running a
pipeline with custom variables requires hunting through confusing menus, and
there's no quick way to see the status of every pipeline across all your
repositories at a glance.

What pushed me over the edge was Atlassian's issue tracker: for **years**, open
tickets for these exact features gathered dozens of upvotes and "me too"
comments from users asking for the same thing — and nothing happened. The
tickets just sat there, some of them for the better part of a decade, while the
community kept asking. I got fed up waiting and decided to build the UI I
wanted myself.

So this is a faster, keyboard-first interface that does what the official UI
should have done all along: filter and search pipelines instantly, trigger runs
with custom branch/selector/variables in a few keystrokes, and view running
pipelines across every project in one place.

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

Config is written to `~/.config/bbc-pipeline-tui.yml` (override the location with
the `BBC_CONFIG` environment variable).

## Docker

> 📖 **Docs:**
> - [`USERGUIDE.md`](USERGUIDE.md) — step-by-step guide to running the webapp via `bbc.sh`.
> - [`WEBAPP_GUIDE.md`](WEBAPP_GUIDE.md) — full description of the webapp frontend's features.

A `Dockerfile` is included that uses **Nix as the build stage** (producing a
statically-linked Go binary with the embedded webapp) and a **`scratch` runtime
image** containing only the binary. The config is expected at `/config.yml`
inside the container.

Manage the container lifecycle with the helper scripts:

```bash
./bbc.sh build    # build the image
./bbc.sh start    # build the image if missing, then run the container (webapp mode)
./bbc.sh status   # show container state + health check
./bbc.sh stop     # stop and remove the container
./bbc.sh restart  # stop then start
./bbc.sh logs     # follow container logs
```

Windows: `bbc.bat build|start|stop|restart|status|logs`.

The scripts mount `./config.yml` (host) at `/config.yml` in the container. If that
file does not exist, `start` prompts interactively: run the **first-time setup
wizard** (inside a throwaway container, writing the result to the host path) or
point at an existing file with `BBC_CONFIG_FILE=/path/to/config.yml`. The webapp
is published on `http://localhost:8080`; override with `BBC_PORT`, `BBC_IMAGE` or
`BBC_CONTAINER` as needed.

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