package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Logs Update ─────────────────────────────────────────────────────────────

// updateLogs handles messages for the log viewer screen.
func (m Model) updateLogs(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case stepLogLoadedMsg:
		if msg.err != nil {
			m.handleErrorMsg("log", msg.err)
			return m, nil
		}
		// If this was a refresh, preserve user scroll position and search state;
		// only reset them on the initial load (StateLoading → StateReady).
		isRefresh := m.LogState == StateReady
		// Track previous size for delta display
		if isRefresh {
			m.LogPrevSize = strings.Count(m.LogContent, "\n") + 1
		}
		m.LogContent = msg.content
		m.LogStepName = msg.stepName
		m.LogState = StateReady
		if !isRefresh {
			m.LogSearchTerm = ""
			m.LogMatchIndex = -1
			m.LogMatchLines = nil
			m.LogScrollOff = 0
			m.LogHScroll = 0
			m.LogPrevSize = 0
		}
		// Reset countdown on each successful refresh
		if m.LogAutoRefresh {
			m.LogAutoRefreshCountdown = 10
		}
		// Stop auto-refresh when the step reaches a terminal state.
		if msg.stepCompleted && m.LogAutoRefresh {
			m.LogAutoRefresh = false
		}
		return m, nil

	case tickMsg:
		// Auto-refresh: decrement countdown, fire when it reaches 0.
		if m.LogAutoRefresh && m.LogPipelineUUID != "" && m.LogStepUUID != "" {
			if m.LogAutoRefreshCountdown > 1 {
				m.LogAutoRefreshCountdown--
				return m, tickCmd()
			}
			// Countdown reached 0 — fire refresh
			m.LogAutoRefreshCountdown = 10
			m.LogScrollOff = m.maxLogScroll()
			workspace := m.Projects[m.ActiveProject].Workspace
			repoSlug := m.Projects[m.ActiveProject].RepoSlug
			m.Client.InvalidateCacheForRepo(workspace, repoSlug)
			return m, tea.Batch(
				fetchStepLogWithStatus(m.Client, workspace, repoSlug,
					m.LogPipelineUUID, m.LogStepUUID, m.LogStepName),
				tickCmd(),
			)
		}
		return m, nil

	case tea.KeyMsg:
		// Search mode
		if m.LogSearchMode {
			return m.handleLogSearchInput(msg)
		}

		switch msg.String() {
		case "up", "k":
			if m.LogScrollOff > 0 {
				m.LogScrollOff--
			}
			return m, nil

		case "down", "j":
			maxScroll := m.maxLogScroll()
			if m.LogScrollOff < maxScroll {
				m.LogScrollOff++
			}
			return m, nil

		case "left", "h":
			if m.LogHScroll > 0 {
				m.LogHScroll--
			}
			return m, nil

		case "right", "l":
			maxH := m.maxLogHScroll()
			if m.LogHScroll < maxH {
				m.LogHScroll++
			}
			return m, nil

		case "pgup":
			pageSize := m.logViewportHeight()
			m.LogScrollOff -= pageSize
			if m.LogScrollOff < 0 {
				m.LogScrollOff = 0
			}
			return m, nil

		case "pgdown":
			pageSize := m.logViewportHeight()
			maxScroll := m.maxLogScroll()
			m.LogScrollOff += pageSize
			if m.LogScrollOff > maxScroll {
				m.LogScrollOff = maxScroll
			}
			return m, nil

		case "/":
			m.LogSearchMode = true
			m.LogSearchTerm = ""
			m.LogMatchIndex = -1
			m.LogMatchLines = nil
			return m, nil

		case "v":
			// Toggle variable display: parse from log content and show parsed vars
			if m.LogShowVars {
				m.LogShowVars = false
			} else {
				if m.LogContent != "" {
					m.ParsedLogVars = ParsePipelineVariablesFromLog(m.LogContent)
				}
				m.LogShowVars = true
			}
			return m, nil

		case "n":
			// Next search match
			if len(m.LogMatchLines) > 0 {
				m.LogMatchIndex++
				if m.LogMatchIndex >= len(m.LogMatchLines) {
					m.LogMatchIndex = 0
				}
				// Scroll to match
				m.scrollToMatchLine(m.LogMatchLines[m.LogMatchIndex])
			}
			return m, nil

		case "N", "shift+n":
			// Previous search match
			if len(m.LogMatchLines) > 0 {
				m.LogMatchIndex--
				if m.LogMatchIndex < 0 {
					m.LogMatchIndex = len(m.LogMatchLines) - 1
				}
				m.scrollToMatchLine(m.LogMatchLines[m.LogMatchIndex])
			}
			return m, nil

		case "r":
			// Refresh the step log once, preserving scroll and search.
			// Stop auto-refresh if active.
			m.LogAutoRefresh = false
			if m.LogPipelineUUID != "" && m.LogStepUUID != "" {
				workspace := m.Projects[m.ActiveProject].Workspace
				repoSlug := m.Projects[m.ActiveProject].RepoSlug
				m.Client.InvalidateCacheForRepo(workspace, repoSlug)
				return m, fetchStepLogWithStatus(m.Client, workspace, repoSlug,
					m.LogPipelineUUID, m.LogStepUUID, m.LogStepName)
			}
			return m, nil

		case "R", "shift+r":
			// Toggle auto-refresh (every 30s, scroll-to-bottom, stop on completion).
			m.LogAutoRefresh = !m.LogAutoRefresh
			if m.LogAutoRefresh && m.LogPipelineUUID != "" && m.LogStepUUID != "" {
				m.LogScrollOff = m.maxLogScroll()
				workspace := m.Projects[m.ActiveProject].Workspace
				repoSlug := m.Projects[m.ActiveProject].RepoSlug
				m.Client.InvalidateCacheForRepo(workspace, repoSlug)
				return m, tea.Batch(
					fetchStepLogWithStatus(m.Client, workspace, repoSlug,
						m.LogPipelineUUID, m.LogStepUUID, m.LogStepName),
					tickCmd(),
				)
			}
			return m, nil

		case "esc":
			// If in vars mode, return to log
			if m.LogShowVars {
				m.LogShowVars = false
				return m, nil
			}
			// Clear search if active, otherwise go back to detail
			if len(m.LogMatchLines) > 0 {
				m.LogSearchTerm = ""
				m.LogMatchIndex = -1
				m.LogMatchLines = nil
			} else {
				m.Screen = ScreenDetail
			}
			return m, nil

		case "enter", " ":
			// Clear search
			m.LogSearchTerm = ""
			m.LogMatchIndex = -1
			m.LogMatchLines = nil
			return m, nil
		}
	}
	return m, nil
}

