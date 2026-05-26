package ui

import (
	"fmt"
	"strings"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	tea "github.com/charmbracelet/bubbletea"
)

// runFieldBranch is the special RunFocus value for the branch field.
const runFieldBranch = -1

// runFieldSelector is the special RunFocus value for the selector field.
const runFieldSelector = -2

// updateRun handles messages for the trigger pipeline form screen.
func (m Model) updateRun(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pipelineTriggeredMsg:
		if msg.err != nil {
			m.handleErrorMsg("run", msg.err)
			return m, nil
		}
		m.RunSuccessMsg = fmt.Sprintf("Pipeline #%d triggered successfully!", msg.pipeline.BuildNumber)
		m.RunState = StateReady
		return m, nil

	case tea.KeyMsg:
		// Global keyboard shortcuts (always active)
		switch msg.String() {
		case "ctrl+s":
			return m.submitRun()

		case "esc":
			// Cancel editing if in edit mode, otherwise go back to detail
			if m.RunEditMode {
				m.RunEditMode = false
				m.RunEditorInput.Blur()
				return m, nil
			}
			m.Screen = ScreenDetail
			return m, nil
		}

		// If editing a variable value inline, route to editor handler
		if m.RunEditMode {
			return m.updateVarEditor(msg)
		}

		// Normal (non-edit) key handling
		switch msg.String() {
		case "tab":
			m.runNextField()
			return m, nil

		case "shift+tab":
			m.runPrevField()
			return m, nil

		case "enter":
			if m.RunFocus == runFieldBranch {
				// Enter on Branch → submit
				return m.submitRun()
			}
			// Enter on a variable → edit its value
			if len(m.RunVars) > 0 && m.RunFocus >= 0 && m.RunFocus < len(m.RunVars) {
				m.RunEditMode = true
				m.RunEditorInput.SetValue(m.RunVars[m.RunFocus].Value)
				m.RunEditorInput.CursorEnd()
				m.RunEditorInput.Focus()
			}
			return m, nil

		case "a":
			// Add a new variable and switch to editing it
			newVar := bitbucket.PipelineVariable{Key: "new_var", Value: ""}
			m.RunVars = append(m.RunVars, newVar)
			m.RunFocus = len(m.RunVars) - 1
			m.RunEditMode = true
			m.RunEditorInput.SetValue("")
			m.RunEditorInput.Focus()
			return m, nil

		case "d":
			// Delete the currently selected variable
			if m.RunFocus >= 0 && m.RunFocus < len(m.RunVars) {
				m.RunVars = append(m.RunVars[:m.RunFocus], m.RunVars[m.RunFocus+1:]...)
				if m.RunFocus >= len(m.RunVars) {
					// If we deleted the last variable and it was the only one,
					// move focus to branch
					if len(m.RunVars) == 0 {
						m.RunFocus = runFieldBranch
					} else {
						m.RunFocus = len(m.RunVars) - 1
					}
				}
			}
			return m, nil

		case "up", "k":
			if !m.RunEditMode {
				m.RunFocus = clampCursor(m.RunFocus-1, len(m.RunVars))
			}
			return m, nil

		case "down", "j":
			if !m.RunEditMode {
				m.RunFocus = clampCursor(m.RunFocus+1, len(m.RunVars))
			}
			return m, nil
		}
	}

	// Route non-key messages to branch text input
	return m, m.updateRunInputs(msg)
}

// runNextField advances focus to the next field (wrap-around).
func (m *Model) runNextField() {
	if m.RunEditMode {
		m.RunEditMode = false
		m.RunEditorInput.Blur()
	}
	m.RunBranch.Blur()
	m.RunSelector.Blur()

	switch m.RunFocus {
	case runFieldBranch:
		m.RunFocus = runFieldSelector
	case runFieldSelector:
		if len(m.RunVars) > 0 {
			m.RunFocus = 0
		} else {
			m.RunFocus = runFieldBranch
		}
	default:
		next := m.RunFocus + 1
		if next >= len(m.RunVars) {
			m.RunFocus = runFieldBranch
		} else {
			m.RunFocus = next
		}
	}
}

// runPrevField moves focus to the previous field (wrap-around).
func (m *Model) runPrevField() {
	if m.RunEditMode {
		m.RunEditMode = false
		m.RunEditorInput.Blur()
	}
	m.RunBranch.Blur()
	m.RunSelector.Blur()

	switch m.RunFocus {
	case runFieldBranch:
		if len(m.RunVars) > 0 {
			m.RunFocus = len(m.RunVars) - 1
		} else {
			m.RunFocus = runFieldSelector
		}
	case runFieldSelector:
		m.RunFocus = runFieldBranch
	default:
		if m.RunFocus <= 0 {
			m.RunFocus = runFieldSelector
		} else {
			m.RunFocus--
		}
	}
}

// updateVarEditor handles keys while editing a variable's value.
func (m Model) updateVarEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.RunFocus < 0 || m.RunFocus >= len(m.RunVars) {
		m.RunEditMode = false
		m.RunEditorInput.Blur()
		return m, nil
	}

	switch msg.String() {
	case "enter":
		// Commit the value and exit edit mode
		m.RunVars[m.RunFocus].Value = strings.TrimSpace(m.RunEditorInput.Value())
		m.RunEditMode = false
		m.RunEditorInput.Blur()
		return m, nil

	case "esc":
		// Cancel editing
		m.RunEditMode = false
		m.RunEditorInput.Blur()
		return m, nil
	}

	// Pass other keys to the editor input
	var cmd tea.Cmd
	m.RunEditorInput, cmd = m.RunEditorInput.Update(msg)
	return m, cmd
}

