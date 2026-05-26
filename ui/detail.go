package ui

import (
	"fmt"
	"strings"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	tea "github.com/charmbracelet/bubbletea"
)

// ── Detail Update ───────────────────────────────────────────────────────────

// updateDetail handles messages for the pipeline detail screen.
func (m Model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.StepCursor > 0 {
				m.StepCursor--
			}
			return m, nil

		case "down", "j":
			if m.StepCursor < len(m.Steps)-1 {
				m.StepCursor++
			}
			return m, nil

		case "enter":
			// Navigate to run screen
			m.Screen = ScreenRun
			m.RunFocus = 0
			m.RunState = StateReady
			m.RunBranch.SetValue(m.SelectedPipeline.Target.RefName)
			if m.SelectedPipeline.Target.Selector != nil {
				m.RunSelector.SetValue(m.SelectedPipeline.Target.Selector.Pattern)
			} else {
				m.RunSelector.SetValue("")
			}
			m.RunEditMode = false
			m.RunVarCursor = 0
			m.RunError = ""
			m.RunSuccessMsg = ""
			if len(m.ParsedLogVars) > 0 {
				m.RunVars = make([]bitbucket.PipelineVariable, len(m.ParsedLogVars))
				copy(m.RunVars, m.ParsedLogVars)
			} else {
				m.RunVars = make([]bitbucket.PipelineVariable, len(m.ConfigVars))
				copy(m.RunVars, m.ConfigVars)
			}
			return m, nil

		case "l":
			// View step log
			if len(m.Steps) > 0 {
				step := m.Steps[m.StepCursor]
				m.LogStepName = step.Name
				m.LogContent = ""
				m.LogState = StateLoading
				m.Screen = ScreenLogs
				workspace := m.Projects[m.ActiveProject].Workspace
				repoSlug := m.Projects[m.ActiveProject].RepoSlug
				return m, fetchStepLog(m.Client,
					workspace, repoSlug,
					m.SelectedPipeline.UUID,
					step.UUID, step.Name)
			}
			return m, nil

		case "v":
			// Parse pipeline variables from step log
			if m.SelectedPipeline != nil {
				m.DetailState = StateLoading
				workspace := m.Projects[m.ActiveProject].Workspace
				repoSlug := m.Projects[m.ActiveProject].RepoSlug
				return m, fetchLogVarsForPipeline(m.Client,
					workspace, repoSlug,
					m.SelectedPipeline.UUID)
			}
			return m, nil

		case "r":
			m.DetailState = StateLoading
			m.Steps = nil
			m.StepCursor = 0
			workspace := m.Projects[m.ActiveProject].Workspace
			repoSlug := m.Projects[m.ActiveProject].RepoSlug
			return m, tea.Batch(
				fetchPipelineDetail(m.Client, workspace, repoSlug, m.SelectedPipeline.UUID),
				fetchConfigVars(m.Client, workspace, repoSlug),
			)

		case "esc":
			m.Screen = ScreenList
			return m, nil
		}
		return m, nil

	case pipelineDetailLoadedMsg:
		if msg.err != nil {
			m.DetailState = StateError
			m.DetailError = msg.err.Error()
		} else {
			m.SelectedPipeline = msg.pipeline
			m.Steps = msg.steps
			m.DetailState = StateReady
			m.DetailError = ""
		}
		return m, nil

	case configVarsLoadedMsg:
		if msg.err != nil {
			m.DetailError = msg.err.Error()
		} else {
			m.ConfigVars = msg.variables
		}
		return m, nil

	case logVarsLoadedMsg:
		if msg.err != nil {
			m.DetailError = msg.err.Error()
		} else {
			m.ParsedLogVars = msg.variables
		}
		return m, nil
	}
	return m, nil
}

// ── Detail View ─────────────────────────────────────────────────────────────

