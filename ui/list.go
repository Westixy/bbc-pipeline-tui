package ui

import (
	"fmt"
	"strings"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
			m.ListState = StateReady
			m.ListError = ""
			// Trigger preview for the first pipeline
			if len(m.Pipelines) > 0 {
				p := m.Pipelines[0]
				if p.UUID != m.PreviewUUID {
					m.PreviewUUID = p.UUID
					m.PreviewState = StateLoading
					workspace := m.Projects[m.ActiveProject].Workspace
					repoSlug := m.Projects[m.ActiveProject].RepoSlug
					return m, fetchPipelinePreview(m.Client, workspace, repoSlug, p.UUID)
				}
			}
		}
		return m, nil

	case nextPageLoadedMsg:
		if msg.err != nil {
			m.ListError = msg.err.Error()
		} else {
			m.Pipelines = append(m.Pipelines, msg.pipelines...)
			m.NextPageURL = msg.nextPageURL
		}
		return m, nil

	case pipelinePreviewLoadedMsg:
		if msg.err != nil {
			m.PreviewState = StateError
			m.PreviewError = msg.err.Error()
		} else {
			m.PreviewPipeline = msg.pipeline
			m.PreviewSteps = msg.steps
			m.PreviewConfigVars = msg.variables
			m.PreviewState = StateReady
			m.PreviewError = ""
		}
		return m, nil

	case tea.KeyMsg:
		// If filter active, handle in filterInput
		if m.ListFilter != "" {
			return m.handleFilterInput(msg)
		}

		var cmd tea.Cmd

		switch msg.String() {
		case "up", "k":
			if m.FocusPanel == 0 {
				if m.ListCursor > 0 {
					m.ListCursor--
				}
				m.ListScrollOff = clampCursor(m.ListCursor, len(m.Pipelines))
				cmd = m.maybeLoadPreviewCmd()
			} else {
				if m.PreviewScrollOff > 0 {
					m.PreviewScrollOff--
				}
			}

		case "down", "j":
			if m.FocusPanel == 0 {
				if m.ListCursor < len(m.Pipelines)-1 {
					m.ListCursor++
				}
				m.ListScrollOff = clampCursor(m.ListCursor, len(m.Pipelines))
				cmd = m.maybeLoadPreviewCmd()
			} else {
				maxScroll := m.maxPreviewScroll()
				if m.PreviewScrollOff < maxScroll {
					m.PreviewScrollOff++
				}
			}

		case "pgup":
			if m.FocusPanel == 0 {
				pageSize := m.listViewportHeight()
				m.ListCursor -= pageSize
				if m.ListCursor < 0 {
					m.ListCursor = 0
				}
				m.ListScrollOff = clampCursor(m.ListCursor, len(m.Pipelines))
				cmd = m.maybeLoadPreviewCmd()
			} else {
				pageSize := m.previewViewportHeight()
				m.PreviewScrollOff -= pageSize
				if m.PreviewScrollOff < 0 {
					m.PreviewScrollOff = 0
				}
			}

		case "pgdown":
			if m.FocusPanel == 0 {
				pageSize := m.listViewportHeight()
				m.ListCursor += pageSize
				if m.ListCursor >= len(m.Pipelines) {
					m.ListCursor = len(m.Pipelines) - 1
				}
				if m.ListCursor < 0 {
					m.ListCursor = 0
				}
				m.ListScrollOff = clampCursor(m.ListCursor, len(m.Pipelines))
				cmd = m.maybeLoadPreviewCmd()
			} else {
				pageSize := m.previewViewportHeight()
				maxScroll := m.maxPreviewScroll()
				m.PreviewScrollOff += pageSize
				if m.PreviewScrollOff > maxScroll {
					m.PreviewScrollOff = maxScroll
				}
			}

		case "home":
			if m.FocusPanel == 0 {
				m.ListCursor = 0
				m.ListScrollOff = 0
				cmd = m.maybeLoadPreviewCmd()
			}

		case "end":
			if m.FocusPanel == 0 && len(m.Pipelines) > 0 {
				m.ListCursor = len(m.Pipelines) - 1
				m.ListScrollOff = clampCursor(m.ListCursor, len(m.Pipelines))
				cmd = m.maybeLoadPreviewCmd()
			}

		case "tab":
			m.FocusPanel = 1 - m.FocusPanel

		case "enter":
			if m.PreviewPipeline != nil {
				m.Screen = ScreenDetail
				m.SelectedPipeline = m.PreviewPipeline
				m.Steps = m.PreviewSteps
				m.ConfigVars = m.PreviewConfigVars
				m.DetailState = m.PreviewState
				m.DetailError = m.PreviewError
			}

		case "n":
			if m.NextPageURL != "" && m.FocusPanel == 0 {
				cmd = fetchNextPage(m.Client, m.NextPageURL)
			}

		case "r":
			m.Client.InvalidateCacheForRepo(m.Projects[m.ActiveProject].Workspace, m.Projects[m.ActiveProject].RepoSlug)
			m.Pipelines = nil
			m.NextPageURL = ""
			m.ListCursor = 0
			m.ListFilter = ""
			m.ListScrollOff = 0
			m.ListState = StateLoading
			m.ListError = ""
			m.PreviewPipeline = nil
			m.PreviewSteps = nil
			m.PreviewConfigVars = nil
			m.PreviewUUID = ""
			m.PreviewState = 0
			m.PreviewError = ""
			m.PreviewScrollOff = 0
			workspace := m.Projects[m.ActiveProject].Workspace
			repoSlug := m.Projects[m.ActiveProject].RepoSlug
			cmd = fetchPipelines(m.Client, workspace, repoSlug)

		case "p":
			m.Screen = ScreenProjects

		case "/":
			m.ListFilter = "/"

		case "esc":
			if m.ListFilter != "" {
				m.ListFilter = ""
			}

		case "q":
			return m, tea.Quit

		case "ctrl+c":
			return m, tea.Quit
		}

		return m, cmd
	}
	return m, nil
}