// updateRunInputs routes non-key messages to the branch or selector text input.
func (m *Model) updateRunInputs(msg tea.Msg) tea.Cmd {
	if m.RunEditMode {
		return nil
	}
	var cmd tea.Cmd
	switch m.RunFocus {
	case runFieldBranch:
		m.RunBranch, cmd = m.RunBranch.Update(msg)
	case runFieldSelector:
		m.RunSelector, cmd = m.RunSelector.Update(msg)
	}
	return cmd
}

// submitRun builds the trigger request and dispatches it.
func (m Model) submitRun() (tea.Model, tea.Cmd) {
	branch := strings.TrimSpace(m.RunBranch.Value())
	if branch == "" {
		m.RunError = "Branch is required"
		return m, nil
	}

	// Filter out empty-key variables
	var vars []bitbucket.PipelineVariable
	for _, v := range m.RunVars {
		if strings.TrimSpace(v.Key) != "" {
			vars = append(vars, bitbucket.PipelineVariable{Key: v.Key, Value: v.Value})
		}
	}

	target := bitbucket.PipelineTarget{
		Type:    "pipeline_ref_target",
		RefType: "branch",
		RefName: branch,
	}
	if selPattern := strings.TrimSpace(m.RunSelector.Value()); selPattern != "" {
		target.Selector = &bitbucket.PipelineSelector{
			Type:    "custom",
			Pattern: selPattern,
		}
	}
	req := bitbucket.TriggerPipelineRequest{
		Target:    target,
		Variables: vars,
	}

	m.RunState = StateLoading
	m.RunError = ""
	m.RunSuccessMsg = ""
	return m, triggerPipelineCmd(m.Client, m.Projects[m.ActiveProject].Workspace,
		m.Projects[m.ActiveProject].RepoSlug, req)
}

// viewRun renders the trigger pipeline form (clean, borderless layout).
func (m Model) viewRun() string {
	var sb strings.Builder

	// Success / error messages
	if m.RunSuccessMsg != "" {
		sb.WriteString(SuccessStyle.Render("✓ " + m.RunSuccessMsg))
		sb.WriteString("\n\n")
	}
	if m.RunError != "" {
		sb.WriteString(ErrorStyle.Render("✗ " + m.RunError))
		sb.WriteString("\n\n")
	}

	// Title
	sb.WriteString(DetailSectionStyle.Render("Trigger Job:"))
	sb.WriteString("\n\n")

	// ── Branch field ──
	focusedBranch := m.RunFocus == runFieldBranch
	branchLabel := DetailKeyStyle.Render("Branch:")
	if focusedBranch {
		sb.WriteString(ListCursorStyle.Render("▶ "))
		sb.WriteString(branchLabel)
		sb.WriteString(" ")
		sb.WriteString(FocusedInputStyle.Render(m.RunBranch.View()))
	} else {
		sb.WriteString("  ")
		sb.WriteString(branchLabel)
		sb.WriteString(" ")
		sb.WriteString(DimmedStyle.Render(m.RunBranch.Value()))
	}
	sb.WriteString("\n")

	// ── Selector field ──
	focusedSel := m.RunFocus == runFieldSelector
	selLabel := DetailKeyStyle.Render("Selector:")
	if focusedSel {
		sb.WriteString(ListCursorStyle.Render("▶ "))
		sb.WriteString(selLabel)
		sb.WriteString(" ")
		sb.WriteString(FocusedInputStyle.Render(m.RunSelector.View()))
	} else {
		sb.WriteString("  ")
		sb.WriteString(selLabel)
		sb.WriteString(" ")
		sb.WriteString(DimmedStyle.Render(m.RunSelector.Value()))
	}
	sb.WriteString("\n\n")

	// ── Variables section ──
	sb.WriteString(DetailKeyStyle.Render("Variables:"))
	sb.WriteString("\n")

	if len(m.RunVars) == 0 {
		sb.WriteString(DimmedStyle.Render("  No variables. Press 'a' to add one."))
		sb.WriteString("\n")
	} else {
		for i, v := range m.RunVars {
			isEditing := m.RunEditMode && m.RunFocus == i
			isSelected := !m.RunEditMode && m.RunFocus == i

			// Cursor indicator
			var cursor string
			if isSelected {
				cursor = ListCursorStyle.Render("▶")
			} else {
				cursor = " "
			}

			// Key rendering
			keyStyle := DimmedStyle
			if isSelected {
				keyStyle = ListCursorStyle
			}

			if isEditing {
				sb.WriteString(fmt.Sprintf("%s %s %s: %s\n",
					cursor,
					DimmedStyle.Render("-"),
					keyStyle.Render(v.Key),
					FocusedInputStyle.Render(m.RunEditorInput.View())))
			} else {
				sb.WriteString(fmt.Sprintf("%s %s %s: %s\n",
					cursor,
					DimmedStyle.Render("-"),
					keyStyle.Render(v.Key),
					keyStyle.Render(v.Value)))
			}
		}
	}
	sb.WriteString("\n")

	// ── Footer help ──
	if m.RunEditMode {
		sb.WriteString(HelpStyle.Render("Editing value │ Enter: confirm │ Esc: cancel"))
	} else {
		sb.WriteString(HelpStyle.Render("tab: next field │ enter: submit │ a: add var │ d: delete var │ esc: back"))
	}

	return sb.String()
}
