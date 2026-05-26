package ui

import (
	"fmt"
	"strings"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// updateList handles messages for the pipeline list screen.
func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pipelinesLoadedMsg:
		if msg.err != nil {
			m.handleErrorMsg("list", msg.err)
			return m, nil
		}
		m.Pipelines = msg.pipelines
		m.NextPageURL = msg.nextPageURL
		m.ListState = StateReady
		m.ListCursor = 0
		m.ListScrollOff = 0
		return m, nil

	case nextPageLoadedMsg:
		if msg.err != nil {
			m.handleErrorMsg("list", msg.err)
			return m, nil
		}
		if len(msg.pipelines) > 0 {
			m.Pipelines = append(m.Pipelines, msg.pipelines...)
			m.NextPageURL = msg.nextPageURL
		}
		return m, nil

	case tea.KeyMsg:
		filtered := filteredPipelines(m.Pipelines, m.ListFilter)

		switch msg.String() {
		case "up", "k":
			m.ListCursor = clampCursor(m.ListCursor-1, len(filtered))

		case "down", "j":
			m.ListCursor = clampCursor(m.ListCursor+1, len(filtered))

		case "enter":
			if len(filtered) > 0 && m.ListCursor >= 0 && m.ListCursor < len(filtered) {
				p := filtered[m.ListCursor]
				m.SelectedPipeline = &p
				m.Screen = ScreenDetail
				m.DetailState = StateLoading
				m.DetailError = ""
				cmd := fetchPipelineDetail(m.Client, m.Projects[m.ActiveProject].Workspace,
					m.Projects[m.ActiveProject].RepoSlug, p.UUID)
				return m, cmd
			}

		case "r":
			m.ListState = StateLoading
			m.Pipelines = nil
			m.ListFilter = ""
			m.ListScrollOff = 0
			return m, fetchPipelines(m.Client, m.Projects[m.ActiveProject].Workspace, m.Projects[m.ActiveProject].RepoSlug)

		case "n":
			if m.NextPageURL != "" {
				prev := &bitbucket.PaginatedPipelines{
					PaginatedResponse: bitbucket.PaginatedResponse{
						Next: m.NextPageURL,
					},
				}
				params := &bitbucket.ListPipelinesParams{
					Pagelen: 25,
					Sort:    "-created_on",
				}
				return m, fetchNextPage(m.Client, prev, params)
			}

		case "/":
			return m, func() tea.Msg {
				return filterPromptMsg{}
			}

		default:
			if m.ListFilter != "" || msg.String() == "/" {
				if len(msg.String()) == 1 && msg.String()[0] >= 32 && msg.String()[0] < 127 {
					m.ListFilter += msg.String()
					m.ListCursor = 0
					return m, nil
				}
			}
		}

		if msg.Type == tea.KeyBackspace || msg.String() == "backspace" {
			if len(m.ListFilter) > 0 {
				m.ListFilter = m.ListFilter[:len(m.ListFilter)-1]
				m.ListCursor = 0
			}
			return m, nil
		}
	}

	return m, nil
}

type filterPromptMsg struct{}

