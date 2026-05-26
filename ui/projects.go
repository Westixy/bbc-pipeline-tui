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
			idx := int(msg.String()[0] - '1') // 0-indexed
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

// viewProjects renders the project selection overlay/modal.
func (m Model) viewProjects(contentHeight int) string {
	if contentHeight < 4 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(DetailSectionStyle.Render("Select Project"))
	sb.WriteString("\n\n")

	for i, p := range m.Projects {
		label := fmt.Sprintf("[%d] %s/%s", i+1, p.Workspace, p.RepoSlug)
		if i == m.ActiveProject {
			sb.WriteString(ListCursorStyle.Render("▶ " + label))
		} else {
			sb.WriteString("  " + label)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(DimmedStyle.Render(strings.Repeat("─", 40)))
	sb.WriteString("\n")
	sb.WriteString(HelpStyle.Render("↑/↓ navigate │ 1-9 direct │ enter select │ esc cancel"))

	return sb.String()
}