// viewDetail renders the pipeline detail screen.
func (m Model) viewDetail(contentHeight int) string {
	if m.DetailState == StateLoading {
		return viewLoading("Loading pipeline details")
	}
	if m.DetailState == StateError {
		return viewError(m.DetailError)
	}
	if m.SelectedPipeline == nil {
		return viewEmpty("No pipeline selected", "Return to list and press enter on a pipeline")
	}

	p := m.SelectedPipeline
	avail := m.Width - 4

	// Overview card
	status := resolvePipelineResult(p.State.Name, func() string {
		if p.State.Result != nil {
			return p.State.Result.Name
		}
		return ""
	}())
	badge := renderStatusBadge(status)

	overview := fmt.Sprintf(
		"%s\n\n%s\n\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s\n%s %s",
		CardTitleStyle.Render(fmt.Sprintf("Pipeline #%d", p.BuildNumber)),
		badge,
		KeyStyle.Render("Branch:"), ValueStyle.Render(p.Target.RefName),
		KeyStyle.Render("Type:"), ValueStyle.Render(pipelineTypeLabel(p.Target)),
		KeyStyle.Render("Trigger:"), ValueStyle.Render(p.Trigger.Name),
		KeyStyle.Render("Duration:"), ValueStyle.Render(formatDuration(p.CreatedOn, p.CompletedOn, p.BuildSecondsUsed)),
		KeyStyle.Render("Created:"), ValueStyle.Render(formatTime(p.CreatedOn)),
		KeyStyle.Render("Completed:"), ValueStyle.Render(formatTime(p.CompletedOn)),
		KeyStyle.Render("By:"), ValueStyle.Render(creatorName(p.Creator)),
	)

	// Steps section
	stepsContent := buildStepsView(m.Steps, m.StepCursor)

	// Combine
	var sections []string
	sections = append(sections, renderPage("Overview", overview, avail))
	sections = append(sections, renderPage(fmt.Sprintf("Steps (%d)", len(m.Steps)), stepsContent, avail))

	if len(m.ConfigVars) > 0 {
		varsStr := buildVarList(m.ConfigVars, avail-4)
		sections = append(sections, renderPage(fmt.Sprintf("Pipeline Variables (%d)", len(m.ConfigVars)), varsStr, avail))
	}
	if len(m.ParsedLogVars) > 0 {
		varsStr := buildVarList(m.ParsedLogVars, avail-4)
		sections = append(sections, renderPage(fmt.Sprintf("Parsed Log Variables (%d)", len(m.ParsedLogVars)), varsStr, avail))
	}

	return strings.Join(sections, "\n\n")
}

// buildStepsView renders the steps list.
func buildStepsView(steps []bitbucket.PipelineStep, cursor int) string {
	if len(steps) == 0 {
		return DimmedStyle.Render("  No steps available")
	}

	cursor = clampCursor(cursor, len(steps))

	var sb strings.Builder
	sb.WriteString(ListHeaderStyle.Render(fmt.Sprintf("  %-3s %-30s %-16s %s", "#", "STEP", "STATUS", "DURATION")))
	sb.WriteString("\n")

	for i, s := range steps {
		prefix := "  "
		rowStyle := StepNormalStyle
		if i == cursor {
			prefix = "▶ "
			rowStyle = StepCursorStyle
		}

		status := resolveStepStatus(s.State)
		dur := formatDuration(s.StartedOn, s.CompletedOn, s.Duration)

		row := fmt.Sprintf("%s%-3d %-30s %-16s %s",
			prefix, i+1,
			truncate(s.Name, 28),
			truncate(status, 14),
			dur,
		)
		sb.WriteString(rowStyle.Render(row))
		sb.WriteString("\n")
	}
	return sb.String()
}

// buildVarList formats variables as key=value pairs.
func buildVarList(vars []bitbucket.PipelineVariable, width int) string {
	if len(vars) == 0 {
		return DimmedStyle.Render("  No variables")
	}
	var sb strings.Builder
	for _, v := range vars {
		line := fmt.Sprintf("  %s %s", KeyStyle.Render(v.Key+":"), ValueStyle.Render(v.Value))
		sb.WriteString(truncate(line, width))
		sb.WriteString("\n")
	}
	return sb.String()
}
