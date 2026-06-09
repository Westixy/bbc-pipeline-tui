package ui

import (
	"fmt"
	"strings"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Run Update ──────────────────────────────────────────────────────────────

// updateRun handles messages for the trigger pipeline form.
func (m Model) updateRun(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle edit mode for inline variable editing
		if m.RunEditMode {
			return m.handleRunEditInput(msg)
		}

		switch msg.String() {
		case "tab", "shift+tab":
			// Cycle focus: Branch → Selector → Variables
			m.RunFocus = (m.RunFocus + 1) % 3
			return m, nil

		case "enter":
			if m.RunFocus == 2 && len(m.RunVars) > 0 {
				// Edit the selected variable inline
				m.RunEditMode = true
				v := m.RunVars[clampCursor(m.RunVarCursor, len(m.RunVars))]
				m.RunEditorInput.SetValue(v.Value)
				m.RunEditorInput.Focus()
				return m, nil
			}
			// Submit the form
			m.RunState = StateLoading
			m.RunError = ""
			m.RunSuccessMsg = ""
			workspace := m.Projects[m.ActiveProject].Workspace
			repoSlug := m.Projects[m.ActiveProject].RepoSlug
			return m, triggerPipelineCmd(m.Client, workspace, repoSlug, bitbucket.TriggerPipelineRequest{
				Target: bitbucket.PipelineTarget{
					Type:    "pipeline_ref_target",
					RefType: "branch",
					RefName: m.RunBranch.Value(),
					Selector: &bitbucket.PipelineSelector{
						Type:    "custom",
						Pattern: m.RunSelector.Value(),
					},
				},
				Variables: m.RunVars,
			})

		case "up", "k":
			if m.RunFocus == 2 && m.RunVarCursor > 0 {
				m.RunVarCursor--
			}
			return m, nil

		case "down", "j":
			if m.RunFocus == 2 && m.RunVarCursor < len(m.RunVars)-1 {
				m.RunVarCursor++
			}
			return m, nil

		case "a":
			// Add a new variable
			if m.RunFocus == 2 {
				m.RunVars = append(m.RunVars, bitbucket.PipelineVariable{Key: "NEW_VAR", Value: ""})
				m.RunVarCursor = len(m.RunVars) - 1
				// Auto-enter edit mode
				m.RunEditMode = true
				m.RunEditorInput.SetValue("NEW_VAR=")
				m.RunEditorInput.Focus()
			}
			return m, nil

		case "d":
			// Delete selected variable
			if m.RunFocus == 2 && len(m.RunVars) > 0 {
				idx := clampCursor(m.RunVarCursor, len(m.RunVars))
				m.RunVars = append(m.RunVars[:idx], m.RunVars[idx+1:]...)
				if m.RunVarCursor >= len(m.RunVars) && m.RunVarCursor > 0 {
					m.RunVarCursor--
				}
			}
			return m, nil

		case "esc":
			m.Screen = ScreenDetail
			return m, nil
		}

		return m, m.RunBranch.Focus()

	case tea.WindowSizeMsg:
		return m, nil

	case pipelineTriggeredMsg:
		if msg.err != nil {
			m.RunState = StateError
			m.RunError = msg.err.Error()
		} else {
			m.RunSuccessMsg = fmt.Sprintf("Pipeline #%d triggered successfully!", msg.pipeline.BuildNumber)
			m.RunState = StateReady
		}
		return m, nil
	}

	// Forward to focused input
	var cmd tea.Cmd
	switch m.RunFocus {
	case 0:
		m.RunBranch, cmd = m.RunBranch.Update(msg)
	case 1:
		m.RunSelector, cmd = m.RunSelector.Update(msg)
	}
	return m, cmd
}

// handleRunEditInput handles key presses in variable edit mode.
func (m *Model) handleRunEditInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		input := m.RunEditorInput.Value()
		idx := clampCursor(m.RunVarCursor, len(m.RunVars))
		if m.RunVars[idx].Key == "NEW_VAR" {
			// Adding a new variable: parse key=value
			eqIdx := strings.Index(input, "=")
			if eqIdx >= 0 {
				key := strings.TrimSpace(input[:eqIdx])
				value := strings.TrimSpace(input[eqIdx+1:])
				if key != "" {
					m.RunVars[idx] = bitbucket.PipelineVariable{Key: key, Value: value}
				}
			}
		} else {
			// Editing existing variable: only update the value
			m.RunVars[idx].Value = strings.TrimSpace(input)
		}
		m.RunEditMode = false
		return m, nil

	case "esc":
		m.RunEditMode = false
		return m, nil

	default:
		m.RunEditorInput, _ = m.RunEditorInput.Update(msg)
		return m, nil
	}
}

