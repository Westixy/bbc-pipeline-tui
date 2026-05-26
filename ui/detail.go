package ui

import (
	"fmt"
	"strings"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	tea "github.com/charmbracelet/bubbletea"
)

// updateDetail handles messages for the pipeline detail screen.
func (m Model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pipelineDetailLoadedMsg:
		if msg.err != nil {
			m.handleErrorMsg("detail", msg.err)
			return m, nil
		}
		m.SelectedPipeline = msg.pipeline
		m.Steps = msg.steps
		m.StepCursor = 0
		m.DetailState = StateReady
		// Also fetch repository config variables
		return m, fetchConfigVars(m.Client, m.Projects[m.ActiveProject].Workspace,
			m.Projects[m.ActiveProject].RepoSlug)

	case configVarsLoadedMsg:
		if msg.err != nil {
			// Non-fatal; show vars we already have
			m.ConfigVars = nil
		} else {
			m.ConfigVars = msg.variables
		}
		// Also fetch log-based variables from the first step
		return m, fetchLogVarsForPipeline(m.Client, m.Projects[m.ActiveProject].Workspace,
			m.Projects[m.ActiveProject].RepoSlug, m.SelectedPipeline.UUID)

	case logVarsLoadedMsg:
		if msg.err != nil {
			// Non-fatal
			m.ParsedLogVars = nil
		} else {
			m.ParsedLogVars = msg.variables
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.DetailState = StateLoading
			return m, fetchPipelineDetail(m.Client, m.Projects[m.ActiveProject].Workspace,
				m.Projects[m.ActiveProject].RepoSlug, m.SelectedPipeline.UUID)

		case "up", "k":
			if m.StepCursor > 0 {
				m.StepCursor--
			}

		case "down", "j":
			if m.StepCursor < len(m.Steps)-1 {
				m.StepCursor++
			}

		case "l":
			if len(m.Steps) > 0 {
				step := m.Steps[m.StepCursor]
				m.LogStepName = step.Name
				m.LogState = StateLoading
				m.LogContent = ""
				m.LogSearchTerm = ""
				m.LogScrollOff = 0
				m.Screen = ScreenLogs
				return m, fetchStepLog(m.Client, m.Projects[m.ActiveProject].Workspace,
					m.Projects[m.ActiveProject].RepoSlug, m.SelectedPipeline.UUID,
					step.UUID, step.Name)
			}

		case "enter":
			m.Screen = ScreenRun
			m.RunState = StateReady
			m.RunFocus = runFieldBranch
			m.RunBranch.SetValue(m.SelectedPipeline.Target.RefName)
			m.RunBranch.Focus()
			if m.SelectedPipeline.Target.Selector != nil {
				m.RunSelector.SetValue(m.SelectedPipeline.Target.Selector.Pattern)
			} else {
				m.RunSelector.SetValue("")
			}
			m.RunError = ""
			m.RunSuccessMsg = ""
			m.RunVarCursor = 0
			m.RunEditMode = false
			m.RunEditorInput.SetValue("")
			// Pre-populate variables — priority:
			// 1. Log-parsed pipeline variables (actual run variables)
			// 2. Repository config variables (non-secured)
			// 3. Pipeline-level variables from detail (non-secured)
			m.RunVars = nil
			if len(m.ParsedLogVars) > 0 {
				for _, v := range m.ParsedLogVars {
					m.RunVars = append(m.RunVars, bitbucket.PipelineVariable{
						Key:   v.Key,
						Value: v.Value,
					})
				}
			} else if len(m.ConfigVars) > 0 {
				for _, v := range m.ConfigVars {
					if !v.Secured {
						m.RunVars = append(m.RunVars, bitbucket.PipelineVariable{
							Key:   v.Key,
							Value: v.Value,
						})
					}
				}
			} else if m.SelectedPipeline != nil {
				for _, v := range m.SelectedPipeline.Variables {
					if !v.Secured {
						m.RunVars = append(m.RunVars, bitbucket.PipelineVariable{
							Key:   v.Key,
							Value: v.Value,
						})
					}
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// viewDetail renders the pipeline detail screen.
func (m Model) viewDetail(contentHeight int) string {
	switch m.DetailState {
	case StateLoading:
		return viewLoading("Loading pipeline details...")
	case StateError:
		return viewError(m.DetailError)
	}

	if m.SelectedPipeline == nil {
		return DimmedStyle.Render("No pipeline selected")
	}

	var sb strings.Builder
	p := m.SelectedPipeline
	width := m.Width - 4
	if width < 40 {
		width = 40
	}

	// Overview section
	overview := fmt.Sprintf(
		"Build #%d\n"+
			"State: %s\n"+
			"Branch: %s\n"+
			"Trigger: %s\n"+
			"Created: %s\n"+
			"Completed: %s\n"+
			"Duration: %s\n"+
			"UUID: %s",
		p.BuildNumber,
		renderStatusBadge(mergeStatus(p.State)),
		p.Target.RefName,
		p.Trigger.Name,
		formatTime(p.CreatedOn),
		formatTime(p.CompletedOn),
		formatDuration(p.CreatedOn, p.CompletedOn, p.BuildSecondsUsed),
		p.UUID,
	)
	if p.Creator != nil {
		overview += fmt.Sprintf("\nTriggered by: %s", p.Creator.DisplayName)
	}
	if p.Target.Commit != nil {
		overview += fmt.Sprintf("\nCommit: %s", truncate(p.Target.Commit.Hash, 12))
	}
	sb.WriteString(renderPage("Pipeline Overview", overview, m.Width))
	sb.WriteString("\n")

	// Pipeline run variables section
	if len(p.Variables) > 0 {
		var vars []string
		for _, v := range p.Variables {
			val := v.Value
			if v.Secured {
				val = "•••••••• (secured)"
			}
			vars = append(vars, fmt.Sprintf("  %s=%s", v.Key, val))
		}
		sb.WriteString(renderPage("Pipeline Variables", strings.Join(vars, "\n"), m.Width))
		sb.WriteString("\n")
	}

	// Parsed log variables section (actual run variables from log)
	if len(m.ParsedLogVars) > 0 {
		var vars []string
		for _, v := range m.ParsedLogVars {
			vars = append(vars, fmt.Sprintf("  %s=%s", v.Key, v.Value))
		}
		sb.WriteString(renderPage("Log Variables (actual run)", strings.Join(vars, "\n"), m.Width))
		sb.WriteString("\n")
	}

	// Repository config variables section
	if len(m.ConfigVars) > 0 {
		var vars []string
		for _, v := range m.ConfigVars {
			val := v.Value
			if v.Secured {
				val = "•••••••• (secured)"
			}
			vars = append(vars, fmt.Sprintf("  %s=%s", v.Key, val))
		}
		sb.WriteString(renderPage("Repository Config Variables", strings.Join(vars, "\n"), m.Width))
		sb.WriteString("\n")
	}

	// Steps section
	if len(m.Steps) > 0 {
		var stepLines []string
		for i, s := range m.Steps {
			state := renderStatusBadge(resolveStepStatus(s.State))
			name := s.Name
			if name == "" {
				name = fmt.Sprintf("Step %d", i+1)
			}
			image := s.Image.Name
			if image == "" {
				image = "default"
			}
			// Cursor indicator
			cursor := " "
			if i == m.StepCursor {
				cursor = ListCursorStyle.Render("▶")
			}
			// Dynamically size step line
			nameW := width - 30
			if nameW < 10 {
				nameW = 10
			}
			imageW := width - nameW - 25
			if imageW < 8 {
				imageW = 8
			}
			stepLines = append(stepLines,
				fmt.Sprintf("%s[%d] %-*s %-20s Image: %-*s", cursor, i+1, nameW, truncate(name, nameW), state, imageW, truncate(image, imageW)))
		}
		sb.WriteString(renderPage("Pipeline Steps", strings.Join(stepLines, "\n"), m.Width))
	}

	return sb.String()
}
