package ui

import (
	"fmt"
	"strings"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	tea "github.com/charmbracelet/bubbletea"
)

// updateLogs handles messages for the log viewer screen.
func (m Model) updateLogs(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case stepLogLoadedMsg:
		if msg.err != nil {
			m.handleErrorMsg("log", msg.err)
			return m, nil
		}
		m.LogContent = msg.content
		m.LogStepName = msg.stepName
		m.LogState = StateReady
		m.LogScrollOff = 0
		m.LogSearchTerm = ""
		m.LogSearchMode = false
		m.LogMatchLines = nil
		m.LogMatchIndex = -1
		return m, nil

	case tea.KeyMsg:
		logLines := strings.Split(m.LogContent, "\n")

		// When search mode is active, capture text input before routing to
		// navigation keys.  This keeps the "/" key out of the search term.
		if m.LogSearchMode {
			switch msg.String() {
			case "esc":
				m.LogSearchMode = false
				return m, nil
			case "enter":
				m.LogSearchTerm = ""
				m.LogSearchMode = false
				return m, nil
			case "backspace":
				if len(m.LogSearchTerm) > 0 {
					m.LogSearchTerm = m.LogSearchTerm[:len(m.LogSearchTerm)-1]
					m.updateMatchLines()
				}
				return m, nil
			}
			// Capture printable runes (letters, digits, symbols)
			if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
				m.LogSearchTerm += string(msg.Runes)
				m.updateMatchLines()
				return m, nil
			}
			// For any other key while searching, fall through to navigation.
		}

		switch msg.String() {
		case "r":
			return m, nil

		case "/":
			m.LogSearchMode = true
			return m, nil

		case "up", "k":
			if m.LogScrollOff > 0 {
				m.LogScrollOff--
			}

		case "down", "j":
			maxScroll := len(logLines) - 1
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.LogScrollOff < maxScroll {
				m.LogScrollOff++
			}

		case "pgup", "u":
			// Page up: scroll up by half the visible area (conservative default)
			pageSize := (m.Height - 9) / 2
			if pageSize < 1 {
				pageSize = 1
			}
			m.LogScrollOff -= pageSize
			if m.LogScrollOff < 0 {
				m.LogScrollOff = 0
			}

		case "pgdown", "d", " ":
			// Page down: scroll down by half the visible area
			pageSize := (m.Height - 9) / 2
			if pageSize < 1 {
				pageSize = 1
			}
			maxScroll := len(logLines) - pageSize
			if maxScroll < 0 {
				maxScroll = 0
			}
			m.LogScrollOff += pageSize
			if m.LogScrollOff > maxScroll {
				m.LogScrollOff = maxScroll
			}

		case "n":
			// Next match
			m.nextMatch()
			return m, nil

		case "N":
			// Previous match
			m.prevMatch()
			return m, nil

		case "v":
			// Parse pipeline variables from log and navigate to run form
			vars := ParsePipelineVariablesFromLog(m.LogContent)
			m.Screen = ScreenRun
			m.RunState = StateReady
			m.RunFocus = runFieldBranch
			m.RunBranch.SetValue("")
			m.RunBranch.Focus()
			m.RunError = ""
			m.RunSuccessMsg = ""
			m.RunVarCursor = 0
			m.RunEditMode = false
			m.RunEditorInput.SetValue("")
			m.RunVars = nil
			if len(vars) > 0 {
				// Pre-populate variables parsed from log
				for _, v := range vars {
					m.RunVars = append(m.RunVars, bitbucket.PipelineVariable{
						Key:   v.Key,
						Value: v.Value,
					})
				}
			}
			// Also pre-populate branch and selector if pipeline is selected
			if m.SelectedPipeline != nil {
				m.RunBranch.SetValue(m.SelectedPipeline.Target.RefName)
				if m.SelectedPipeline.Target.Selector != nil {
					m.RunSelector.SetValue(m.SelectedPipeline.Target.Selector.Pattern)
				} else {
					m.RunSelector.SetValue("")
				}
			}
			return m, nil
		}
	}

	return m, nil
}

