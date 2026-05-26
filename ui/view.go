package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the full TUI based on the current screen, filling the terminal.
func (m Model) View() string {
	if !m.Ready {
		return "Initializing..."
	}

	// Layout:
	//  - Header: 1 line
	//  - Content: remaining height minus 1 for help
	//  - Help: 1 line
	contentHeight := m.Height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	header := m.viewHeader()
	content := m.viewContent(contentHeight)
	help := m.viewHelp()

	return lipgloss.JoinVertical(lipgloss.Left,
		HeaderStyle.Width(m.Width).Render(header),
		ContentStyle.Width(m.Width).Height(contentHeight).Render(content),
		HelpStyle.Width(m.Width).Render(help),
	)
}

// viewHeader renders the top bar with title and tabs.
func (m Model) viewHeader() string {
	title := TitleStyle.Render("Bitbucket Pipeline TUI")

	var tabs []string
	for i, p := range m.Projects {
		label := fmt.Sprintf("[%d] %s/%s", i+1, p.Workspace, p.RepoSlug)
		if i == m.ActiveProject {
			tabs = append(tabs, ActiveTabStyle.Render(label))
		} else {
			tabs = append(tabs, TabStyle.Render(label))
		}
	}
	tabsStr := strings.Join(tabs, " ")

	screenTitle := m.ScreenTitle()
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", tabsStr, "  ", DimmedStyle.Render(screenTitle))
	return header
}

// viewContent routes to the appropriate screen view, passing available height.
func (m Model) viewContent(contentHeight int) string {
	switch m.Screen {
	case ScreenList:
		return m.viewList(contentHeight)
	case ScreenDetail:
		return m.viewDetail(contentHeight)
	case ScreenLogs:
		return m.viewLogs(contentHeight)
	case ScreenRun:
		return m.viewRun()
	default:
		return ""
	}
}

// viewHelp renders the keyboard shortcuts bar.
func (m Model) viewHelp() string {
	help := HelpStyle
	var items []string

	switch m.Screen {
	case ScreenList:
		items = []string{
			"↑/↓ or j/k: navigate",
			"/: filter",
			"enter: details",
			"r: refresh",
			"n: next page",
			"1-9: select project",
			"[/]: prev/next project",
			"esc: back",
			"q: quit",
		}
	case ScreenDetail:
		items = []string{
			"enter: trigger pipeline",
			"l: view step logs",
			"r: refresh",
			"esc: back",
			"q: quit",
		}
	case ScreenLogs:
		items = []string{
			"↑/↓ or j/k: scroll",
			"/: search",
			"v: parse vars & trigger",
			"esc: back",
			"q: quit",
		}
	case ScreenRun:
		items = []string{
			"tab: next field",
			"enter: submit form",
			"esc: back",
			"q: quit",
		}
	}

	return help.Render(strings.Join(items, " │ "))
}

// renderPage renders a bordered page with title and content.
func renderPage(title string, content string, width int) string {
	titleBar := DetailSectionStyle.Render(title)
	body := PanelStyle.Width(width - 4).Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, titleBar, body)
}

// viewLoading renders a loading indicator.
func viewLoading(message string) string {
	return LoadingStyle.Render("⏳ " + message)
}

// viewError renders an error message.
func viewError(err string) string {
	return ErrorStyle.Render("❌ Error: " + err)
}