// viewList renders the pipeline list screen, filling the available height.
// contentHeight is the number of rows available for content.
func (m Model) viewList(contentHeight int) string {
	switch m.ListState {
	case StateLoading:
		return viewLoading("Loading pipelines for " + m.FullRepoName() + "...")
	case StateError:
		return viewError(m.ListError)
	}

	filtered := filteredPipelines(m.Pipelines, m.ListFilter)
	width := m.Width - 2 // account for content padding
	if width < 40 {
		width = 40
	}

	// Measure status badge visual width once
	sampleStatus := renderStatusBadge("IN_PROGRESS")
	statusVisW := lipgloss.Width(sampleStatus)
	// Layout: Build(6) | spacer | Branch(fixed) | spacer | Status(visual) | spacer | Author(fixed) | spacer | Trigger(flex) | spacer | Created(19) | spacer | Duration(10)
	buildW := 6
	createdW := 19
	durationW := 10
	authorW := 20
	statusW := statusVisW
	// Fixed columns + 7 separators
	fixed := buildW + statusW + createdW + durationW + authorW + 7
	remaining := width - fixed
	if remaining < 10 {
		remaining = 10
	}
	branchW := remaining * 38 / 100
	triggerW := remaining - branchW
	if triggerW < 6 {
		triggerW = 6
	}

	// Compute viewport lines available for items (subtract header, filter, divider, footer)
	nonItemLines := 1 // column header
	if m.ListFilter != "" {
		nonItemLines += 1 // filter bar
	}
	nonItemLines += 2 // divider + pagination footer
	viewportHeight := contentHeight - nonItemLines
	if viewportHeight < 0 {
		viewportHeight = 0
	}

	// Auto-scroll: ensure cursor is visible in viewport
	if m.ListCursor < m.ListScrollOff {
		m.ListScrollOff = m.ListCursor
	}
	if m.ListCursor >= m.ListScrollOff+viewportHeight && viewportHeight > 0 {
		m.ListScrollOff = m.ListCursor - viewportHeight + 1
	}
	if m.ListScrollOff < 0 {
		m.ListScrollOff = 0
	}

	// Slice items to viewport
	start := m.ListScrollOff
	end := start + viewportHeight
	if end > len(filtered) {
		end = len(filtered)
	}
	if start > len(filtered) {
		start = len(filtered)
	}

	var sb strings.Builder

	// Filter bar
	if m.ListFilter != "" {
		sb.WriteString(FilterStyle.Render(fmt.Sprintf("Filter: %s", m.ListFilter)))
		sb.WriteString("\n")
	}

	// Column headers using padToWidth for alignment
	header := DetailSectionStyle.Render(
		padToWidth("#", buildW) + " " +
			padToWidth("Branch", branchW) + " " +
			padToWidth("Status", statusW) + " " +
			padToWidth("Author", authorW) + " " +
			padToWidth("Trigger", triggerW) + " " +
			padToWidth("Created", createdW) + " " +
			"Duration",
	)
	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(DimmedStyle.Render(strings.Repeat("─", width)))
	sb.WriteString("\n")

	// List items — each column padded to visual width with padToWidth
	for i := start; i < end; i++ {
		p := filtered[i]
		buildNum := fmt.Sprintf("#%d", p.BuildNumber)
		branch := truncate(p.Target.RefName, branchW)
		status := renderStatusBadge(mergeStatus(p.State))
		author := truncate(creatorName(p.Creator), authorW)
		trigger := truncate(p.Trigger.Name, triggerW)
		created := formatTime(p.CreatedOn)
		duration := formatDuration(p.CreatedOn, p.CompletedOn, p.BuildSecondsUsed)

		line := padToWidth(buildNum, buildW) + " " +
			padToWidth(branch, branchW) + " " +
			padToWidth(status, statusW) + " " +
			padToWidth(author, authorW) + " " +
			padToWidth(trigger, triggerW) + " " +
			padToWidth(created, createdW) + " " +
			duration

		if i == m.ListCursor {
			sb.WriteString(ListCursorStyle.Render(line))
		} else {
			sb.WriteString(ListNormalStyle.Render(line))
		}
		sb.WriteString("\n")
	}

	if len(filtered) == 0 {
		sb.WriteString(DimmedStyle.Render("No pipelines found"))
		sb.WriteString("\n")
	}

	// Fill remaining lines with empty space if needed
	if end-start < viewportHeight {
		for i := 0; i < viewportHeight-(end-start); i++ {
			sb.WriteString("\n")
		}
	}

	// Pagination footer
	sb.WriteString(DimmedStyle.Render(strings.Repeat("─", width)))
	sb.WriteString("\n")
	pageInfo := fmt.Sprintf("Showing %d pipelines", len(filtered))
	if m.ListScrollOff > 0 || end < len(filtered) {
		pageInfo += fmt.Sprintf(" (%d-%d of %d)", start+1, end, len(filtered))
	}
	if m.NextPageURL != "" {
		pageInfo += " | Press 'n' for next page"
	}
	sb.WriteString(DimmedStyle.Render(pageInfo))

	return sb.String()
}

// mergeStatus merges the pipeline state name with result.
func mergeStatus(state bitbucket.PipelineState) string {
	if state.Name == "COMPLETED" && state.Result != nil {
		return state.Result.Name
	}
	return state.Name
}