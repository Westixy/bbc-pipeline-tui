# infra-pipeline-ui — Webapp Frontend Guide

This guide describes **all the functionality of the webapp frontend** — the
browser UI served by the Go backend. It assumes the app is already running (see
[`USERGUIDE.md`](USERGUIDE.md) for how to start it) and focuses on *what every
screen does and how to use it*.

> **How to run:** `./bbc.sh start`, then open <http://localhost:8080>.

---

## Table of contents

1. [Overview](#1-overview)
2. [Layout & navigation](#2-layout--navigation)
3. [Routes (URL structure)](#3-routes-url-structure)
4. [Pipeline List](#4-pipeline-list)
5. [Pipeline Detail](#5-pipeline-detail)
6. [Step Logs](#6-step-logs)
7. [Trigger Pipeline](#7-trigger-pipeline)
8. [Running Pipelines](#8-running-pipelines)
9. [Manage Projects](#9-manage-projects)
10. [Notifications & confirmation modals](#10-notifications--confirmation-modals)
11. [Click modifiers & keyboard shortcuts](#11-click-modifiers--keyboard-shortcuts)
12. [Status & state resolution](#12-status--state-resolution)

---

## 1. Overview

The frontend is a **single-page application** (Svelte 5) with hash-based routing.
It never talks to Bitbucket directly and never holds your credentials — every
`/api` call goes to the Go server, which proxies Bitbucket and keeps the app
password server-side.

The UI is organised like a desktop IDE:

- A **title bar** (top) with a breadcrumb showing the current screen.
- A **sidebar** (left) with an icon *activity bar* and a *project list*.
- A **content area** (centre) that changes per screen.
- A **status bar** (bottom) with the project count.

There are six screens:

| Screen | Purpose | Route |
| ------ | ------- | ----- |
| **Pipeline List** | Browse a project's pipeline runs | `#/bbc/<workspace>/<repoSlug>` |
| **Pipeline Detail** | Inspect one run's steps & variables | `#/bbc/<ws>/<repo>/<buildNumber>` |
| **Step Logs** | Read a step's live log output | `#/bbc/<ws>/<repo>/<build>/logs/<stepNum>` |
| **Trigger Pipeline** | Start a new run | `#/bbc/<ws>/<repo>/trigger` |
| **Running Pipelines** | All running runs across every project | `#/running` |
| **Manage Projects** | Add / remove favourite repositories | `#/manage` |

## 2. Layout & navigation

### 2.1 Title bar

The top bar shows, from left to right:

- The **app name** ("BBC Pipeline Manager").
- A **breadcrumb** in the centre that reflects the current screen
  (e.g. `Pipelines`, `Pipeline Detail`, `Pipeline Logs`, `Trigger Pipeline`,
  `Manage Projects`, `Running Pipelines`).
- The **version** badge (`v1.0`).

### 2.2 Activity bar (icon rail)

The narrow leftmost rail has five vertically stacked icons:

| Icon | Action |
| ---- | ------ |
| **Pipelines** (list) | Go to the pipeline list for the active project. Disabled until at least one project is configured. |
| **Running Pipelines** (activity pulse) | Go to the cross-project running-pipelines screen. |
| **Trigger Pipeline** (play) | Go to the trigger form for the active project. Disabled with no active project. |
| **Manage Projects** (folder) | Go to the project manager. |
| **Refresh** (arrows, at the bottom) | Re-fetch the current screen. Shown only when a project is active. |

The icon for the current screen is highlighted with a blue accent bar.

### 2.3 Sidebar panel (project list)

Next to the icon rail is a panel listing every configured project:

- Each row shows the project **name** (e.g. `my-workspace/my-repo`) and, under it,
  the `workspace/repo_slug`.
- A **✓** marks the currently active project.
- **Click** a project to make it active and open its pipeline list.
- The **Settings** button at the bottom opens the Manage Projects screen.

### 2.4 Status bar

The bottom bar shows the **number of configured projects** and the copyright
footer.

### 2.5 Landing page

If you open the app with no route in the URL, you see a landing card with two
buttons — **View Pipelines** and **Manage Projects** — which bootstrap the
default route. (When projects are loading, a spinner is shown instead.)

## 3. Routes (URL structure)

Navigation is **hash-based**, so screens are shareable/bookmarkable URLs:

| Route | Screen |
| ----- | ------ |
| `#/manage` | Manage Projects |
| `#/running` | Running Pipelines (all projects) |
| `#/bbc/<workspace>/<repoSlug>` | Pipeline List |
| `#/bbc/<workspace>/<repoSlug>/<buildNumber>` | Pipeline Detail |
| `#/bbc/<workspace>/<repoSlug>/<buildNumber>/logs/<stepNum>` | Step Logs |
| `#/bbc/<workspace>/<repoSlug>/trigger` | Trigger Pipeline |

Deep links survive a full page reload: the app re-matches the URL's
`workspace/repoSlug` to the active project and re-fetches the pipeline by its
build number.

---

## 4. Pipeline List

The default screen. It shows the pipeline runs for the **active project** in a
table, sorted newest-first by default.

### 4.1 Toolbar

- **Title** — the active project name.
- **Count** — `N pipelines` (a `+` suffix indicates more pages are available).
  While filtering, it shows `X of N`.
- **Filter box** — filters the *currently loaded* list **client-side** (it does
  not hit the API). It matches across many fields:
  build number, target branch, commit hash, status, creator (name or username),
  trigger name, pipeline selector pattern, and created date.
- **Sort dropdown** — `Newest first` (`-created_on`, default) or `Oldest first`
  (`+created_on`).

### 4.2 Table columns

| Column | Shows |
| ------ | ----- |
| `#` | Build number. |
| `TARGET` | Branch name plus a short (7-char) commit hash. |
| `PIPELINE` | Pipeline selector pattern (`default` when none). |
| `VIA` | Trigger type: `push`, `manual`, `schedule`, `PR`, or the raw name. |
| `STATUS` | Colour-coded badge (running/pending pulse, succeeded/failed/stopped). |
| `DUR` | Wall-clock duration (or `—`). |
| `CREATOR` | Display name or username. |
| `CREATED` | Creation timestamp. |
| (last) | A **Log** button on running/pending rows. |

### 4.3 Infinite scroll

The list requests **50 pipelines per page** and loads more automatically as you
scroll (via a sentinel row). The sentinel shows `Scroll for more`, a spinner
while loading, or `All pipelines loaded` when there are no more pages.

### 4.4 Auto-refresh

While you stay on the list page, the app polls every **15 seconds** and merges
in state changes (updating running→finished statuses and prepending new
pipelines). This is seamless — you don't need to click Refresh.

### 4.5 Clicking a row

- **Plain click** → opens Pipeline Detail for that run.
- **Ctrl / Cmd / Shift + click** (or **middle-click**) → opens Detail in a **new
  browser tab**.
- **Ctrl / Cmd + Alt + click** → opens the run on **bitbucket.org** in a new tab.

### 4.6 Keyboard navigation

A row can be focused and navigated with:

- **↑ / ↓** — move selection up/down (and scroll it into view).
- **Enter** or **Space** — open the selected pipeline's detail.

### 4.7 Hover: log variables

Hovering a row (after a short delay) shows a **popover with that pipeline's
runtime variables** (key/value pairs, parsed from the first step's log).
Secured variables are masked as `••••••••`. Results are cached per pipeline for
the session.

### 4.8 Quick "Log" jump

For a running or pending pipeline, the last column shows a **Log** button that
jumps straight to the **currently running step's log** (resolving the right step
for you).

### 4.9 States

- **Loading** — a skeleton grid while the first page loads.
- **Error** — a message plus a **Retry** button.
- **Empty** — "No pipelines" when the repository has no runs.
- **Filter empty** — "No matching pipelines" plus a **Clear filter** button.
- **Scroll-to-top** — a floating button appears after scrolling down.

## 5. Pipeline Detail

Opened by clicking a pipeline. It shows everything about a single run.

### 5.1 Sticky header

- **Back** (`Pipelines`) — return to the list.
- **Build number** (`#123`) and a colour-coded **status badge**.
- A **result tag** (e.g. `FAILED`) when it adds info beyond the badge.
- **Actions** (right side):
  - **Stop** — shown only while the pipeline is running; stops it (with a
    confirmation modal).
  - **Re-run** — opens the Trigger form pre-filled with this pipeline's target
    branch, selector, and variables.
  - **Refresh** — re-fetch details immediately.
  - **Bitbucket** — open the run on bitbucket.org in a new tab.
- A **Live** indicator and an elapsed counter while the pipeline is running
  (auto-refreshes every **5 s** until it finishes).

### 5.2 Metadata strip

A row of fields: **Branch**, **Pattern** (pipeline selector), **Trigger**,
**Creator**, **Duration**, **Created**, **Completed** (when finished), and
**Commit** — the first 8 chars of the hash with a **copy** button for the full
hash.

### 5.3 Pipeline progress timeline

A visual timeline of the pipeline's steps:

- **Summary chips** count the steps by state: Done, Failed, Skipped, Running,
  Pending, Stopped.
- A `finished/total` **percentage** indicator with a ✓ / ✕ / ■ glyph.
- **Step nodes** coloured by status, connected by lines. Steps after a failed or
  stopped step are shown as **skipped**.
- Clicking a step node opens that step's log (only for steps that have started).

### 5.4 Steps table

Below the timeline, a table lists each step with:

- A status **dot**,
- The step **name**,
- A status **badge** (including `skipped` for steps after a break),
- **Duration** — wall (`⏱`), run (`▶`) and build (`⚙`) durations when available,
- A **Log** button (for started steps) that opens the Step Logs screen.

### 5.5 Variables panel

On the right, the **Variables** panel shows the pipeline's variables with a
segmented **Log / Pipeline** tab switcher:

- **Log** — variables parsed from the `Pipeline variables:` block in the first
  step's log.
- **Pipeline** — repository-level pipeline configuration variables.

Features:

- A **search box** (shown when there are more than a handful of variables).
- **Secured** variables are masked (`••••••••••••••••`) and tagged `SECURED`.
- A **"Show all N variables…" / "Show fewer"** toggle limits the list to the
  first 6 entries.

### 5.6 States

- **Loading** spinner, **Error** with Retry, and an empty state when there are
  no variables.

## 6. Step Logs

The log viewer for a single step. Reached by clicking a step node, a step row's
**Log** button, or a list row's quick **Log** button.

### 6.1 Header

- **Back** — returns to Pipeline Detail.
- **Step selector** — a dropdown listing every step (with status dots/badges) to
  jump between steps.
- **Step status badge** for the current step.
- **Live** indicator + an **elapsed** counter while the step is running.
- **Actions**:
  - **Stop** / **Re-run** (same semantics as Detail),
  - **Toggle word wrap**,
  - **Refresh**,
  - **Copy full log**,
  - **Bitbucket** (open the run on bitbucket.org),
  - **Prev / Next step** (`n/N` of total),
  - **Scroll to bottom**.

### 6.2 Info strip & step progress bar

Below the header, a compact info strip repeats Branch, Pattern, Trigger, Creator,
Duration, Created, Completed and Commit (with copy). A horizontal **step progress
bar** shows all steps as clickable segments, colouring done/failed/active steps.

### 6.3 Variables section

A collapsible section shows the pipeline's variables (same Log / Pipeline tabs,
search, and secured-masking as the Detail screen).

### 6.4 Log viewer

- **Line numbers** on the left.
- **ANSI colour rendering** — the log's escape codes (colours, bold, dim, etc.)
  are converted to styled HTML, so coloured build output renders correctly.
- **Word wrap** toggle (on by default; off shows horizontal scroll).
- **Auto-scroll** — sticks to the bottom while new lines arrive; scrolling up
  pauses auto-refresh, scrolling back down resumes it.
- A **bottom bar** shows the line count, auto-scroll state, and last update time.

### 6.5 Search

- Press **`/`** to focus the search box (or click it).
- As you type, matching lines are highlighted and the box shows `current/total`
  matches (or `no matches`).
- **`n`** / **`N`** jump to the next/previous match (or use the ▲/▼ buttons).
- **`Esc`** clears the search.

### 6.6 Live behaviour

While the current step is running, the viewer **auto-refreshes every 5 s** and
**auto-advances**: when the step finishes, it detects the next running/pending
step, loads its log, and updates the URL — so you can watch a whole pipeline
progress without touching the UI. It stops after 3 consecutive fetch errors.

### 6.7 Keyboard shortcuts (log screen)

| Key | Action |
| --- | ------ |
| `/` | Focus search |
| `n` / `N` | Next / previous search match |
| `Esc` | Clear search (or close step dropdown, or go back to Detail) |

## 7. Trigger Pipeline

Starts a new pipeline run for the active project. Reached from the activity bar,
or via **Re-run** on Detail/Logs (which pre-fills the form).

### 7.1 Form fields

- **Target Branch** *(required)* — the git branch to run against
  (e.g. `main`, `develop`, `feature/xyz`).
- **Pipeline Selector** *(optional)* — a `custom` pattern to filter which steps
  execute (e.g. `default` or a custom pattern).
- **Variables** *(optional)* — a dynamic list of `key` / `value` pairs:
  - **"+ Add Variable"** appends a row,
  - **✕** removes a row,
  - empty keys are dropped on submit.

### 7.2 Pre-filled ("Run again")

When you click **Re-run** on a pipeline, the form shows a
**"Pre-filled from previous pipeline"** badge and is populated with that run's
target branch, selector, and variables (from its parsed log variables).

### 7.3 Submitting

Click **▶ Run Pipeline**. On success:

- A success notification shows the new build number,
- The pipeline list refreshes,
- You're taken back to the list.

On failure (e.g. missing branch, API error) an error notification is shown and
the form stays open.

## 8. Running Pipelines

A cross-project dashboard showing **every currently running pipeline** across
*all* configured projects in one table.

### 8.1 Toolbar

- **Title** and a **count** — `N running pipelines across M repos`.
- **Repository filter** — `All repositories` or a specific project.
- **Auto-refresh interval** selector: `PAUSE`, `ASAP` (5 s), `1m`, `5m`.
- A **live/paused indicator** and a **Refresh** button.
- The elapsed column **ticks every second** while the page is open.

### 8.2 Table columns

| Column | Shows |
| ------ | ----- |
| `PROJECT` | `workspace/repo_slug` of the owning project. |
| `#` | Build number. |
| `TARGET` | Branch + short commit hash. |
| `VIA` | Trigger type (`push`/`manual`/`schedule`/`PR`). |
| `STATUS` | Colour-coded badge (animated while running). |
| `ELAPSED` | Live wall-clock time since creation. |
| `CREATOR` | Display name or username. |
| `CREATED` | Creation timestamp. |

### 8.3 Clicking a row

Same modifier conventions as the Pipeline List:

- **Plain click** — switches the active project to the row's project and opens
  its Pipeline Detail.
- **Ctrl / Cmd / Shift + click** (or middle-click) — opens Detail in a new tab.
- **Ctrl / Cmd + Alt + click** — opens the run on bitbucket.org.

### 8.4 Hover variables

Hovering a row shows a popover with that pipeline's runtime variables (secured
values masked), fetched against the row's own project.

### 8.5 Per-repo errors

If some repositories couldn't be reached, a banner lists them individually
(`⚠ Some repositories could not be reached`) while still showing the rest.

### 8.6 States

Loading skeleton, error with Retry, and an empty state when nothing is running.

## 9. Manage Projects

Add and remove the `workspace/repo_slug` projects shown in the sidebar.

### 9.1 Header

- **Back** button and the title **Manage Projects** with a project count badge.
- **Add Project** button toggles the inline add form.

### 9.2 Add a project

The form has two fields and an **Add** button:

1. **Workspace** — as you type (≥ 2 chars), it **debounces and searches** the
   workspace's repositories via the API.
2. **Repository** — once the workspace is known, this field shows an
   **autocomplete dropdown** of matching repositories. Navigate with
   **↑/↓**, select with **Enter**, and **Esc** dismisses it.

Helpful status hints appear under the fields: "Searching repositories…",
"N repositories found", "No repositories found", or an error message.

On success the project list refreshes and the form collapses.

### 9.3 Project list

Each configured project is a card showing:

- A **coloured letter icon** (generated from the project name),
- The project **name** and `workspace/repo_slug`,
- An **Active** badge on the currently active project,
- **Open** — make this project active and view its pipeline list,
- **Remove** — deletes the project (with a confirmation modal).

### 9.4 Empty state

With no projects configured, a prompt invites you to add your first repository.

## 10. Notifications & confirmation modals

### 10.1 Toast notifications

Transient **success** (green) and **error** (red) toasts appear in the top-right
corner. They auto-dismiss (success ~5 s, errors ~8 s) and can be dismissed with
the ✕ button. They report events like "Project added", "Pipeline #123 started",
"Pipeline stopped", and API/validation errors.

### 10.2 Confirmation modals

Destructive actions (**Stop Pipeline**, **Remove Project**) open a modal with
**Cancel** and a confirm button:

- **Esc** cancels,
- **Enter** confirms,
- clicking the backdrop cancels,
- the confirm button shows the action ("Stop Pipeline", "Remove") with a
  danger (red) style.

## 11. Click modifiers & keyboard shortcuts

### 11.1 Click modifiers (work on pipeline rows everywhere)

| Input | Action |
| ----- | ------ |
| Plain click | Navigate in the current tab |
| **Ctrl / Cmd / Shift + click**, or **middle-click** | Open in a **new tab** |
| **Ctrl / Cmd + Alt + click** | Open the run on **bitbucket.org** in a new tab |

### 11.2 Pipeline List keyboard navigation

| Key | Action |
| --- | ------ |
| `↑` / `↓` | Move selection |
| `Enter` / `Space` | Open selected pipeline |

### 11.3 Step Logs keyboard shortcuts

| Key | Action |
| --- | ------ |
| `/` | Focus search |
| `n` / `N` | Next / previous search match |
| `Esc` | Clear search → close dropdown → back to Detail |

### 11.4 Manage Projects form keyboard navigation

| Key | Action |
| --- | ------ |
| `↑` / `↓` | Move through repository suggestions |
| `Enter` | Select highlighted suggestion |
| `Esc` | Dismiss suggestions |

### 11.5 Global shortcut behaviour

The keyboard manager ignores shortcuts while you're typing in an input/textarea
or select — except **Esc**, which always works. This prevents accidental
navigation while filling forms or searching.

## 12. Status & state resolution

Bitbucket's API reports pipeline/step state as two levels: a top-level
`state.name` (`IN_PROGRESS`, `PENDING`, `COMPLETED`, …) plus a
`state.result.name` (`SUCCESSFUL`, `FAILED`, …) when completed. The frontend
collapses this into the labels and colours you see:

| Bitbucket state | Displayed label | Colour |
| --------------- | --------------- | ------ |
| `IN_PROGRESS` | `running` | blue (animated pulse) |
| `PENDING` / `NOT_STARTED` | `pending` / `not started` | neutral |
| `COMPLETED` + `SUCCESSFUL` | `succeeded` | green |
| `COMPLETED` + `FAILED` / `ERROR` | `failed` | red |
| `STOPPED` / `IN_PROGRESS_STOPPING` | `stopped` | amber |
| steps after a failed/stopped step | `skipped` | muted |

This mirrors the TUI's resolution logic, so the webapp and terminal UI always
agree on what a given state means.

---

## Glossary

| Term | Meaning |
| ---- | ------- |
| **Active project** | The currently selected `workspace/repo_slug` (used for the list/detail/trigger screens). |
| **Build number** | Bitbucket's incrementing run number for a pipeline. |
| **Pipeline selector** | A pattern that chooses which steps run (e.g. `default`). |
| **Trigger** | What started the run: `push`, `manual`, `schedule`, or pull request. |
| **Log variables** | Variables parsed from the `Pipeline variables:` block in the first step's log. |
| **Secured variable** | A variable Bitbucket marks as secret; always masked in the UI. |










