package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateProjects handles messages for the project selection screen.
func (m Model) updateProjects(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			prev := m.ActiveProject - 1
			if prev < 0 {
				prev = len(m.Projects) - 1
			}
			m.ActiveProject = prev
			return m, nil

		case "down", "j":
			next := m.ActiveProject + 1
			if next >= len(m.Projects) {
				next = 0
			}
			m.ActiveProject = next
			return m, nil

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '1')
			if idx >= 0 && idx < len(m.Projects) {
				m.ActiveProject = idx
			}
			return m, nil

		case "enter", " ":
			// Switch to selected project and go back to list
			m.Screen = ScreenList
			m.ListState = StateLoading
			m.Pipelines = nil
			m.ListCursor = 0
			m.ListFilter = ""
			return m, fetchPipelines(m.Client,
				m.Projects[m.ActiveProject].Workspace,
				m.Projects[m.ActiveProject].RepoSlug)

		case "esc":
			// Cancel selection, return to list without switching
			m.Screen = ScreenList
			return m, nil
		}
	}
	return m, nil
}

// viewProjects renders the project selection overlay.
func (m Model) viewProjects(contentHeight int) string {
	if contentHeight < 4 {
		return ""
	}

	avail := m.Width - 4

	var sb strings.Builder
	sb.WriteString(CardTitleStyle.Render("Select Project"))
	sb.WriteString("\n")
	sb.WriteString(DividerStyle.Render(strings.Repeat("─", avail)))
	sb.WriteString("\n\n")

	// Calculate how many projects fit in the viewport
	nonItemLines := 4
	viewportHeight := contentHeight - nonItemLines
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	// Slice visible projects to fit viewport
	start := m.ActiveProject - viewportHeight/2
	if start < 0 {
		start = 0
	}
	end := start + viewportHeight
	if end > len(m.Projects) {
		end = len(m.Projects)
	}
	if end-start < viewportHeight && start > 0 {
		start = end - viewportHeight
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		p := m.Projects[i]

		// Build a card-like row with workspace/repo info
		label := fmt.Sprintf("[%d]  %s / %s", i+1, p.Workspace, p.RepoSlug)

		rowStyle := ListNormalStyle
		prefix := "  "
		if i == m.ActiveProject {
			rowStyle = ListCursorStyle
			prefix = "▶ "
		} else if i%2 == 1 {
			rowStyle = ListAltStyle
		}

		sb.WriteString(rowStyle.Render(prefix + label))
		sb.WriteString("\n")
	}

	return sb.String()
}
