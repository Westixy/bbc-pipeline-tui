# infra-pipeline-ui — Repository Context

> Note: this repo already contains a lower-case `llm.md`. That file is **stale/out of date**
> (it describes a Gin backend, `-serve`/`-port` flags, per-project credentials, and a
> `~/.config/bbc-pipeline-tui/config.yml` path — none of which match the current code).
> Trust this `LLM.md` and the source code over `llm.md`.

## 1. Project overview

`infra-pipeline-ui` is an internal tool for browsing, inspecting, and triggering
[Bitbucket Cloud](https://bitbucket.org) pipelines. It ships **two frontends** backed by a
single Go codebase:

- A **terminal UI (TUI)** built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)
  + [Lip Gloss](https://github.com/charmbracelet/lipgloss). This is the default mode.
- A **web app** — a [Svelte 5](https://svelte.dev) SPA served by an embedded Go HTTP server.
  This mode is selected with the `--webapp` CLI flag.

Both frontends proxy the Bitbucket Cloud REST API v2.0 (`https://api.bitbucket.org/2.0`)
through the same `bitbucket` Go client, authenticating with a Bitbucket **app password** over
HTTP Basic Auth. Users list pipelines, filter them, view pipeline/step details, read step logs,
trigger new runs (with custom branch/selector/variables), stop in-progress runs, and manage
favourite `workspace/repo_slug` projects. Configuration is stored in a local YAML file.

- **Language / toolchain**: Go (`go.mod` declares `go 1.24`); module path
  `github.com/bbc/infra-pipeline-ui`.
- **Go deps**: `github.com/charmbracelet/bubbletea v1.3.3`, `bubbles v0.20.0`,
  `lipgloss v1.1.0`, `gopkg.in/yaml.v3 v3.0.1`.
- **Frontend deps** (`webapp/package.json`): `svelte ^5.0.0`, `vite ^6.0.0`,
  `@sveltejs/vite-plugin-svelte ^5.0.0`.
- **Build/packaging**: Nix (`flake.nix`), targeting `nixpkgs-unstable`.

## 2. Repository structure

```
.
├── main.go                    # Entry point: loads config, chooses TUI vs webapp mode
├── go.mod / go.sum            # Go module (github.com/bbc/infra-pipeline-ui, go 1.24)
├── flake.nix / flake.lock     # Nix flake: dev shell, package (Go binary + embedded webapp)
├── README.md                  # User-facing docs (NOTE: some details are out of date — see §5)
├── bitbucket.openapi.json     # Bitbucket API 2.0 OpenAPI spec (reference only, not consumed by code)
├── test_webapp.sh             # Smoke-test script for the webapp HTTP API
├── bbc-pipeline-ui            # Prebuilt binary (committed; 13 MB)
│
├── bitbucket/                 # Bitbucket API client library (the only thing that talks to Bitbucket)
│   ├── client.go              # HTTP client, Basic auth, shared types, pagination types
│   ├── pipelines.go           # Pipeline/step/log/variable API methods
│   ├── repositories.go        # Workspace repository listing
│   ├── cached_client.go       # CachedClient: transparent in-memory TTL cache wrapper
│   └── cache.go               # Thread-safe TTL cache implementation
│
├── config/                    # Configuration handling
│   ├── config.go              # YAML load/save, Project type, validation, project helpers
│   └── wizard.go              # Interactive first-run setup wizard (stdin-based)
│
├── ui/                        # Bubble Tea TUI
│   ├── types.go               # Model struct, screen/state constants
│   ├── update.go              # NewModel, Init, Update router, global key handling
│   ├── view.go                # View router, header/help/content layout, shared render helpers
│   ├── commands.go            # Async tea.Cmd functions + result message types
│   ├── helpers.go             # Formatting, filtering, pagination, log-var parsing, status resolution
│   ├── styles.go              # Lip Gloss style definitions
│   ├── list.go                # Pipeline list screen (split-panel preview)
│   ├── detail.go              # Pipeline detail screen (steps, variables)
│   ├── logs.go                # Step log viewer (search + auto-refresh)
│   ├── run.go                 # Trigger-pipeline form
│   ├── projects.go            # Project selection screen
│   └── manage_projects.go     # Two-pane project manager (favourites + workspace browser)
│
├── server/                    # HTTP API server (backend for the Svelte webapp)
│   ├── server.go              # net/http ServeMux routes, CORS, handlers, embedded dist
│   └── webapp-dist/           # Built frontend (vite output); embedded via go:embed; gitignored
│
└── webapp/                    # Svelte 5 frontend
    ├── package.json
    ├── svelte.config.js       # vitePreprocess only (no adapter; plain SPA)
    ├── vite.config.js         # outDir: ../server/webapp-dist; dev proxy /api → :8080
    ├── index.html             # Mounts to <div id="app">
    └── src/
        ├── main.js            # Svelte mount point
        ├── App.svelte         # Root component: loads projects, renders route-based screens
        ├── app.css            # Global CSS (dark theme, custom properties)
        ├── stores/
        │   ├── api.js         # All fetch() calls to the Go backend
        │   ├── appState.js    # Svelte writable/derived stores (global state)
        │   └── router.js      # Hash-based router (#/bbc/<workspace>/<repo>/...)
        └── lib/               # UI components
            ├── Navbar.svelte
            ├── PipelineList.svelte
            ├── PipelineDetail.svelte
            ├── PipelineLog.svelte
            ├── PipelineTrigger.svelte
            ├── ManageProjects.svelte
            ├── ConfirmModal.svelte
            ├── Notification.svelte
            ├── keyboard.js
            └── utils.js       # formatDate/formatDuration/statusLabel/resolveStepStatus helpers
```

## 3. Architecture

### 3.1 High-level flow

```
                    ┌─────────────────────────────────────────────┐
  TUI (Bubble Tea)  │  ui/  ──► bitbucket.CachedClient ──┐         │
  (default)         │                                     │         │
                    │                                     ▼         │   Basic Auth
                    │                            bitbucket.Client ──┼──► api.bitbucket.org/2.0
  Webapp (Svelte 5) │  webapp/ ──► server/ (net/http) ────▲         │
  (--webapp flag)   │                 │                   │         │
                    └─────────────────┼───────────────────┼─────────┘
                                      └─ config.Config ───┘
```

- **`bitbucket.Client`** is the only component that performs HTTP against Bitbucket. It sets the
  `Authorization: Basic base64(user:pass)` header and a `User-Agent: bbc-pipeline-tui/1.0`
  header on every request, and decodes JSON responses into typed structs (defined in
  `bitbucket/client.go`).
- **`bitbucket.CachedClient`** wraps `Client` with an in-memory, thread-safe TTL cache
  (`bitbucket/cache.go`). Read methods are cached; **mutating methods (Trigger/Stop) pass through
  and are never cached**. Step logs are never cached. Default TTL is 30s; step detail uses 5s.
- **`server`** is a plain `net/http` `ServeMux` (no framework) using Go 1.22+ method-and-path
  route patterns (e.g. `GET /api/projects/{id}/pipelines`). It embeds the built Svelte bundle
  via `//go:embed all:webapp-dist` and serves it with an SPA fallback to `index.html`.
- **`ui`** is a standard Bubble Tea `Model` with a screen-based `Update` router
  (`ui/update.go` dispatches to `updateList`/`updateDetail`/`updateLogs`/`updateRun`/
  `updateProjects`/`updateManageProjects`). Async work is done via `tea.Cmd` functions in
  `ui/commands.go` that return typed `*Msg` structs consumed by the update handlers.
- **`webapp`** is a Svelte 5 SPA (rune-based reactivity: `$state`, `$effect`, `$derived`). It has
  no client-side copy of credentials — it talks only to the local Go server's `/api` endpoints,
  which look up credentials server-side.

### 3.2 Config model

`config.Config` (in `config/config.go`) holds **global** credentials plus a project list:

```yaml
username: "your-bitbucket-username"
app_password: "your-app-password"
projects:
  - workspace: "my-workspace"
    repo_slug: "my-repo"
  - workspace: "other-workspace"
    repo_slug: "other-repo"
```

- Default path is `~/.config/bbc-pipeline-tui.yml` (see `config.DefaultPath()`).
- Credentials are **not** per-project — one username/app-password is shared across all projects.
- Projects are identified **by array index** everywhere (`ActiveProject` in the TUI, `{id}` path
  segments in the HTTP API). Adding/removing projects mutates the slice and re-saves the file.

### 3.3 HTTP API surface (Go server → webapp)

All routes are under `/api` and scoped by project **index** (`{id}`):

| Method | Path | Purpose |
| ------ | ---- | ------- |
| GET    | `/api/health` | Health check (`{"status":"ok"}`) |
| GET    | `/api/projects` | List projects (`{projects:[{id, name, workspace, repo_slug}]}`) |
| POST   | `/api/projects` | Add a project (`{workspace, repo_slug}` body) |
| DELETE | `/api/projects/{id}` | Remove a project |
| GET    | `/api/projects/{id}/pipelines` | List pipelines (`?sort&pagelen&page&filter`) |
| GET    | `/api/projects/{id}/pipelines/{uuid}` | Pipeline detail `{pipeline, steps}` |
| POST   | `/api/projects/{id}/pipelines` | Trigger a run (`{target, variables}`) |
| POST   | `/api/projects/{id}/pipelines/{uuid}/stop` | Stop a running pipeline |
| GET    | `/api/projects/{id}/pipelines/{uuid}/steps` | List steps |
| GET    | `/api/projects/{id}/pipelines/{uuid}/steps/{stepUuid}` | Step detail |
| GET    | `/api/projects/{id}/pipelines/{uuid}/steps/{stepUuid}/log` | Step log text `{log, stepUuid}` |
| GET    | `/api/projects/{id}/pipelines/{uuid}/log-vars` | Variables parsed from first step's log |
| GET    | `/api/projects/{id}/variables` | Repo-level pipeline config variables |
| GET    | `/api/repositories/{workspace}` | List workspace repositories (paginated) |

### 3.4 Patterns

- **Single shared client**: one `bitbucket.Client` is created in `main.go` from the global
  credentials and shared by both modes; the TUI additionally wraps it in a `CachedClient`.
- **Proxy/anti-corruption layer**: the Svelte app never calls Bitbucket directly; the Go server
  owns all credentials and Bitbucket knowledge.
- **Elm/Tea architecture** for the TUI: pure `Update(msg) → (Model, Cmd)`, `View(model) → string`,
  async effects as `Cmd`s.
- **Hash-based routing** for the SPA (`webapp/src/stores/router.js`), driven by
  `window.location.hash` and `hashchange` listeners.

## 4. Development workflows

### 4.1 Build & run

**Nix (recommended, builds frontend + backend together):**

```bash
nix run .          # build + run the TUI
nix build .        # build release binary → ./result/bin/infra-pipeline-ui
nix develop        # enter dev shell (adds go, golangci-lint, gnumake, nodejs to PATH)
nix flake check    # validate the flake
```

**Go (manual):**

```bash
cd webapp && npm install && npm run build   # build frontend → server/webapp-dist (REQUIRED first)
go build -o infra-pipeline-ui .             # build the Go binary
go run .                                    # run the TUI
go run . --webapp localhost:8080            # run the webapp HTTP server
```

> ⚠️ The frontend **must** be built before `go build`/`go run` on a fresh checkout: the
> `//go:embed all:webapp-dist` directive in `server/server.go` is a compile-time check, and
> `server/webapp-dist/` is gitignored. `nix build` handles this automatically via its
> `preBuild` hook; manual Go builds do not.

**Webapp dev server (with HMR):**

```bash
# terminal 1 — Go backend
go run . --webapp localhost:8080
# terminal 2 — Vite dev server on :5173, proxies /api → http://localhost:8080
cd webapp && npm run dev
```

### 4.2 Test & lint

```bash
go test ./...          # unit tests — note: the repo currently contains NO *_test.go files
golangci-lint run      # lint (available in nix develop)
./test_webapp.sh       # end-to-end smoke test of the webapp HTTP API
```

`test_webapp.sh` runs the prebuilt `./bbc-pipeline-ui --webapp localhost:18090`, then curls
`/api/health`, `/api/projects/0/pipelines`, pipeline detail, variables, and the frontend root.

### 4.3 CI/CD

There is **no CI/CD configuration** in this repository — no `.github/workflows`, no
`bitbucket-pipelines.yml`, and no `Makefile` (despite the README referencing one). Build/lint/test
are run locally or via `nix flake check`. The `flake.nix` packages are the only automated build
definitions.

## 5. Conventions and gotchas

### 5.1 UUID braces — the most important gotcha

Bitbucket returns pipeline and step UUIDs **wrapped in curly braces**, e.g.
`{a29195fa-6f0d-...}`.

- **TUI path**: UUIDs are taken directly from Bitbucket API responses and passed back verbatim,
  so braces are preserved and no special handling is needed.
- **Webapp/server path**: Go's `http.ServeMux` strips `{` and `}` from `{uuid}`/`{stepUuid}`
  path wildcards. So:
  - The webapp client **strips** braces before building URLs (`cleanUuid()` in
    `webapp/src/stores/api.js`).
  - The server **re-adds** them with `ensureBraces()` (`server/server.go`) before calling the
    Bitbucket client.

Do not "fix" this by removing one side — the pair (strip on client, re-wrap on server) must stay
in sync.

### 5.2 Status resolution

Pipeline/step state is a two-level structure: `state.name` (`IN_PROGRESS`, `PENDING`,
`COMPLETED`, …) plus an optional `state.result.name` (`SUCCESSFUL`, `FAILED`, …) when completed.
There is no flat `SUCCESSFUL`/`FAILED` top-level name. Canonical resolution lives in
`ui/helpers.go` (`resolvePipelineResult`/`resolveStepStatus`) and is mirrored in
`webapp/src/lib/utils.js` (`resolveStepStatus`/`statusLabel`). Keep both in sync if you change it.

### 5.3 Log-variable parsing

Pipeline variables are not always returned by the API (especially for custom pipelines). Both
backends parse a `Pipeline variables:` block out of the first step's log:

- TUI: `ui/helpers.go` → `ParsePipelineVariablesFromLog`
- Server: `server/server.go` → `parsePipelineVariablesFromLog`

The regex (`(?m)^Pipeline variables:\n(...)`) is duplicated in both files; update both if it
changes.

### 5.4 Pagination sort quirk

Bitbucket's `next` pagination URL may drop the `sort` query parameter, causing subsequent pages
to return oldest-first. `bitbucket/pipelines.go` (`ListPipelinesNext`) and
`ui/commands.go` (`fetchNextPage`) both explicitly re-append `sort=-created_on` when missing.

### 5.5 Caching semantics

- `CachedClient` is only used by the **TUI**. The webapp server uses the raw `Client` (no cache).
- Cache keys are the full request path+query. Entries larger than 256 KB are dropped (`cache.go`).
- Mutations (`TriggerPipeline`, `StopPipeline`) bypass the cache, as do step logs.

### 5.6 Naming / file organization

- Go packages are `main`, `bitbucket`, `config`, `ui`, `server` — one concept per directory.
- TUI files split `update`/`view`/`commands`/`helpers`/`styles` per concern, with one file per
  screen (`list.go`, `detail.go`, `logs.go`, `run.go`, `projects.go`, `manage_projects.go`).
- Svelte components are PascalCase files in `webapp/src/lib/`; shared fetch logic lives only in
  `webapp/src/stores/api.js`; shared formatting lives only in `webapp/src/lib/utils.js`.
- Svelte 5 runes are used throughout (`$state`, `$effect`, `$derived`) — do not introduce legacy
  Svelte 4 `onMount`/`let`-reactivity patterns.

### 5.7 Known discrepancies / stale docs (do not trust)

- `README.md` says the config path is `~/.config/infra-pipeline-ui.yml`; the actual code
  (`config/config.go`) uses `~/.config/bbc-pipeline-tui.yml`.
- `README.md` lists a `Makefile` in the project structure; **no Makefile exists**.
- `llm.md` (lowercase) describes a Gin backend, `-serve`/`-port` flags, per-project credentials,
  and `~/.config/bbc-pipeline-tui/config.yml` — all incorrect for the current code. The flag is
  `--webapp`, the backend is `net/http`, credentials are global.
- `config/wizard.go` has an `init()` that references `strconv.Itoa` solely to avoid an unused
  import; leave it alone unless you also remove the import.

## 6. Where to look for what

| Task | Relevant files |
| ---- | -------------- |
| Add a Bitbucket API call | `bitbucket/client.go` (types) + `bitbucket/pipelines.go`/`repositories.go` (methods); add a cached wrapper in `bitbucket/cached_client.go` if it's a read |
| Add a webapp HTTP endpoint | `server/server.go` (route + handler) + `webapp/src/stores/api.js` (fetch fn) |
| Add a webapp screen | New component in `webapp/src/lib/`, add route case in `webapp/src/stores/router.js` + `webapp/src/App.svelte`, add state in `webapp/src/stores/appState.js` |
| Add a TUI screen | New `ui/<screen>.go` with `update<Screen>`/`view<Screen>`, register in `ui/update.go` and `ui/view.go` routers, add screen constant in `ui/types.go` |
| Change config fields | `config/config.go` (struct + Validate) and `config/wizard.go` (first-run); both TUI and server read from `config.Config` |
| Change styling (TUI) | `ui/styles.go` |
| Change styling (webapp) | `webapp/src/app.css` (global CSS custom properties) |
| Change trigger-pipeline request shape | `bitbucket/client.go` (`TriggerPipelineRequest`) + `webapp/src/stores/api.js` (`triggerPipeline`) + `ui/run.go` (TUI form) |
| Fix pipeline/step status display | `ui/helpers.go` + `webapp/src/lib/utils.js` (keep in sync) |
| Fix log-variable parsing | `ui/helpers.go` + `server/server.go` (duplicated regex) |
| Verify the webapp end-to-end | `test_webapp.sh` |
