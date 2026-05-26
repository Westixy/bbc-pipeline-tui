package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ── List Update ─────────────────────────────────────────────────────────────

// updateList handles messages for the pipeline list screen.
func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pipelinesLoadedMsg:
		if msg.err != nil {
			m.ListState = StateError
			m.ListError = msg.err.Error()
		} else {
			m.Pipelines = msg.pipelines
			m.NextPageURL = msg.nextPageURL
			m.ListCursor = 0
			m.ListState = StateReady
		}
		return m, nil

	case nextPageLoadedMsg:
		if msg.err != nil {
			m.ListState = StateError
			m.ListError = msg.err.Error()
		} else {
			m.Pipelines = append(m.Pipelines, msg.pipelines...)
			m.NextPageURL = msg.nextPageURL
			m.ListState = StateReady
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.ListCursor > 0 {
				m.ListCursor--
			}
			return m, nil

		case "down", "j":
			if m.ListCursor < len(m.Pipelines)-1 {
				m.ListCursor++
			}
			return m, nil

		case "enter":
			if len(m.Pipelines) == 0 {
				return m, nil
			}
			// Navigate to pipeline detail
			m.SelectedPipeline = &m.Pipelines[m.ListCursor]
			m.Screen = ScreenDetail
			m.StepCursor = 0
			m.DetailState = StateLoading
			m.DetailError = ""
			m.ParsedLogVars = nil
			m.LogShowVars = false
			return m, tea.Batch(
				fetchPipelineDetail(m.Client, m.Projects[m.ActiveProject].Workspace, m.Projects[m.ActiveProject].RepoSlug, m.SelectedPipeline.UUID),
				fetchConfigVars(m.Client, m.Projects[m.ActiveProject].Workspace, m.Projects[m.ActiveProject].RepoSlug),
				fetchLogVarsForPipeline(m.Client, m.Projects[m.ActiveProject].Workspace, m.Projects[m.ActiveProject].RepoSlug, m.SelectedPipeline.UUID),
			)

		case "r":
			m.ListState = StateLoading
			m.Pipelines = nil
			m.ListCursor = 0
			return m, fetchPipelines(m.Client,
				m.Projects[m.ActiveProject].Workspace,
				m.Projects[m.ActiveProject].RepoSlug)

		case "p":
			// Show project switcher
			m.Screen = ScreenProjects
			return m, nil

		case "n":
			// Load next page
			if m.NextPageURL != "" && m.ListState == StateReady {
				m.ListState = StateLoading
				return m, fetchNextPage(m.Client, m.NextPageURL)
			}
			return m, nil

		case "/":
			// Start filtering (built into list via ListFilter)
			return m, nil

		case "esc":
			m.ListFilter = ""
			return m, nil

		default:
			// Any other key — add to filter if printable
			if len(msg.String()) == 1 && msg.String() >= " " && msg.String() != "/" {
				m.ListFilter += msg.String()
				// Clamp cursor to filtered results
				filtered := filteredPipelines(m.Pipelines, m.ListFilter)
				if m.ListCursor >= len(filtered) && len(filtered) > 0 {
					m.ListCursor = len(filtered) - 1
				}
			}
			return m, nil
		}
	}
	return m, nil
}

// ── List View ───────────────────────────────────────────────────────────────