// handleLogSearchInput handles key presses while in search mode.
func (m *Model) handleLogSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Execute search
		m.LogSearchMode = false
		m.performLogSearch()
		return m, nil

	case "esc":
		// Cancel search
		m.LogSearchMode = false
		m.LogSearchTerm = ""
		m.LogMatchIndex = -1
		m.LogMatchLines = nil
		return m, nil

	default:
		if len(msg.String()) == 1 && msg.String() >= " " {
			m.LogSearchTerm += msg.String()
		}
		return m, nil
	}
}

// performLogSearch finds all lines matching the search term.
func (m *Model) performLogSearch() {
	if m.LogSearchTerm == "" || m.LogContent == "" {
		return
	}
	lower := strings.ToLower(m.LogSearchTerm)
	lines := strings.Split(m.LogContent, "\n")
	m.LogMatchLines = nil
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), lower) {
			m.LogMatchLines = append(m.LogMatchLines, i)
		}
	}
	if len(m.LogMatchLines) > 0 {
		m.LogMatchIndex = 0
		m.scrollToMatchLine(m.LogMatchLines[0])
	}
}

// scrollToMatchLine scrolls the log view so the given line is visible.
func (m *Model) scrollToMatchLine(line int) {
	if m.LogScrollOff <= line && line < m.LogScrollOff+m.logViewportHeight() {
		return
	}
	m.LogScrollOff = line - m.logViewportHeight()/2
	if m.LogScrollOff < 0 {
		m.LogScrollOff = 0
	}
}

