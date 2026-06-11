package ui

import (
	"fmt"
	"strings"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

// updateManageProjects handles messages for the manage projects screen.
func (m Model) updateManageProjects(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If workspace input is focused, let the textinput handle it
		if m.ManageFocus == 2 {
			switch msg.String() {
			case "enter":
				// Fetch repos for the entered workspace
				ws := strings.TrimSpace(m.ManageWorkspaceInput.Value())
				if ws != "" {
					m.ManageFocus = 1
					m.ManageWorkspaceInput.Blur()
					m.WorkspaceReposState = StateLoading
					m.WorkspaceRepos = nil
					m.WorkspaceReposError = ""
					m.ManageRepoCursor = 0
					m.ManageRepoScrollOff = 0
					return m, fetchWorkspaceRepos(m.Client, ws)
				}
				return m, nil
			case "esc":
				// Move focus from input back to repo list
				m.ManageFocus = 1
				m.ManageWorkspaceInput.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.ManageWorkspaceInput, cmd = m.ManageWorkspaceInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "tab":
			// Cycle focus: favorites(0) -> repos(1) -> workspace input(2) -> favorites(0)
			m.ManageWorkspaceInput.Blur()
			if m.ManageFocus == 0 {
				m.ManageFocus = 1
			} else if m.ManageFocus == 1 {
				m.ManageFocus = 2
				m.ManageWorkspaceInput.Focus()
				m.ManageWorkspaceInput.SetValue(m.Projects[m.ActiveProject].Workspace)
			}
			return m, nil

		case "up", "k":
			if m.ManageFocus == 0 && len(m.Projects) > 0 {
				m.ManageFavCursor--
				if m.ManageFavCursor < 0 {
					m.ManageFavCursor = len(m.Projects) - 1
				}
				m.scrollFavoritesIntoView()
			} else if m.ManageFocus == 1 && len(m.WorkspaceRepos) > 0 {
				m.ManageRepoCursor--
				if m.ManageRepoCursor < 0 {
					m.ManageRepoCursor = len(m.WorkspaceRepos) - 1
				}
				m.scrollReposIntoView()
			}
			return m, nil

		case "down", "j":
			if m.ManageFocus == 0 && len(m.Projects) > 0 {
				m.ManageFavCursor++
				if m.ManageFavCursor >= len(m.Projects) {
					m.ManageFavCursor = 0
				}
				m.scrollFavoritesIntoView()
			} else if m.ManageFocus == 1 && len(m.WorkspaceRepos) > 0 {
				m.ManageRepoCursor++
				if m.ManageRepoCursor >= len(m.WorkspaceRepos) {
					m.ManageRepoCursor = 0
				}
				m.scrollReposIntoView()
			}
			return m, nil

		case "enter", " ":
			if m.ManageFocus == 0 && len(m.Projects) > 0 {
				// Select favorite as active project and return to list
				m.ActiveProject = m.ManageFavCursor
				m.Screen = ScreenList
				m.ListState = StateLoading
				m.Pipelines = nil
				m.ListCursor = 0
				m.ListFilter = ""
				return m, fetchPipelines(m.Client,
					m.Projects[m.ActiveProject].Workspace,
					m.Projects[m.ActiveProject].RepoSlug)
			} else if m.ManageFocus == 1 && len(m.WorkspaceRepos) > 0 {
				// Add repo to favorites
				repo := m.WorkspaceRepos[m.ManageRepoCursor]
				parts := strings.SplitN(repo.FullName, "/", 2)
				if len(parts) == 2 {
					if m.Config.AddProject(parts[0], parts[1]) {
						m.Projects = m.Config.Projects
						_ = m.Config.Save()
					}
				}
			}
			return m, nil

		case "d":
			if m.ManageFocus == 0 && len(m.Projects) > 1 {
				// Remove from favorites
				p := m.Projects[m.ManageFavCursor]
				if m.Config.RemoveProject(p.Workspace, p.RepoSlug) {
					m.Projects = m.Config.Projects
					_ = m.Config.Save()
					if m.ManageFavCursor >= len(m.Projects) {
						m.ManageFavCursor = len(m.Projects) - 1
					}
					if m.ActiveProject >= len(m.Projects) {
						m.ActiveProject = len(m.Projects) - 1
					}
					m.scrollFavoritesIntoView()
				}
			}
			return m, nil

		case "r":
			// Refresh workspace repos
			if m.ManageFocus == 1 || m.ManageFocus == 0 {
				ws := strings.TrimSpace(m.ManageWorkspaceInput.Value())
				if ws == "" {
					ws = m.Projects[m.ActiveProject].Workspace
				}
				if ws != "" {
					m.Client.InvalidateCacheForWorkspace(ws)
					m.WorkspaceReposState = StateLoading
					m.WorkspaceRepos = nil
					m.WorkspaceReposError = ""
					m.ManageRepoCursor = 0
					m.ManageRepoScrollOff = 0
					return m, fetchWorkspaceRepos(m.Client, ws)
				}
			}
			return m, nil

		case "esc":
			// Step back: input(2) -> repos(1) -> favs(0), already handled above
			// At pane 0, esc leaves the screen
			if m.ManageFocus == 0 {
				m.Screen = ScreenList
			} else {
				// Step back one pane
				m.ManageFocus--
				m.ManageWorkspaceInput.Blur()
			}
			return m, nil
		}

	case workspaceReposMsg:
		if msg.err != nil {
			m.WorkspaceReposState = StateError
			m.WorkspaceReposError = msg.err.Error()
		} else {
			m.WorkspaceRepos = msg.repos
			m.WorkspaceReposState = StateReady
			m.WorkspaceReposError = ""
		}
		return m, nil
	}
	return m, nil
}

// scrollFavoritesIntoView adjusts ManageFavScrollOff so ManageFavCursor is visible.
func (m *Model) scrollFavoritesIntoView() {
	visible := m.favoritesVisibleRows()
	if visible <= 0 {
		return
	}
	if m.ManageFavCursor < m.ManageFavScrollOff {
		m.ManageFavScrollOff = m.ManageFavCursor
	}
	if m.ManageFavCursor >= m.ManageFavScrollOff+visible {
		m.ManageFavScrollOff = m.ManageFavCursor - visible + 1
	}
	if m.ManageFavScrollOff < 0 {
		m.ManageFavScrollOff = 0
	}
}

// scrollReposIntoView adjusts ManageRepoScrollOff so ManageRepoCursor is visible.
func (m *Model) scrollReposIntoView() {
	visible := m.reposVisibleRows()
	if visible <= 0 {
		return
	}
	if m.ManageRepoCursor < m.ManageRepoScrollOff {
		m.ManageRepoScrollOff = m.ManageRepoCursor
	}
	if m.ManageRepoCursor >= m.ManageRepoScrollOff+visible {
		m.ManageRepoScrollOff = m.ManageRepoCursor - visible + 1
	}
	if m.ManageRepoScrollOff < 0 {
		m.ManageRepoScrollOff = 0
	}
}

// favoritesVisibleRows returns how many favorite rows can be displayed.
func (m Model) favoritesVisibleRows() int {
	// Left pane overhead: title(1) + divider(1) = 2 lines
	overhead := 2
	total := m.manageContentHeight()
	avail := total - overhead
	if avail < 0 {
		return 0
	}
	return avail
}

// reposVisibleRows returns how many repo rows can be displayed.
func (m Model) reposVisibleRows() int {
	// Right pane overhead: title(1) + divider(1) + input row(1) + blank(1) = 4 lines
	overhead := 4
	total := m.manageContentHeight()
	avail := total - overhead
	if avail < 0 {
		return 0
	}
	return avail
}

// manageContentHeight returns usable height for the manage panes.
func (m Model) manageContentHeight() int {
	headerLines := 2
	helpLines := 2
	contentHeight := m.Height - headerLines - helpLines
	if contentHeight < 4 {
		return 0
	}
	// The viewManageProjects receives contentHeight - 3 from viewContent
	return contentHeight - 3
}

// viewManageProjects renders the two-pane manage projects screen with scrolling.
func (m Model) viewManageProjects(contentHeight int) string {
	if contentHeight < 4 {
		return ""
	}

	avail := m.Width - 4
	leftWidth := avail / 2
	rightWidth := avail - leftWidth - 1

	// Calculate visible rows for each pane
	leftVisible := m.favoritesVisibleRows()
	rightVisible := m.reposVisibleRows()

	var leftSB, rightSB strings.Builder

	// ---- Left pane: Favorites ----
	leftSB.WriteString(CardTitleStyle.Render("★ Favorites"))
	leftSB.WriteString("\n")
	leftSB.WriteString(DividerStyle.Render(strings.Repeat("─", leftWidth)))
	leftSB.WriteString("\n")

	if len(m.Projects) == 0 {
		leftSB.WriteString(DimmedStyle.Render("  No favorites yet"))
		leftSB.WriteString("\n")
	} else {
		for i := m.ManageFavScrollOff; i < len(m.Projects) && (i-m.ManageFavScrollOff) < leftVisible; i++ {
			p := m.Projects[i]
			label := fmt.Sprintf("  %s / %s", p.Workspace, p.RepoSlug)
			rowStyle := ListNormalStyle
			prefix := "  "
			if m.ManageFocus == 0 && i == m.ManageFavCursor {
				rowStyle = ListCursorStyle
				prefix = "▶ "
			} else if i%2 == 1 {
				rowStyle = ListAltStyle
			}
			if i == m.ActiveProject {
				label += "  ◀ active"
			}
			line := rowStyle.Render(prefix + label)
			leftSB.WriteString(truncateStyled(line, leftWidth))
			leftSB.WriteString("\n")
		}
	}

	// Fill remaining lines
	currentLines := strings.Count(leftSB.String(), "\n")
	// subtract header+divider: we need exactly leftVisible item rows + header(2)
	needed := leftVisible + 2 // header + divider
	for currentLines < needed {
		leftSB.WriteString("\n")
		currentLines++
	}

	// ---- Right pane: Browse Workspace ----
	rightSB.WriteString(CardTitleStyle.Render("Browse Workspace"))
	rightSB.WriteString("\n")
	rightSB.WriteString(DividerStyle.Render(strings.Repeat("─", rightWidth)))
	rightSB.WriteString("\n")

	// Workspace input row
	inputLabel := "  Workspace: "
	if m.ManageFocus == 2 {
		inputLabel = "▶ Workspace: "
	}
	inputRow := inputLabel + m.ManageWorkspaceInput.View()
	rightSB.WriteString(inputRow)
	rightSB.WriteString("\n\n")

	switch m.WorkspaceReposState {
	case StateLoading:
		rightSB.WriteString(DimmedStyle.Render("  Loading repositories..."))
		rightSB.WriteString("\n")
	case StateError:
		rightSB.WriteString(ErrorStyle.Render("  Error: " + m.WorkspaceReposError))
		rightSB.WriteString("\n")
	case StateReady:
		if len(m.WorkspaceRepos) == 0 {
			rightSB.WriteString(DimmedStyle.Render("  No repositories found"))
			rightSB.WriteString("\n")
		} else {
			for i := m.ManageRepoScrollOff; i < len(m.WorkspaceRepos) && (i-m.ManageRepoScrollOff) < rightVisible; i++ {
				repo := m.WorkspaceRepos[i]
				parts := strings.SplitN(repo.FullName, "/", 2)
				repoSlug := repo.FullName
				if len(parts) == 2 {
					repoSlug = parts[1]
				}

				// Mark favorites
				isFav := false
				if len(parts) == 2 {
					isFav = m.Config.HasProject(parts[0], parts[1])
				}
				star := "  "
				if isFav {
					star = "★ "
				} else {
					star = "+ "
				}

				rowStyle := ListNormalStyle
				prefix := "  "
				if m.ManageFocus == 1 && i == m.ManageRepoCursor {
					rowStyle = ListCursorStyle
					prefix = "▶ "
				} else if i%2 == 1 {
					rowStyle = ListAltStyle
				}

				line := rowStyle.Render(prefix + star + repoSlug)
				rightSB.WriteString(truncateStyled(line, rightWidth))
				rightSB.WriteString("\n")
			}
		}
	default:
		rightSB.WriteString(DimmedStyle.Render("  Press tab to enter a workspace"))
		rightSB.WriteString("\n")
	}

	// Fill remaining lines
	currentLines = strings.Count(rightSB.String(), "\n")
	needed = rightVisible + 4 // header + divider + input + blank
	for currentLines < needed {
		rightSB.WriteString("\n")
		currentLines++
	}

	// Join left and right panes
	leftContent := leftSB.String()
	rightContent := rightSB.String()

	leftLinesArr := strings.Split(leftContent, "\n")
	rightLinesArr := strings.Split(rightContent, "\n")

	var result strings.Builder
	maxLines := len(leftLinesArr)
	if len(rightLinesArr) > maxLines {
		maxLines = len(rightLinesArr)
	}

	for i := 0; i < maxLines && i < contentHeight; i++ {
		l := ""
		r := ""
		if i < len(leftLinesArr) {
			l = leftLinesArr[i]
		}
		if i < len(rightLinesArr) {
			r = rightLinesArr[i]
		}
		// Pad left pane to leftWidth
		pad := leftWidth - lipglossWidth(l)
		if pad < 0 {
			pad = 0
		}
		sep := "│"
		result.WriteString(l + strings.Repeat(" ", pad) + sep + r + "\n")
	}

	result.WriteString("\n")
	result.WriteString(DividerStyle.Render(strings.Repeat("─", avail)))
	result.WriteString("\n")

	// Scroll indicators
	totalFav := len(m.Projects)
	totalRepo := len(m.WorkspaceRepos)
	scrollInfo := ""
	if totalFav > leftVisible {
		scrollInfo += fmt.Sprintf("Favs %d-%d/%d  ",
			m.ManageFavScrollOff+1, minInt(m.ManageFavScrollOff+leftVisible, totalFav), totalFav)
	}
	if totalRepo > rightVisible && m.WorkspaceReposState == StateReady {
		scrollInfo += fmt.Sprintf("Repos %d-%d/%d",
			m.ManageRepoScrollOff+1, minInt(m.ManageRepoScrollOff+rightVisible, totalRepo), totalRepo)
	}
	if scrollInfo != "" {
		result.WriteString(DimmedStyle.Render(scrollInfo))
		result.WriteString("\n")
	}

	return result.String()
}

// minInt returns the minimum of two ints.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// lipglossWidth returns the visual width of a string, counting only visible characters.
func lipglossWidth(s string) int {
	return lipgloss.Width(s)
}

// ---- Messages and commands ----

type workspaceReposMsg struct {
	repos []bitbucket.Repository
	err   error
}

func fetchWorkspaceRepos(client *bitbucket.CachedClient, workspace string) tea.Cmd {
	return func() tea.Msg {
		repos, err := client.ListAllRepositories(workspace)
		if err != nil {
			return workspaceReposMsg{err: err}
		}
		return workspaceReposMsg{repos: repos}
	}
}
