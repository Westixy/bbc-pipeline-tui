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

	// Header: 1 line of text + 1 line border-bottom = 2 visual lines
	// Help: 1 line of text + 1 line border-top = 2 visual lines
	// Content: remaining height
	headerLines := 2
	helpLines := 2
	contentHeight := m.Height - headerLines - helpLines
	if contentHeight < 1 {
		contentHeight = 1
	}

	header := m.viewHeader()
	content := m.viewContent(contentHeight - 3)
	help := m.viewHelp()

	// Don't force Height on ContentStyle; let it flow naturally within viewContent
	return lipgloss.JoinVertical(lipgloss.Left,
		HeaderStyle.Width(m.Width).Render(header),
		ContentStyle.Width(m.Width).MaxHeight(contentHeight).Render(content),
		HelpStyle.Width(m.Width).Render(help),
	)
}

// viewHeader renders the top bar with title and tabs.
func (m Model) viewHeader() string {
	title := "⚡ " + TitleStyle.Render("Pipeline TUI")

	var tabs []string
	for i, p := range m.Projects {
		label := fmt.Sprintf(" %d:%s/%s ", i+1, p.Workspace, p.RepoSlug)
		if i == m.ActiveProject {
			tabs = append(tabs, ActiveTabStyle.Render(label))
		} else {
			tabs = append(tabs, TabStyle.Render(label))
		}
	}
	tabsStr := strings.Join(tabs, "")

	screenTitle := DimmedStyle.Render(" · " + m.ScreenTitle())

	left := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", tabsStr)
	right := screenTitle

	// Fill the middle with space so right aligns
	used := lipgloss.Width(left) + lipgloss.Width(right)
	filler := ""
	if m.Width > used {
		filler = strings.Repeat(" ", m.Width-used)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, left, filler, right)
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
	case ScreenProjects:
		return m.viewProjects(contentHeight)
	case ScreenManageProjects:
		return m.viewManageProjects(contentHeight)
	default:
		return ""
	}
}

// viewHelp renders the keyboard shortcuts bar grouped by category.
func (m Model) viewHelp() string {
	nav := renderHelpGroup("Navigate", m.navKeys())
	act := renderHelpGroup("Actions", m.actionKeys())
	sys := renderHelpGroup("System", []string{"q:quit"})
	help := lipgloss.JoinHorizontal(lipgloss.Top, nav, "  ", act, "  ", sys)
	return HelpStyle.Render(" " + help + " ")
}

// navKeys returns navigation shortcuts for the current screen.
func (m Model) navKeys() []string {
	switch m.Screen {
	case ScreenList:
		return []string{"↑↓:move", "/:filter", "enter:detail", "n:next page"}
	case ScreenDetail:
		return []string{"↑↓:steps", "enter:trigger"}
	case ScreenLogs:
		return []string{"↑↓:scroll", "pgup/pgdn:page", "/:search"}
	case ScreenRun:
		return []string{"tab:next", "↑↓:vars", "enter:edit/submit"}
	case ScreenProjects:
		return []string{"↑↓:move", "1-9:select"}
	case ScreenManageProjects:
		return []string{"↑↓:move", "tab:pane"}
	default:
		return nil
	}
}

// actionKeys returns action shortcuts for the current screen.
func (m Model) actionKeys() []string {
	switch m.Screen {
	case ScreenList:
		return []string{"r:refresh", "p:projects"}
	case ScreenDetail:
		return []string{"l:logs", "r:refresh"}
	case ScreenLogs:
		return []string{"v:parse vars", "esc:back"}
	case ScreenRun:
		return []string{"a:add var", "d:delete var"}
	case ScreenProjects:
		return []string{"enter:confirm"}
	case ScreenManageProjects:
		return []string{"enter:select", "d:remove", "r:refresh"}
	default:
		return nil
	}
}

// renderHelpGroup formats a group of shortcuts.
func renderHelpGroup(label string, keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts, HelpKeyStyle.Render(k))
	}
	return HelpGroupStyle.Render(label+": ") + strings.Join(parts, " ")
}

// renderPage renders a bordered section with title and content.
func renderPage(title string, content string, width int) string {
	titleBar := SectionTitleStyle.Render(title)
	body := CardStyle.Width(width - 4).Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, titleBar, body)
}

// viewLoading renders a loading indicator.
func viewLoading(message string) string {
	return LoadingStyle.Render("  ⏳ " + message + "...")
}

// viewError renders an error message.
func viewError(err string) string {
	return ErrorStyle.Render("  ✗ " + err)
}

// viewSuccess renders a success message.
func viewSuccess(msg string) string {
	return SuccessStyle.Render("  ✓ " + msg)
}

// viewEmpty renders an empty state message with optional hint.
func viewEmpty(message, hint string) string {
	if hint != "" {
		message += "\n\n" + DimmedStyle.Render(hint)
	}
	return EmptyStateStyle.Render(message)
}
