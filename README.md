# infra-pipeline-ui

A faster interface for browsing, inspecting, and triggering Bitbucket Cloud pipelines. It ships **two frontends** backed by a single Go codebase:

- **Terminal UI (TUI)** — built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss). This is the default mode.
- **Web app** — a [Svelte 5](https://svelte.dev) single-page app served by an embedded Go HTTP server. Selected with the `--webapp` flag.

Both frontends proxy the Bitbucket Cloud REST API v2.0 through the same Go client, authenticating with a Bitbucket **app password** over HTTP Basic Auth. You can list, filter, and inspect pipelines, read step logs, trigger and stop runs, and manage favourite `workspace/repo_slug` projects.

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

- **Pipeline list** — browse pipelines across your repositories with status-aware loading.
- **Filtering & search** — filter by build number, branch, pipeline type, or status; search step logs with match highlighting.
- **Detail view** — inspect pipeline steps, parsed log variables, and run metadata.
- **Step logs** — view step output inline, with ANSI colour rendering and live auto-refresh.
- **Trigger runs** — start a pipeline with a custom branch, selector, and runtime variables.
- **Stop runs** — stop an in-progress pipeline.
- **Running pipelines** — see all running pipelines across every configured project in one place.
- **Multi-project favourites** — save `workspace/repo_slug` pairs and switch between them.
- **Project manager** — add repositories from any workspace or remove favourites, with autocomplete.
- **Config wizard** — interactive first-run setup for Bitbucket credentials and projects.

Every capability is available in both the TUI and the web app; they differ only in how you drive them (keyboard vs. browser).

## Quick start

### Docker (recommended)

No Go, Node, or Nix needed locally — the image is built inside Docker using Nix.

```bash
./bbc.sh start     # build the image if needed, then run the webapp
./bbc.sh status    # confirm it's healthy
./bbc.sh stop      # shut down
```

Then open http://localhost:8080. Windows: `bbc.bat start|stop|status`. See [Docker](#docker) and the [User Guide](USERGUIDE.md) for details.

### Nix

```bash
nix run .                                 # run the TUI
nix run . -- --webapp localhost:8080      # run the webapp
```

Or build the binary:

```bash
nix build .
./result/bin/infra-pipeline-ui                          # TUI
./result/bin/infra-pipeline-ui --webapp localhost:8080  # webapp
```

### Go (dev shell)

```bash
nix develop   # or ensure Go + Node are on PATH

# Build the webapp frontend first (required for the embedded webapp):
(cd webapp && npm install && npm run build)

go build -o infra-pipeline-ui .
./infra-pipeline-ui                          # TUI
./infra-pipeline-ui --webapp localhost:8080  # webapp
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

### TUI keybindings

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

### Web app

Open the URL printed on start (default `http://localhost:8080`). For a complete description of the webapp's screens, features, and keyboard shortcuts, see [WEBAPP_GUIDE.md](WEBAPP_GUIDE.md).

## Project structure

```
.
├── main.go                   # Entry point: loads config, chooses TUI vs webapp mode
├── go.mod / go.sum           # Go module (github.com/bbc/infra-pipeline-ui)
├── flake.nix / flake.lock    # Nix flake: dev shell, package, app
├── Dockerfile                # Nix build stage → scratch runtime image
├── bbc.sh / bbc.bat          # Container lifecycle scripts (build/start/stop/…)
├── bitbucket/
│   ├── client.go             # HTTP client, Basic auth, shared types, pagination
│   ├── pipelines.go          # Pipeline/step/log/variable API methods
│   ├── repositories.go       # Workspace repository listing
│   ├── cached_client.go      # CachedClient: in-memory TTL cache wrapper
│   └── cache.go              # Thread-safe TTL cache implementation
├── config/
│   ├── config.go             # YAML load/save, Project type, validation, helpers
│   └── wizard.go             # Interactive first-run setup wizard
├── ui/                       # Bubble Tea TUI
│   ├── types.go              # Model struct, screen/state constants
│   ├── update.go             # NewModel, Init, Update router, global key handling
│   ├── view.go               # View router, header/help/content layout
│   ├── commands.go           # Async tea.Cmd functions + result messages
│   ├── helpers.go            # Formatting, filtering, pagination, log-var parsing
│   ├── styles.go             # Lip Gloss style definitions
│   ├── list.go               # Pipeline list screen
│   ├── detail.go             # Pipeline detail screen
│   ├── logs.go               # Step log viewer (search + auto-refresh)
│   ├── run.go                # Trigger pipeline form
│   ├── projects.go           # Project/favourite helpers
│   └── manage_projects.go    # Project manager screen
├── server/
│   └── server.go             # Webapp HTTP server (JSON API + embedded SPA)
├── webapp/                   # Svelte 5 single-page app
│   ├── src/                  # Components, stores, styles
│   └── package.json
├── test_webapp.sh            # Smoke-test script for the webapp HTTP API
├── bitbucket.openapi.json    # Bitbucket API 2.0 spec (reference only)
├── USERGUIDE.md              # Webapp run guide (Docker + bbc.sh)
├── WEBAPP_GUIDE.md           # Webapp frontend feature guide
└── README.md
```

## Nix flake

| Command           | Description                                      |
| ----------------- | ------------------------------------------------ |
| `nix develop`     | Enter dev shell with Go, Node, and golangci-lint |
| `nix build`       | Build release binary                             |
| `nix run .`       | Build and run the app (TUI mode by default)      |
| `nix flake check` | Validate the flake                               |

## Documentation

- [USERGUIDE.md](USERGUIDE.md) — how to run the app in webapp mode (Docker + `bbc.sh`, configuration, troubleshooting).
- [WEBAPP_GUIDE.md](WEBAPP_GUIDE.md) — complete description of the webapp frontend's features.

> Built out of spite — a frustration-fueled side project, developed with DeepSeek V4.