// handleFilterInput handles key presses when the filter bar is active.
func (m *Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m, nil
	case "esc":
		m.ListFilter = ""
		return m, nil
	default:
		if len(msg.String()) == 1 {
			if m.ListFilter == "/" {
				m.ListFilter = msg.String()
			} else {
				m.ListFilter += msg.String()
			}
		}
		return m, nil
	}
}

// maybeLoadPreviewCmd returns a Cmd to load the preview for the currently
// highlighted pipeline, if it differs from the already-loaded preview.
func (m *Model) maybeLoadPreviewCmd() tea.Cmd {
	if m.ListCursor < 0 || m.ListCursor >= len(m.Pipelines) {
		return nil
	}
	p := m.Pipelines[m.ListCursor]
	if p.UUID == m.PreviewUUID {
		return nil
	}
	m.PreviewUUID = p.UUID
	m.PreviewState = StateLoading
	m.PreviewScrollOff = 0

	workspace := m.Projects[m.ActiveProject].Workspace
	repoSlug := m.Projects[m.ActiveProject].RepoSlug
	return fetchPipelinePreview(m.Client, workspace, repoSlug, p.UUID)
}

// ── List View ───────────────────────────────────────────────────────────────

// viewList renders the pipeline list with a split-panel preview on the right.
func (m Model) viewList(contentHeight int) string {
	if m.ListState == StateLoading {
		return viewLoading("Loading pipelines")
	}
	if m.ListState == StateError {
		return viewError(m.ListError)
	}
	if len(m.Pipelines) == 0 {
		return viewEmpty("No pipelines", "Press 'r' to refresh or 'p' to change project")
	}

	avail := m.Width - 4

	// ── Compute split widths ──────────────────────────────────────────────
	leftW := avail * 55 / 100
	if leftW < 55 {
		leftW = 55
	}
	rightW := avail - leftW - 3 // 3 for divider " │ "
	if rightW < 30 {
		// Not enough space for split — fall back to list-only
		return m.viewListCompact(contentHeight)
	}

	m.SplitLeftWidth = leftW

	leftContent := m.viewListLeft(leftW, contentHeight)
	rightContent := m.viewListRight(rightW, contentHeight)

	divider := DimmedStyle.Render(" │ ")

	leftHeight := lipgloss.Height(leftContent)
	rightHeight := lipgloss.Height(rightContent)
	maxH := leftHeight
	if rightHeight > maxH {
		maxH = rightHeight
	}

	if leftHeight < maxH {
		leftContent += strings.Repeat("\n", maxH-leftHeight)
	}
	if rightHeight < maxH {
		rightContent += strings.Repeat("\n", maxH-rightHeight)
	}

	leftLines := strings.Split(leftContent, "\n")
	rightLines := strings.Split(rightContent, "\n")

	var rows []string
	for i := 0; i < maxH; i++ {
		leftLine := ""
		if i < len(leftLines) {
			leftLine = lipgloss.NewStyle().Width(leftW).Render(leftLines[i])
		} else {
			leftLine = strings.Repeat(" ", leftW)
		}
		rightLine := ""
		if i < len(rightLines) {
			rightLine = rightLines[i]
		}
		rows = append(rows, leftLine+divider+rightLine)
	}

	return strings.Join(rows, "\n")
}