// viewLogs renders the step log viewer screen, filling the available height.
func (m Model) viewLogs(contentHeight int) string {
	switch m.LogState {
	case StateLoading:
		return viewLoading("Loading log for step: " + m.LogStepName + "...")
	case StateError:
		return viewError(m.LogError)
	}

	var sb strings.Builder

	// Search bar
	if m.LogSearchTerm != "" {
		sb.WriteString(FilterStyle.Render(fmt.Sprintf("Search: %s", m.LogSearchTerm)))
		sb.WriteString("\n")
	} else {
		sb.WriteString(HelpStyle.Render("Press / to search, enter to clear search"))
		sb.WriteString("\n")
	}

	// Match info / status bar at bottom
	if m.LogSearchTerm != "" && len(m.LogMatchLines) > 0 {
		status := fmt.Sprintf("Match %d/%d", m.LogMatchIndex+1, len(m.LogMatchLines))
		sb.WriteString(HelpStyle.Render(status))
	} else if m.LogSearchTerm != "" {
		sb.WriteString(HelpStyle.Render("No matches"))
	}
	sb.WriteString("\n")

	// Log content viewport
	viewportHeight := contentHeight - 2 // subtract search bar and status line
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	if m.LogContent == "" {
		sb.WriteString(DimmedStyle.Render("No log output available"))
	} else {
		logLines := strings.Split(m.LogContent, "\n")

		// Clamp scroll offset
		maxScroll := len(logLines) - viewportHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.LogScrollOff > maxScroll {
			m.LogScrollOff = maxScroll
		}
		if m.LogScrollOff < 0 {
			m.LogScrollOff = 0
		}

		start := m.LogScrollOff
		end := start + viewportHeight
		if end > len(logLines) {
			end = len(logLines)
		}

		for i := start; i < end; i++ {
			line := logLines[i]
			if m.LogSearchTerm != "" {
				line = highlightText(line, m.LogSearchTerm)
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// highlightText highlights occurrences of term in text.
func highlightText(text, term string) string {
	if term == "" {
		return text
	}
	lower := strings.ToLower(text)
	termLower := strings.ToLower(term)
	var result strings.Builder
	lastIdx := 0
	for {
		idx := strings.Index(lower[lastIdx:], termLower)
		if idx == -1 {
			result.WriteString(text[lastIdx:])
			break
		}
		absIdx := lastIdx + idx
		result.WriteString(text[lastIdx:absIdx])
		result.WriteString(LogHighlightStyle.Render(text[absIdx : absIdx+len(term)]))
		lastIdx = absIdx + len(term)
	}
	return result.String()
}

// updateMatchLines finds all line numbers containing the search term and sets
// the match index to the first match that is on or after the current scroll
// position (or 0 if none found).
func (m *Model) updateMatchLines() {
	m.LogMatchLines = nil
	m.LogMatchIndex = -1
	if m.LogSearchTerm == "" {
		return
	}
	lines := strings.Split(m.LogContent, "\n")
	termLower := strings.ToLower(m.LogSearchTerm)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), termLower) {
			m.LogMatchLines = append(m.LogMatchLines, i)
		}
	}
	// Position cursor at the first match on or after the current scroll offset
	if len(m.LogMatchLines) == 0 {
		return
	}
	bestIdx := 0
	for i, lineNo := range m.LogMatchLines {
		if lineNo >= m.LogScrollOff {
			bestIdx = i
			break
		}
		bestIdx = i // keep the last before scroll offset
	}
	m.LogMatchIndex = bestIdx
	// Scroll so that the selected match is visible
	if m.LogMatchIndex >= 0 && m.LogMatchIndex < len(m.LogMatchLines) {
		m.LogScrollOff = m.LogMatchLines[m.LogMatchIndex]
	}
}

// nextMatch moves to the next search match.
func (m *Model) nextMatch() {
	if len(m.LogMatchLines) == 0 {
		return
	}
	m.LogMatchIndex++
	if m.LogMatchIndex >= len(m.LogMatchLines) {
		m.LogMatchIndex = 0
	}
	m.LogScrollOff = m.LogMatchLines[m.LogMatchIndex]
}

// prevMatch moves to the previous search match.
func (m *Model) prevMatch() {
	if len(m.LogMatchLines) == 0 {
		return
	}
	m.LogMatchIndex--
	if m.LogMatchIndex < 0 {
		m.LogMatchIndex = len(m.LogMatchLines) - 1
	}
	m.LogScrollOff = m.LogMatchLines[m.LogMatchIndex]
}