// viewList renders the pipeline list with column headers, zebra-striped rows,
// and a filter bar.
func (m Model) viewList(contentHeight int) string {
	if m.ListState == StateLoading {
		return viewLoading("Loading pipelines")
	}
	if m.ListState == StateError {
		return viewError(m.ListError)
	}

	filtered := filteredPipelines(m.Pipelines, m.ListFilter)

	// Calculate column widths dynamically
	// Columns: #, Status, Branch, Type, Trigger, Duration, Created
	// Adapt to available width — approximate based on terminal width
	avail := m.Width - 4 // padding

	// Fixed: # (5), Duration (7), Type (12)
	// Flexible: Status (14), Branch, Trigger, Created (16)
	fixedWidths := 5 + 14 + 7 + 16 // 42
	_ = 3                          // Branch, Trigger, Type
	// Don't let fixed exceed available
	if fixedWidths >= avail {
		fixedWidths = avail - 20
		if fixedWidths < 40 {
			fixedWidths = 40
		}
	}
	colBranch := 16
	colTrigger := 12
	colType := 12
	remaining := avail - 42
	if remaining > 0 {
		// Distribute remaining to flexible columns
		colBranch += remaining / 3
		colTrigger += remaining / 3
		colType += remaining / 3
	}
	if colBranch > 30 {
		colBranch = 30
	}
	if colTrigger > 20 {
		colTrigger = 20
	}

	bNumW := 5
	bStatusW := 14
	bBranchW := colBranch
	bTypeW := colType
	bTriggerW := colTrigger
	bDurW := 7
	bCreatedW := 16

	// Build header
	headerLine := ListHeaderStyle.Render(
		" " +
			padToWidth("#", bNumW) +
			padToWidth("STATUS", bStatusW) +
			padToWidth("BRANCH", bBranchW) +
			padToWidth("TYPE", bTypeW) +
			padToWidth("TRIGGER", bTriggerW) +
			padToWidth("DUR", bDurW) +
			padToWidth("CREATED", bCreatedW))

	var sb strings.Builder
	sb.WriteString(headerLine)
	sb.WriteString("\n")
	sb.WriteString(DividerStyle.Render(strings.Repeat("─", avail)))
	sb.WriteString("\n")

	// Calculate viewport
	nonItemLines := 4 // header + divider + footer filter bar + bottom spacer
	viewportHeight := contentHeight - nonItemLines
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	if len(filtered) == 0 {
		if m.ListFilter != "" {
			sb.WriteString(viewEmpty("No pipelines match filter: "+m.ListFilter, "esc to clear / type to refine"))
		} else {
			sb.WriteString(viewEmpty("No pipelines found", "Press 'r' to refresh or 'p' to switch project"))
		}
		sb.WriteString("\n")
	} else {
		// Clamp cursor and compute visible slice
		cursor := clampCursor(m.ListCursor, len(filtered))
		start := cursor - viewportHeight/2
		if start < 0 {
			start = 0
		}
		end := start + viewportHeight
		if end > len(filtered) {
			end = len(filtered)
		}
		if end-start < viewportHeight && start > 0 {
			start = end - viewportHeight
			if start < 0 {
				start = 0
			}
		}

		// Build visible rows with zebra striping
		for i := start; i < end; i++ {
			p := filtered[i]
			rowStyle := ListNormalStyle
			if cursor == i {
				rowStyle = ListCursorStyle
			} else if i%2 == 1 {
				rowStyle = ListAltStyle
			}

			status := resolvePipelineResult(p.State.Name, func() string {
				if p.State.Result != nil {
					return p.State.Result.Name
				}
				return ""
			}())
			statusBadge := statusBadgeCompact(status)

			branch := truncate(p.Target.RefName, bBranchW-1)
			typeLabel := truncate(pipelineTypeLabel(p.Target), bTypeW-1)
			trigger := truncate(p.Trigger.Name, bTriggerW-1)
			if trigger == "" {
				trigger = "—"
			}
			dur := formatDuration(p.CreatedOn, p.CompletedOn, p.BuildSecondsUsed)
			created := formatRelativeTime(p.CreatedOn)
			if created == "" {
				created = formatTime(p.CreatedOn)
			}

			// Build row with ANSI-aware padding so status colors don't misalign columns
			row := " " + // 1-char left padding
				padToWidth(fmt.Sprintf("#%d", p.BuildNumber), bNumW) +
				padToWidth(statusBadge, bStatusW) +
				padToWidth("⎇ "+branch, bBranchW) +
				padToWidth(typeLabel, bTypeW) +
				padToWidth(trigger, bTriggerW) +
				padToWidth(dur, bDurW) +
				padToWidth(created, bCreatedW)

			sb.WriteString(rowStyle.Render(row))
			sb.WriteString("\n")
		}
	}

	// Footer: filter bar + pagination info
	sb.WriteString("\n")
	sb.WriteString(DividerStyle.Render(strings.Repeat("─", avail)))
	sb.WriteString("\n")

	filterInfo := ""
	if m.ListFilter != "" {
		filterInfo = FilterStyle.Render(" Filter: "+m.ListFilter+" ") + " │ "
	}

	pageInfo := fmt.Sprintf("%d pipelines", len(m.Pipelines))
	if m.NextPageURL != "" {
		pageInfo += "  ┃  " + HelpKeyStyle.Render("n") + " more..."
	}

	sb.WriteString(DimmedStyle.Render(filterInfo + pageInfo))

	return sb.String()
}
