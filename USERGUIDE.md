# infra-pipeline-ui — User Guide (Webapp Mode)

This guide covers everything you need to run **infra-pipeline-ui** in **webapp
mode** (the browser UI) using the `bbc.sh` script. It is written so you can go
from "never touched this repo" to "looking at your pipelines in a browser" in a
few minutes.

> **Webapp mode** means the app runs a small embedded web server that serves a
> single-page frontend (Svelte) plus a JSON API. It is the same Go binary as the
> terminal UI (TUI) — the only difference is the `--webapp` flag, which `bbc.sh`
> passes for you automatically.

---

## Table of contents

1. [What is this?](#1-what-is-this)
2. [How it works (30-second overview)](#2-how-it-works-30-second-overview)
3. [Prerequisites](#3-prerequisites)
4. [Quick start (TL;DR)](#4-quick-start-tldr)
5. [First run, step by step](#5-first-run-step-by-step)
6. [The `bbc.sh` command reference](#6-the-bbcsh-command-reference)
7. [Configuration](#7-configuration)
8. [Using the webapp in your browser](#8-using-the-webapp-in-your-browser)
9. [HTTP API reference](#9-http-api-reference)
10. [Changing or resetting configuration](#10-changing-or-resetting-configuration)
11. [Troubleshooting](#11-troubleshooting)
12. [Windows (`bbc.bat`)](#12-windows-bbcbat)
13. [Security notes](#13-security-notes)

---

## 1. What is this?

`infra-pipeline-ui` is an internal tool for browsing, inspecting, and triggering
[Bitbucket Cloud](https://bitbucket.org) pipelines. It authenticates against the
Bitbucket REST API v2.0 using a **Bitbucket app password** over HTTP Basic Auth.

It ships two frontends backed by one Go binary:

| Mode       | How it runs                 | Selected by              |
| ---------- | --------------------------- | ------------------------ |
| **TUI**    | Terminal UI (Bubble Tea)    | default (no flags)       |
| **Webapp** | Browser SPA + JSON API      | `--webapp <host:port>`   |

This guide is about the **webapp** mode only, managed through `bbc.sh`.

---

## 2. How it works (30-second overview)

`bbc.sh` is a small wrapper around Docker that does three things:

1. **Builds a container image** using [Nix](https://nixos.org) as the build
   stage. The Nix build produces a single **static** Go binary with the webapp
   frontend embedded inside it. The runtime image is `scratch` — it contains
   *only* the binary and a CA certificate bundle, nothing else.
2. **Ensures a config file exists** (Bitbucket username, app password, and a list
   of projects). If `config.yml` is missing, it prompts you to run the built-in
   setup wizard or point at an existing file.
3. **Runs a container** that mounts your `config.yml` at `/config.yml` and
   publishes the webapp on a local port (default `8080`).

```
┌─────────────── host ───────────────┐        ┌────────── container ──────────┐
│  ./bbc.sh start                    │        │  scratch image                │
│    └─ docker run -p 8080:8080      │ ─────► │    /infra-pipeline-ui         │
│       -v ./config.yml:/config.yml  │        │      --webapp 0.0.0.0:8080    │
│                                    │        │                              │
│  browser → http://localhost:8080   │ ─────► │  Go server (Svelte SPA + /api)│
└────────────────────────────────────┘        └──────────────┬───────────────┘
                                                             │ HTTPS (Basic Auth)
                                                             ▼
                                               https://api.bitbucket.org/2.0
```

**Important:** you do **not** need Nix, Go, or Node.js installed on your host.
The Nix-based build happens entirely inside the Docker build stage.

---

## 3. Prerequisites

You need exactly two things:

### 3.1 Docker

A working Docker installation with the daemon running:

- **Linux** — Docker Engine (`docker`), or Podman with a `docker` alias.
- **macOS** — Docker Desktop.
- **Windows** — Docker Desktop (with the WSL 2 backend recommended for `bbc.sh`),
  or run `bbc.bat` (see [§12](#12-windows-bbcbat)).

Verify it is working:

```bash
docker run --rm hello-world
```

If this fails, start Docker Desktop / the Docker daemon first.

### 3.2 A Bitbucket app password

The app authenticates as *you* using a Bitbucket **app password** (not your
regular login password).

1. In Bitbucket Cloud, go to **Personal settings → App passwords → Create app
   password**.
2. Give it a label (e.g. `infra-pipeline-ui`).
3. Grant it at least the permissions the tool needs:
   - **Pipelines — Read** (to list pipelines, steps, and logs),
   - **Pipelines — Write** (to trigger and stop runs),
   - **Repositories — Read** (to list workspace repositories in the project manager).
4. Copy the generated password — it is shown only once.

You will also need the `workspace` and `repo_slug` of the repositories you want to
track (these are the URL segments in
`https://bitbucket.org/<workspace>/<repo_slug>`).

## 4. Quick start (TL;DR)

From the repository root:

```bash
# 1. Start it. On first run you'll be prompted to create the config.
./bbc.sh start

# 2. Check it is healthy.
./bbc.sh status

# 3. Open the webapp in your browser.
#    → http://localhost:8080

# 4. Stop it when you're done.
./bbc.sh stop
```

That's the entire normal workflow. The first `start` may take a few minutes
while Docker pulls the `nixos/nix` base image and Nix downloads build
dependencies — subsequent starts are fast because the image is cached.

---

## 5. First run, step by step

Here is exactly what happens the first time you run `./bbc.sh start`, and what
you'll see.

### 5.1 The image builds

```
[bbc] Image not found — building it now.
[bbc] Building image 'bbc-pipeline-ui' (Nix builds the Go binary + webapp)…
```

Docker pulls `nixos/nix`, copies the repository in, and runs `nix build` inside.
On a cold cache this downloads from `cache.nixos.org` and can take a few minutes.
You only pay this cost once (until the source changes).

### 5.2 The config prompt

If there is no `config.yml` next to the script, you are asked how to proceed:

```
[bbc] Config file not found: /path/to/config.yml

  1) Run the first-time setup wizard
  2) Set BBC_CONFIG_FILE to another path
  q) Abort

Choose an option [1/2/q]:
```

| Choice | What it does |
| ------ | ------------ |
| `1`    | Runs the built-in setup wizard (recommended for first use) |
| `2`    | Lets you type the path to an existing config file |
| `q`    | Aborts (nothing is started) |

### 5.3 The setup wizard (option 1)

Choosing `1` starts the interactive wizard inside a throwaway container. It asks
for, in order:

1. Your **Bitbucket username**,
2. Your **Bitbucket app password**,
3. One or more **`workspace` / `repo_slug`** pairs (press Enter on an empty
   workspace to finish).

```
Enter your Bitbucket username: your-username
Enter your Bitbucket app password: **************

Enter workspace slug (or press Enter to finish): my-workspace
Enter repository slug: my-repo
Added: my-workspace/my-repo

Enter workspace slug (or press Enter to finish):
```

The wizard writes the config to your host path (e.g. `./config.yml`) with
owner/permissions matching your user (`0600`).

> After the wizard finishes, the app briefly opens its **terminal UI** using the
> new config. Just press **`q`** to quit it — the script then proceeds to start
> the webapp container. (This is normal; it is a side effect of running the
> binary without the `--webapp` flag.)

### 5.4 The container starts

```
[bbc] Starting container 'bbc-pipeline-ui' (webapp on http://localhost:8080)…
[bbc] Started. Webapp: http://localhost:8080  (config: /path/to/config.yml)
```

You can now open **http://localhost:8080** in your browser.

## 6. The `bbc.sh` command reference

All commands are run from the repository root (the script resolves its own
directory, so you can also call it from anywhere via its full path).

| Command            | What it does |
| ------------------ | ------------ |
| `./bbc.sh build`   | Build (or rebuild) the Docker image. Does **not** run anything. |
| `./bbc.sh start`   | Ensure config exists (prompting if needed), build the image if missing, then run the container in webapp mode. Idempotent — safe to run repeatedly. |
| `./bbc.sh stop`    | Stop and remove the container. |
| `./bbc.sh restart` | `stop` then `start`. Use this after editing `config.yml`. |
| `./bbc.sh status`  | Show container state, ports, image, and (if running) a live health check. |
| `./bbc.sh logs`    | Follow (`-f`) the container's stdout logs. Ctrl+C to exit. |
| `./bbc.sh help`    | Print usage and the environment variable reference. |

Example session:

```bash
./bbc.sh start          # ensure running
./bbc.sh status         # confirm + health check
./bbc.sh logs           # tail logs
./bbc.sh restart        # pick up config changes
./bbc.sh stop           # shut down
```

### 6.1 Environment variables

`bbc.sh` is fully configurable through environment variables (set them inline,
in your shell, or in a `.env`-style wrapper):

| Variable            | Default          | Purpose |
| ------------------- | ---------------- | ------- |
| `BBC_IMAGE`         | `bbc-pipeline-ui` | Docker image name/tag to build and run. |
| `BBC_CONTAINER`     | `bbc-pipeline-ui` | Container name. |
| `BBC_CONFIG_FILE`   | `./config.yml`    | Host path to the config file mounted at `/config.yml`. |
| `BBC_PORT`          | `8080`            | Host port the webapp is published on. |
| `BBC_RESTART`       | `unless-stopped`  | Docker restart policy. |

Examples:

```bash
BBC_PORT=9090 ./bbc.sh start                    # serve on http://localhost:9090
BBC_CONFIG_FILE=/etc/bbc/config.yml ./bbc.sh start
BBC_IMAGE=bbc:dev BBC_CONTAINER=bbc-dev ./bbc.sh start
```

---

## 7. Configuration

### 7.1 File location and format

`bbc.sh` mounts a single YAML file into the container at `/config.yml`. By
default it looks for `config.yml` next to the script.

```yaml
username: "your-bitbucket-username"
app_password: "your-app-password"
projects:
  - workspace: "my-workspace"
    repo_slug: "my-repo"
  - workspace: "other-workspace"
    repo_slug: "other-repo"
```

Validation rules (the app refuses to start if these fail):

- `username` must be non-empty.
- `app_password` must be non-empty.
- At least **one** project is required.
- Every project needs both `workspace` and `repo_slug`.

Credentials are **global**, not per-project: one username/app-password is shared
across all listed projects.

### 7.2 Where else the config can live

- The container always reads `/config.yml` (the image sets `BBC_CONFIG=/config.yml`).
- On the host, point at a different file with `BBC_CONFIG_FILE`:
  ```bash
  BBC_CONFIG_FILE="$HOME/.config/bbc.yml" ./bbc.sh start
  ```
- Outside Docker, the binary honours the `BBC_CONFIG` environment variable and
  otherwise defaults to `~/.config/bbc-pipeline-tui.yml`.

### 7.3 Editing the config manually

You can skip the wizard entirely and create the file by hand:

```bash
cat > config.yml <<'EOF'
username: "your-bitbucket-username"
app_password: "your-app-password"
projects:
  - workspace: "my-workspace"
    repo_slug: "my-repo"
EOF
chmod 600 config.yml
./bbc.sh start
```

## 8. Using the webapp in your browser

Open **http://localhost:8080** (or your `BBC_PORT`). The webapp is a single-page
app with hash-based routing; the Go server serves the SPA and its `/api`
endpoints.

What you can do:

| Area | Capability |
| ---- | ---------- |
| **Pipeline list** | Browse pipelines for the active project; filter by build number, branch, type, or status. |
| **Pipeline detail** | Inspect a run's steps, variables, and metadata. |
| **Step logs** | Read a step's log output inline. |
| **Trigger** | Kick off a new run with a custom branch, selector, and runtime variables. |
| **Stop** | Stop an in-progress run. |
| **Running pipelines** | Overview of currently running pipelines across all configured projects. |
| **Projects** | Add/remove favourite `workspace/repo_slug` projects (stored in your config). |

The webapp never talks to Bitbucket directly and never stores your credentials in
the browser — the Go server proxies every Bitbucket call and holds the
credentials server-side.

### 8.1 Quick API sanity check

You can verify the server without a browser:

```bash
curl -s http://localhost:8080/api/health
# {"status":"ok"}
```

---

## 9. HTTP API reference

The webapp is driven by this JSON API (all routes under `/api`, scoped by project
**index** `{id}` — the 0-based position in the `projects` array):

| Method | Path | Purpose |
| ------ | ---- | ------- |
| GET    | `/api/health` | Health check → `{"status":"ok"}` |
| GET    | `/api/projects` | List configured projects |
| POST   | `/api/projects` | Add a project (`{workspace, repo_slug}`) |
| DELETE | `/api/projects/{id}` | Remove a project |
| GET    | `/api/projects/{id}/pipelines` | List pipelines (`?sort&pagelen&page&filter`) |
| GET    | `/api/projects/{id}/pipelines/{uuid}` | Pipeline detail (+ steps) |
| GET    | `/api/projects/{id}/pipeline-by-build` | Look up a pipeline by build number (`?build=`) |
| POST   | `/api/projects/{id}/pipelines` | Trigger a run (`{target, variables}`) |
| POST   | `/api/projects/{id}/pipelines/{uuid}/stop` | Stop a running pipeline |
| GET    | `/api/projects/{id}/pipelines/{uuid}/steps` | List steps |
| GET    | `/api/projects/{id}/pipelines/{uuid}/steps/{stepUuid}` | Step detail |
| GET    | `/api/projects/{id}/pipelines/{uuid}/steps/{stepUuid}/log` | Step log text |
| GET    | `/api/projects/{id}/pipelines/{uuid}/log-vars` | Variables parsed from the first step's log |
| GET    | `/api/projects/{id}/variables` | Repo-level pipeline config variables |
| GET    | `/api/repositories/{workspace}` | List workspace repositories (paginated) |
| GET    | `/api/running-pipelines` | Running pipelines across all projects |

> **UUID braces note:** Bitbucket returns pipeline/step UUIDs wrapped in `{...}`.
> When calling this API directly, strip the braces (e.g. `{abc}` → `abc`).

## 10. Changing or resetting configuration

| Want to… | Do this |
| -------- | ------- |
| Add/remove a project or change credentials | Edit `config.yml`, then `./bbc.sh restart` (or use the webapp's Projects UI). |
| Re-run the setup wizard from scratch | Delete the config and start again: `rm config.yml && ./bbc.sh start` |
| Use a different config file | `BBC_CONFIG_FILE=/path ./bbc.sh restart` |
| Change the port | `BBC_PORT=9090 ./bbc.sh restart` |

---

## 11. Troubleshooting

| Symptom | Likely cause / fix |
| ------- | ------------------ |
| `docker: command not found` or "Cannot connect to the Docker daemon" | Docker isn't installed or the daemon isn't running. Start Docker Desktop / the daemon and retry. |
| First build is slow (minutes) | Normal on a cold cache — Nix downloads the toolchain and dependencies from `cache.nixos.org`. |
| Port already in use: `bind: address already in use` | Something else is on `8080`. Use `BBC_PORT=8081 ./bbc.sh start`. |
| `config file not found` prompt loops forever | The path you entered for option `2` doesn't exist. Enter a path to a real YAML file, or choose `1` to create it. |
| Webapp loads but pipelines show an API error / `401` | Wrong username or app password, or the app password lacks the required scopes. Fix `config.yml` and `./bbc.sh restart`. |
| `bbc.sh status` prints `(not created)` but you expected it running | The container was stopped/removed. Run `./bbc.sh start`. |
| `curl: (7) Failed to connect` in `status` | Container isn't running or hasn't finished starting. Check `./bbc.sh logs`. |
| Container keeps restarting | Invalid `config.yml` (e.g. empty fields). Run `./bbc.sh logs` to see the validation error, fix the file, then `./bbc.sh restart`. |
| Config file written but owned by `root` (permission denied) | Fixed in current builds via `--user`. Rebuild with `./bbc.sh build` to pick up the latest script/Dockerfile behaviour. |
| `docker: invalid reference format` | `BBC_IMAGE`/`BBC_CONTAINER` contains invalid characters. |
| `NAME conflict` — container name already in use | Another container uses that name. Use `BBC_CONTAINER=other-name`, or `./bbc.sh stop` first. |

### 11.1 Getting help

```bash
./bbc.sh help                      # usage + env vars
docker logs bbc-pipeline-ui        # raw container logs
```

---

## 12. Windows (`bbc.bat`)

A Windows equivalent is provided: `bbc.bat`. It supports the same commands
(`build`, `start`, `stop`, `restart`, `status`, `logs`, `help`) and the same
environment variables (`BBC_IMAGE`, `BBC_CONTAINER`, `BBC_CONFIG_FILE`,
`BBC_PORT`).

```bat
bbc.bat start
bbc.bat status
bbc.bat stop
```

> Prefer `bbc.sh` under WSL 2 (Docker Desktop's WSL backend) if you have it
> available — the wizard's interactive TTY flow is more reliable there.

---

## 13. Security notes

- `config.yml` contains your Bitbucket **app password** in plain text. It is
  written with `0600` permissions and is **git-ignored** — do not commit it or
  share it.
- The container runs with `--restart unless-stopped` by default; stop it with
  `./bbc.sh stop` when not in use.
- The webapp binds to all interfaces inside the container (`0.0.0.0:8080`) but is
  published only to the host port you choose (`localhost` by default). Don't
  publish it to a public interface (`BBC_PORT` maps host-side only, so
  `localhost:PORT` is the normal access path).
- Rotate/revoke the app password in Bitbucket if you ever suspect it leaked; a
  scoped app password is far safer than your main account password.

---

## Glossary

| Term | Meaning |
| ---- | ------- |
| **TUI** | Terminal user interface (the default mode). |
| **Webapp** | Browser single-page app + JSON API (`--webapp`). |
| **App password** | Bitbucket's scoped credential used instead of your login password. |
| **Workspace** | Bitbucket account/team slug (first URL segment). |
| **`repo_slug`** | Repository name slug (second URL segment). |
| **`{id}`** | 0-based index of a project in the `projects` config list. |