// maxLogScroll returns the maximum vertical scroll offset for the log content.
func (m Model) maxLogScroll() int {
	lines := strings.Count(m.LogContent, "\n") + 1
	maxScroll := lines - m.logViewportHeight()
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

// maxLogHScroll returns the maximum horizontal scroll offset for the current
// viewport, based on the longest visible line.
func (m Model) maxLogHScroll() int {
	contentWidth := m.Width - 4 - 7 // avail minus line number prefix
	if contentWidth < 20 {
		return 0
	}
	lines := strings.Split(m.LogContent, "\n")
	totalLines := len(lines)
	viewportH := m.logViewportHeight()
	start := m.LogScrollOff
	end := start + viewportH
	if end > totalLines {
		end = totalLines
	}
	if start > end {
		return 0
	}

	maxLen := 0
	for i := start; i < end; i++ {
		line := strings.TrimRight(lines[i], "\r")
		l := lipgloss.Width(line)
		if l > maxLen {
			maxLen = l
		}
	}
	maxScroll := maxLen - contentWidth
	if maxScroll < 0 {
		return 0
	}
	// Allow generous overscroll for CJK/narrow glyph correction
	return maxScroll + 2
}

// logViewportHeight returns the available number of lines for log content.
func (m Model) logViewportHeight() int {
	// contentHeight = m.Height - header(2) - help(2) - padding(3) = m.Height - 7
	// Within that: search bar(1) + divider(1) + divider(1) + scroll info(1) = 4 overhead
	contentHeight := m.Height - 7
	h := contentHeight - 4
	if h < 1 {
		return 1
	}
	return h
}

// ── Logs View ───────────────────────────────────────────────────────────────

// viewLogs renders the step log viewer with line numbers, search highlighting,
// and match counter.
func (m Model) viewLogs(contentHeight int) string {
	if m.LogState == StateLoading {
		return viewLoading("Loading step log")
	}
	if m.LogState == StateError {
		return viewError(m.LogError)
	}
	if m.LogContent == "" {
		return viewEmpty("No log content", "Press 'r' to retry or 'esc' to go back")
	}

	avail := m.Width - 4

	// ── Search bar ────────────────────────────────────────────────────────
	var searchBar string
	if m.LogSearchMode {
		searchBar = FilterStyle.Render(" Search: " + m.LogSearchTerm + "_ ")
	} else if m.LogSearchTerm != "" {
		searchBar = FilterStyle.Render(" Search: " + m.LogSearchTerm + " ")
	} else {
		// Auto-refresh indicator
		autoRefreshInfo := ""
		if m.LogAutoRefresh {
			autoRefreshInfo = " " + BadgeStyle.Render(fmt.Sprintf("  ⟳ AUTO %ds  ", m.LogAutoRefreshCountdown))
		}
		searchBar = DimmedStyle.Render(" / search  r refresh  R auto-refresh  esc back") + autoRefreshInfo
	}

	// Match counter
	if len(m.LogMatchLines) > 0 && m.LogMatchIndex >= 0 {
		searchBar += "  " + LogMatchCounterStyle.Render(
			fmt.Sprintf("Match %d of %d", m.LogMatchIndex+1, len(m.LogMatchLines)),
		)
	}

	// ── Parsed variables display ──────────────────────────────────────────
	if m.LogShowVars {
		// Show parsed pipeline variables
		sectionTitle := SectionTitleStyle.Render(fmt.Sprintf("Pipeline Variables (%d)", len(m.ParsedLogVars)))
		var content string
		if len(m.ParsedLogVars) == 0 {
			content = viewEmpty("No pipeline variables found",
				"The step log did not contain a \"Pipeline variables:\" block")
		} else {
			content = CardStyle.Width(avail - 4).Render(buildVarList(m.ParsedLogVars, avail-8))
		}

		// Vars-only toggle hint
		varsHint := BadgeStyle.Render(" variables mode ") + " " + DimmedStyle.Render("Press v to return to log")

		return lipgloss.JoinVertical(lipgloss.Left,
			searchBar,
			DividerStyle.Render(strings.Repeat("─", avail)),
			lipgloss.JoinVertical(lipgloss.Left, sectionTitle, content),
			DividerStyle.Render(strings.Repeat("─", avail)),
			DimmedStyle.Render(varsHint),
		)
	}

	// ── Log content with line numbers ─────────────────────────────────────
	lines := strings.Split(m.LogContent, "\n")
	totalLines := len(lines)
	lowerSearch := strings.ToLower(m.LogSearchTerm)

	viewportH := contentHeight - 4 // search bar + divider + help
	if viewportH < 1 {
		viewportH = 1
	}
	// Use stored scroll offset
	scrollOff := m.LogScrollOff

	end := scrollOff + viewportH
	if end > totalLines {
		end = totalLines
	}
	start := scrollOff
	if start > end {
		start = end
	}
	visible := lines[start:end]

	// Scroll indicator
	scrollPct := 0
	if totalLines > viewportH {
		scrollPct = scrollOff * 100 / (totalLines - viewportH)
	}
	// Size delta: compare current line count with previous
	sizeDelta := 0
	if m.LogPrevSize > 0 && totalLines != m.LogPrevSize {
		sizeDelta = totalLines - m.LogPrevSize
	}
	scrollText := fmt.Sprintf("Lines %d-%d of %d (%d%%)",
		start+1, end, totalLines, scrollPct)
	if sizeDelta > 0 {
		scrollText += fmt.Sprintf("  +%d lines", sizeDelta)
	} else if sizeDelta < 0 {
		scrollText += fmt.Sprintf("  %d lines", sizeDelta)
	}
	// Auto-refresh countdown in scroll bar
	if m.LogAutoRefresh {
		scrollText += fmt.Sprintf("  next refresh in %ds", m.LogAutoRefreshCountdown)
	}
	scrollInfo := DimmedStyle.Render(scrollText)

	// Pre-compute highlight positions for each visible line
	highlightPositions := make([][2]int, len(visible))
	if m.LogSearchTerm != "" && m.LogMatchIndex >= 0 {
		for i, line := range visible {
			absLineNum := start + i
			// Highlight if this is the current match
			if absLineNum == m.LogMatchLines[m.LogMatchIndex] {
				idx := strings.Index(strings.ToLower(line), lowerSearch)
				if idx >= 0 {
					highlightPositions[i] = [2]int{idx, idx + len(m.LogSearchTerm)}
				}
			}
		}
	}

	// Content width (accounting for line number prefix)
	contentWidth := avail - 7 // 5 for line num + 2 safety margin
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Clamp horizontal scroll
	hScroll := m.LogHScroll
	maxH := m.maxLogHScroll()
	if hScroll > maxH {
		hScroll = maxH
	}

	var sb strings.Builder
	for i, line := range visible {
		lineNum := start + i + 1
		lineNumStr := LogLineNumStyle.Render(fmt.Sprintf("%d ", lineNum))

		// Strip trailing \r for cleanliness
		line = strings.TrimRight(line, "\r")

		// Apply horizontal scroll: slice from offset
		runes := []rune(line)
		hOff := hScroll
		trimmedStart := 0
		col := 0
		for j, r := range runes {
			charW := 1
			if r >= 0x4e00 && r <= 0x9fff || r >= 0x3000 && r <= 0x303f || r >= 0xff00 {
				charW = 2
			}
			if col >= hOff {
				trimmedStart = j
				break
			}
			col += charW
		}
		displayLine := string(runes[trimmedStart:])

		// Truncate to content width if still too wide
		displayLine = truncateLogLine(displayLine, contentWidth)

		// Adjust highlight positions for horizontal offset
		hp := highlightPositions[i]
		if hp[0] < hp[1] && hp[0] >= hOff {
			adjStart := hp[0] - hOff
			adjEnd := hp[1] - hOff
			if adjEnd > len(displayLine) {
				adjEnd = len(displayLine)
			}
			if adjStart < len(displayLine) && adjEnd > adjStart {
				before := displayLine[:adjStart]
				match := displayLine[adjStart:adjEnd]
				after := displayLine[adjEnd:]
				sb.WriteString(lineNumStr)
				sb.WriteString(before)
				sb.WriteString(LogHighlightStyle.Render(match))
				sb.WriteString(after)
			} else {
				sb.WriteString(lineNumStr)
				sb.WriteString(displayLine)
			}
		} else {
			sb.WriteString(lineNumStr)
			sb.WriteString(displayLine)
		}
		if i < len(visible)-1 {
			sb.WriteString("\n")
		}
	}

	// Horizontal scroll indicator
	hScrollInfo := ""
	if maxH > 0 {
		hEnd := hScroll + contentWidth
		if hEnd > maxH+hScroll {
			hEnd = maxH + hScroll
		}
		hScrollInfo = fmt.Sprintf("  col %d-%d (←→)", hScroll+1, hEnd)
	}

	// ── Rendered variables indicator ──────────────────────────────────────
	varsNote := ""
	if len(m.ParsedLogVars) > 0 {
		varsNote = "  " + BadgeStyle.Render(fmt.Sprintf(" %d vars parsed ", len(m.ParsedLogVars)))
	}

	// ── Combine ───────────────────────────────────────────────────────────
	return lipgloss.JoinVertical(lipgloss.Left,
		searchBar,
		DividerStyle.Render(strings.Repeat("─", avail)),
		sb.String(),
		DividerStyle.Render(strings.Repeat("─", avail)),
		DimmedStyle.Render(scrollInfo+hScrollInfo+varsNote),
	)
}