// ── Run View ────────────────────────────────────────────────────────────────

// viewRun renders the trigger pipeline form with branch, selector, and variables.
func (m Model) viewRun() string {
	if m.RunState == StateLoading {
		return viewLoading("Triggering pipeline")
	}
	if m.RunSuccessMsg != "" {
		return viewSuccess(m.RunSuccessMsg)
	}
	if m.RunState == StateError {
		return viewError(m.RunError)
	}

	avail := m.Width - 4

	// ── Branch Input ──────────────────────────────────────────────────────
	m.RunBranch.Placeholder = "e.g. main"
	if m.RunFocus == 0 {
		m.RunBranch.Focus()
	} else {
		m.RunBranch.Blur()
	}

	// ── Selector Input ────────────────────────────────────────────────────
	m.RunSelector.Placeholder = "optional pattern"
	if m.RunFocus == 1 {
		m.RunSelector.Focus()
	} else {
		m.RunSelector.Blur()
	}

	// Style the inputs based on focus
	branchStyle := InputStyle
	if m.RunFocus == 0 {
		branchStyle = FocusedInputStyle
	}
	selStyle := InputStyle
	if m.RunFocus == 1 {
		selStyle = FocusedInputStyle
	}

	branchLabel := RequiredMarkerStyle.Render("*") + " Branch"
	selLabel := "  Selector"

	inputWidth := avail - 18
	if inputWidth < 20 {
		inputWidth = 20
	}

	branchInput := lipgloss.JoinHorizontal(lipgloss.Top,
		KeyStyle.Render(branchLabel+" "),
		branchStyle.Width(inputWidth).Render(m.RunBranch.View()),
	)

	selInput := lipgloss.JoinHorizontal(lipgloss.Top,
		KeyStyle.Render(selLabel+" "),
		selStyle.Width(inputWidth).Render(m.RunSelector.View()),
	)

	// ── Variables Section ─────────────────────────────────────────────────
	varCount := BadgeStyle.Render(fmt.Sprintf(" %d ", len(m.RunVars)))
	variablesContent := m.buildRunVariablesSection()

	// ── Edit mode overlay ─────────────────────────────────────────────────
	if m.RunEditMode {
		m.RunEditorInput.Focus()
		idx := clampCursor(m.RunVarCursor, len(m.RunVars))
		var editPrompt string
		if idx < len(m.RunVars) && m.RunVars[idx].Key == "NEW_VAR" {
			editPrompt = CardTitleStyle.Render("New Variable (key=value):") + "  " +
				FocusedInputStyle.Width(50).Render(m.RunEditorInput.View())
		} else {
			editPrompt = CardTitleStyle.Render("Edit Value:") + "  " +
				FocusedInputStyle.Width(50).Render(m.RunEditorInput.View())
		}
		editHint := DimmedStyle.Render("  enter to save  │  esc to cancel")

		full := lipgloss.JoinVertical(lipgloss.Left,
			branchInput, "", selInput, "",
			CardTitleStyle.Render("Variables")+" "+varCount,
			variablesContent,
			"",
			DividerStyle.Render(strings.Repeat("─", avail)),
			editPrompt,
			editHint,
		)
		return full
	}

	// ── Footer help ───────────────────────────────────────────────────────
	footer := DimmedStyle.Render(
		fmt.Sprintf("tab to focus  │  a:add var  │  d:delete var  │  enter to %s",
			func() string {
				if m.RunFocus == 2 && len(m.RunVars) > 0 {
					return "edit var"
				}
				return "trigger"
			}()),
	)

	full := lipgloss.JoinVertical(lipgloss.Left,
		branchInput,
		"",
		selInput,
		"",
		CardTitleStyle.Render("Variables")+" "+varCount,
		variablesContent,
		"",
		DividerStyle.Render(strings.Repeat("─", avail)),
		footer,
	)
	return full
}

// buildRunVariablesSection renders the variable list for the run form.
func (m Model) buildRunVariablesSection() string {
	if len(m.RunVars) == 0 {
		return DimmedStyle.Render("  No variables yet — press 'a' to add one")
	}

	cursor := clampCursor(m.RunVarCursor, len(m.RunVars))
	isFocused := m.RunFocus == 2

	var sb strings.Builder
	for i, v := range m.RunVars {
		prefix := "  "
		style := ValueStyle
		if isFocused && i == cursor {
			prefix = "▶ "
			style = StepCursorStyle
		}

		// Truncate long values safely (handles multi-byte UTF-8)
		val := truncate(v.Value, 48)

		line := fmt.Sprintf("%s%s %s", prefix,
			KeyStyle.Render(v.Key+":"),
			style.Render(val),
		)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}
