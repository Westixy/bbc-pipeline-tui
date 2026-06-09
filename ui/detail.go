package ui

import (
	"fmt"
	"os/exec"
	"runtime"
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

		case "o":
			// Open pipeline in web browser
			m.DetailMessage = ""
			if m.SelectedPipeline != nil {
				proj := m.Projects[m.ActiveProject]
				url := fmt.Sprintf("https://bitbucket.org/%s/%s/pipelines/results/%d",
					proj.Workspace, proj.RepoSlug, m.SelectedPipeline.BuildNumber)
				if err := openBrowser(url); err != nil {
					m.DetailMessage = "URL: " + url
				} else {
					m.DetailMessage = "Opened: " + url
				}
			}
			return m, nil

		case "r":
			m.Client.InvalidateCacheForRepo(m.Projects[m.ActiveProject].Workspace, m.Projects[m.ActiveProject].RepoSlug)
			m.DetailState = StateLoading
			m.Steps = nil
			m.StepCursor = 0
			m.DetailMessage = ""
			workspace := m.Projects[m.ActiveProject].Workspace
			repoSlug := m.Projects[m.ActiveProject].RepoSlug
			return m, tea.Batch(
				fetchPipelineDetail(m.Client, workspace, repoSlug, m.SelectedPipeline.UUID),
				fetchConfigVars(m.Client, workspace, repoSlug),
			)

		case "pgup":
			m.DetailScrollOff -= 5
			if m.DetailScrollOff < 0 {
				m.DetailScrollOff = 0
			}
			return m, nil

		case "pgdown":
			m.DetailScrollOff += 5
			if maxScroll := m.maxDetailScroll(); m.DetailScrollOff > maxScroll {
				m.DetailScrollOff = maxScroll
			}
			return m, nil

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
	stepsContent := buildStepsViewWithScroll(m.Steps, m.StepCursor, m.DetailScrollOff)

	// Combine sections
	var sections []string
	sections = append(sections, renderPage("Overview", overview, avail))
	sections = append(sections, renderPage(fmt.Sprintf("Steps (%d)", len(m.Steps)), stepsContent, avail))

	if m.DetailMessage != "" {
		sections = append(sections, InfoStyle.Render("  "+m.DetailMessage))
	}

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

// maxDetailScroll returns the maximum scroll offset for the detail steps view.
func (m Model) maxDetailScroll() int {
	if len(m.Steps) <= 1 {
		return 0
	}
	maxScroll := len(m.Steps) - 1
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

// buildStepsView renders the steps list (backwards-compatible stub).
func buildStepsView(steps []bitbucket.PipelineStep, cursor int) string {
	return buildStepsViewWithScroll(steps, cursor, 0)
}

// buildStepsViewWithScroll renders steps with scroll offset and viewport height.
func buildStepsViewWithScroll(steps []bitbucket.PipelineStep, cursor, scrollOff int) string {
	if len(steps) == 0 {
		return DimmedStyle.Render("  No steps available")
	}

	cursor = clampCursor(cursor, len(steps))

	// Calculate viewport
	viewportH := 10 // default steps viewport
	if len(steps) < viewportH {
		viewportH = len(steps)
	}

	// Auto-center on cursor if scrollOff would hide it
	start := scrollOff
	end := start + viewportH
	if end > len(steps) {
		end = len(steps)
	}
	if cursor < start || cursor >= end {
		start = cursor - viewportH/2
		if start < 0 {
			start = 0
		}
		end = start + viewportH
		if end > len(steps) {
			end = len(steps)
			start = end - viewportH
			if start < 0 {
				start = 0
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(ListHeaderStyle.Render(fmt.Sprintf("  %-3s %-30s %-16s %s", "#", "STEP", "STATUS", "DURATION")))
	sb.WriteString("\n")

	// Show "..." indicator if steps are hidden above
	if start > 0 {
		sb.WriteString(DimmedStyle.Render(fmt.Sprintf("  ... %d more steps above", start)))
		sb.WriteString("\n")
	}

	for i := start; i < end; i++ {
		s := steps[i]
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

	// Show "..." indicator if steps are hidden below
	if end < len(steps) {
		sb.WriteString(DimmedStyle.Render(fmt.Sprintf("  ... %d more steps below", len(steps)-end)))
		sb.WriteString("\n")
	}

	return sb.String()
}

// openBrowser opens the given URL in the default web browser.
// Returns an error if no browser command is available.
func openBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
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