// viewListCompact renders the list only (fallback when terminal is too narrow).
func (m Model) viewListCompact(contentHeight int) string {
	return m.viewListLeft(m.Width-4, contentHeight)
}

// viewListLeft renders the left pipeline list panel.
func (m Model) viewListLeft(width int, contentHeight int) string {
	filtered := filterPipelines(m.Pipelines, m.ListFilter)
	if len(filtered) == 0 {
		return viewEmpty("No matches", "esc to clear filter")
	}

	colNum := 5
	colStatus := 12
	colType := 28
	colBranch := width - colNum - colStatus - colType - 13
	if colBranch < 10 {
		colBranch = 10
	}
	bNumW := colNum
	bStatusW := colStatus
	bTypeW := colType
	bBranchW := colBranch

	headerLine := ListHeaderStyle.Render(
		" " +
			padToWidth("#", bNumW) +
			padToWidth("STATUS", bStatusW) +
			padToWidth("TYPE", bTypeW) +
			"BRANCH")

	var sb strings.Builder
	sb.WriteString(headerLine)
	sb.WriteString("\n")
	sb.WriteString(DividerStyle.Render(strings.Repeat("─", width)))
	sb.WriteString("\n")

	viewportHeight := contentHeight - 4
	if viewportHeight < 1 {
		viewportHeight = 1
	}

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

	for i := start; i < end; i++ {
		p := filtered[i]
		rowStyle := ListNormalStyle
		prefix := " "
		if cursor == i {
			rowStyle = ListCursorStyle
			if m.FocusPanel == 0 {
				prefix = "▶"
			} else {
				prefix = " "
			}
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
		typeLabel := truncate(pipelineTypeLabel(p.Target), bTypeW-1)
		branch := truncate(p.Target.RefName, bBranchW-1)

		row := prefix +
			padToWidth(fmt.Sprintf("#%d", p.BuildNumber), bNumW) +
			padToWidth(statusBadge, bStatusW) +
			padToWidth(typeLabel, bTypeW) +
			"⎇ " + branch + " "

		row = padToWidth(row, width)
		sb.WriteString(rowStyle.Render(row))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(DividerStyle.Render(strings.Repeat("─", width)))
	sb.WriteString("\n")

	filterInfo := ""
	if m.ListFilter != "" {
		filterInfo = FilterStyle.Render(" Filter: "+m.ListFilter+" ") + " │ "
	}

	pageInfo := fmt.Sprintf("%d pipelines", len(m.Pipelines))
	if m.NextPageURL != "" {
		pageInfo += "  ┃  " + HelpKeyStyle.Render("n") + " more..."
	}

	scrollInfo := DimmedStyle.Render(
		fmt.Sprintf("%s%d-%d of %d", filterInfo, start+1, end, len(m.Pipelines)))
	if pageInfo != "" {
		scrollInfo += "  " + DimmedStyle.Render(pageInfo)
	}

	sb.WriteString(scrollInfo)
	return sb.String()
}

// viewListRight renders the right preview panel with pipeline details.
func (m Model) viewListRight(width int, contentHeight int) string {
	if width < 20 {
		return ""
	}

	innerW := width - 2

	focusIndicator := ""
	if m.FocusPanel == 1 {
		focusIndicator = "◀ "
	}
	title := focusIndicator + "Preview"
	titleLine := PreviewTitleStyle.Render(padToWidth(title, width))
	div := DividerStyle.Render(strings.Repeat("─", width))

	if m.PreviewState == StateLoading {
		return lipgloss.JoinVertical(lipgloss.Left,
			titleLine,
			div,
			viewLoading("Loading..."),
		)
	}
	if m.PreviewState == StateError {
		return lipgloss.JoinVertical(lipgloss.Left,
			titleLine,
			div,
			viewError(m.PreviewError),
		)
	}
	if m.PreviewPipeline == nil {
		placeholder := DimmedStyle.Render(
			lipgloss.Place(width, contentHeight-3, lipgloss.Center, lipgloss.Center, "Select a pipeline"))
		return lipgloss.JoinVertical(lipgloss.Left,
			titleLine,
			div,
			placeholder,
		)
	}

	p := m.PreviewPipeline

	status := resolvePipelineResult(p.State.Name, func() string {
		if p.State.Result != nil {
			return p.State.Result.Name
		}
		return ""
	}())
	badge := renderStatusBadge(status)

	overview := fmt.Sprintf("%s\n\n%s\n\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
		CardTitleStyle.Render(fmt.Sprintf("#%d", p.BuildNumber)),
		badge,
		KeyStyle.Render("Branch:"), ValueStyle.Render(truncate(p.Target.RefName, innerW-10)),
		KeyStyle.Render("Type:"), ValueStyle.Render(pipelineTypeLabel(p.Target)),
		KeyStyle.Render("Trigger:"), ValueStyle.Render(truncate(p.Trigger.Name, innerW-12)),
		KeyStyle.Render("Duration:"), ValueStyle.Render(formatDuration(p.CreatedOn, p.CompletedOn, p.BuildSecondsUsed)),
		KeyStyle.Render("Created:"), ValueStyle.Render(formatRelativeTime(p.CreatedOn)),
		KeyStyle.Render("By:"), ValueStyle.Render(creatorName(p.Creator)),
	)

	var variablesView string
	if len(m.PreviewConfigVars) > 0 {
		variablesView = buildPreviewVariables(m.PreviewConfigVars, innerW)
	}

	var stepsView string
	if len(m.PreviewSteps) > 0 {
		maxSteps := m.previewStepsMax(contentHeight)
		stepsView = buildPreviewSteps(m.PreviewSteps, m.PreviewScrollOff, maxSteps, innerW)
	} else {
		stepsView = DimmedStyle.Render("  No steps")
	}

	var sb strings.Builder
	sb.WriteString(titleLine)
	sb.WriteString("\n")
	sb.WriteString(div)
	sb.WriteString("\n")
	sb.WriteString(overview)

	if variablesView != "" {
		sb.WriteString("\n")
		sb.WriteString(DividerStyle.Render(strings.Repeat("─", width)))
		sb.WriteString("\n")
		sb.WriteString(SectionTitleStyle.Render(fmt.Sprintf("Parsed Log Variables (%d)", len(m.PreviewConfigVars))))
		sb.WriteString("\n")
		sb.WriteString(variablesView)
	}

	sb.WriteString("\n")
	sb.WriteString(DividerStyle.Render(strings.Repeat("─", width)))
	sb.WriteString("\n")
	sb.WriteString(SectionTitleStyle.Render(fmt.Sprintf("Steps (%d)", len(m.PreviewSteps))))
	sb.WriteString("\n")
	sb.WriteString(stepsView)

	maxScroll := m.maxPreviewScroll()
	if maxScroll > 0 {
		scrollPct := m.PreviewScrollOff * 100 / maxScroll
		sb.WriteString("\n")
		sb.WriteString(DimmedStyle.Render(
			fmt.Sprintf("  scroll %d%% (pgup/pgdn)  Tab to focus", scrollPct)))
	}

	return sb.String()
}

// buildPreviewSteps renders a compact steps list for the preview panel.
func buildPreviewSteps(steps []bitbucket.PipelineStep, scrollOff, maxVisible, width int) string {
	if len(steps) == 0 {
		return DimmedStyle.Render("  No steps")
	}

	start := scrollOff
	end := start + maxVisible
	if end > len(steps) {
		end = len(steps)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	var sb strings.Builder

	statusW := 8
	nameW := width - statusW - 6
	if nameW < 8 {
		nameW = 8
	}

	sb.WriteString(ListHeaderStyle.Render(
		fmt.Sprintf("  %-3s %-*s %s", "#", nameW, "STEP", "STATUS")))
	sb.WriteString("\n")

	if start > 0 {
		sb.WriteString(DimmedStyle.Render(fmt.Sprintf("  ... %d more", start)))
		sb.WriteString("\n")
	}

	for i := start; i < end; i++ {
		s := steps[i]
		status := resolveStepStatus(s.State)
		sb.WriteString(
			fmt.Sprintf("  %-3d %-*s %s\n",
				i+1,
				nameW,
				truncate(s.Name, nameW-1),
				truncate(status, statusW-1)),
		)
	}

	if end < len(steps) {
		sb.WriteString(DimmedStyle.Render(fmt.Sprintf("  ... %d more", len(steps)-end)))
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildPreviewVariables renders a compact variables list for the preview panel.
func buildPreviewVariables(vars []bitbucket.PipelineVariable, width int) string {
	if len(vars) == 0 {
		return DimmedStyle.Render("  No variables")
	}

	keyW := width/2 - 2
	if keyW < 10 {
		keyW = 10
	}
	valW := width - keyW - 4
	if valW < 10 {
		valW = 10
	}

	var sb strings.Builder

	sb.WriteString(ListHeaderStyle.Render(
		fmt.Sprintf("  %-*s %s", keyW, "KEY", "VALUE")))
	sb.WriteString("\n")

	for _, v := range vars {
		value := v.Value
		// Mask secured variables
		if v.Secured {
			value = "••••••••"
		}
		sb.WriteString(
			fmt.Sprintf("  %-*s %s\n",
				keyW,
				truncate(v.Key, keyW-1),
				truncate(value, valW-1)),
		)
	}

	return sb.String()
}

// ── Helper methods ──────────────────────────────────────────────────────────

func (m Model) maxPreviewScroll() int {
	maxVisible := m.previewStepsMax(m.Height - 6)
	if len(m.PreviewSteps) <= maxVisible {
		return 0
	}
	return len(m.PreviewSteps) - maxVisible
}

func (m Model) previewStepsMax(contentHeight int) int {
	overhead := 13
	available := contentHeight - overhead
	if available < 3 {
		return 3
	}
	return available
}

func (m Model) previewViewportHeight() int {
	return m.previewStepsMax(m.Height - 6)
}

func (m Model) listViewportHeight() int {
	leftW := m.SplitLeftWidth
	if leftW < 55 {
		leftW = m.Width - 4
	}
	vh := (m.Height - 6) - 4
	if vh < 1 {
		return 1
	}
	return vh
}

// filterPipelines filters pipelines by the given filter text (case-insensitive).
func filterPipelines(pipelines []bitbucket.Pipeline, filter string) []bitbucket.Pipeline {
	if filter == "" || filter == "/" {
		return pipelines
	}
	lower := strings.ToLower(filter)
	var filtered []bitbucket.Pipeline
	for _, p := range pipelines {
		if strings.Contains(strings.ToLower(p.Target.RefName), lower) ||
			strings.Contains(strings.ToLower(fmt.Sprintf("#%d", p.BuildNumber)), lower) ||
			strings.Contains(strings.ToLower(p.State.Name), lower) ||
			(p.State.Result != nil && strings.Contains(strings.ToLower(p.State.Result.Name), lower)) ||
			(p.Trigger.Name != "" && strings.Contains(strings.ToLower(p.Trigger.Name), lower)) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
