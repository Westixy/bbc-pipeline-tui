package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	"github.com/charmbracelet/lipgloss"
)

// filterMatchesPipeline checks if a pipeline matches the filter text.
func filterMatchesPipeline(p bitbucket.Pipeline, lowerFilter string) bool {
	if strings.Contains(strings.ToLower(fmt.Sprintf("%d", p.BuildNumber)), lowerFilter) {
		return true
	}
	if strings.Contains(strings.ToLower(p.Target.RefName), lowerFilter) {
		return true
	}
	if strings.Contains(strings.ToLower(pipelineTypeLabel(p.Target)), lowerFilter) {
		return true
	}
	if strings.Contains(strings.ToLower(p.State.Name), lowerFilter) {
		return true
	}
	if p.State.Result != nil && strings.Contains(strings.ToLower(p.State.Result.Name), lowerFilter) {
		return true
	}
	if p.Trigger.Name != "" && strings.Contains(strings.ToLower(p.Trigger.Name), lowerFilter) {
		return true
	}
	return false
}

// filteredPipelines returns pipelines that match the filter text.
func filteredPipelines(pipelines []bitbucket.Pipeline, filter string) []bitbucket.Pipeline {
	if filter == "" {
		return pipelines
	}
	lower := strings.ToLower(filter)
	var result []bitbucket.Pipeline
	for _, p := range pipelines {
		if filterMatchesPipeline(p, lower) {
			result = append(result, p)
		}
	}
	return result
}

// truncate truncates a string to maxLen, appending "…" if truncated.
// Uses rune-level slicing to safely handle multi-byte UTF-8 characters.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen < 3 {
		return strings.Repeat("…", maxLen)
	}
	return string(runes[:maxLen-1]) + "…"
}

// truncateStyled truncates a styled string to fit maxLen using visual width.
func truncateStyled(s string, maxLen int) string {
	if maxLen < 1 {
		return ""
	}
	visualLen := lipgloss.Width(s)
	if visualLen <= maxLen {
		return s
	}
	runes := []rune(s)
	var result strings.Builder
	used := 0
	for _, r := range runes {
		if used >= maxLen {
			break
		}
		result.WriteRune(r)
		used++
	}
	return result.String()
}

// truncateLogLine truncates a log line to maxLen, appending "…" if truncated.
// Uses rune-level truncation to handle ANSI codes and Unicode correctly.
func truncateLogLine(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	visualLen := lipgloss.Width(s)
	if visualLen <= maxLen {
		return s
	}
	// Strip trailing \r and truncate by visual width
	s = strings.TrimRight(s, "\r")
	runes := []rune(s)
	var result strings.Builder
	used := 0
	for _, r := range runes {
		charW := 1
		if r >= 0x4e00 && r <= 0x9fff || r >= 0x3000 && r <= 0x303f || r >= 0xff00 {
			charW = 2 // CJK-like wide chars
		}
		if used+charW > maxLen-1 {
			break
		}
		result.WriteRune(r)
		used += charW
	}
	return result.String() + "…"
}

// creatorName returns the display name of the pipeline creator, or "N/A" if nil.
func creatorName(c *bitbucket.Account) string {
	if c == nil {
		return "N/A"
	}
	return c.DisplayName
}

// resolveStepStatus resolves a step state to a canonical status string.
func resolveStepStatus(state bitbucket.PipelineStepState) string {
	resultName := ""
	if state.Result != nil {
		resultName = state.Result.Name
	}
	return resolvePipelineResult(state.Name, resultName)
}

// resolvePipelineResult resolves a pipeline/step state name and optional
// result name into a canonical status string (IN_PROGRESS, PENDING,
// SUCCESSFUL, FAILED, STOPPED, COMPLETED).
func resolvePipelineResult(stateName, resultName string) string {
	switch stateName {
	case "IN_PROGRESS":
		return "IN_PROGRESS"
	case "PENDING":
		return "PENDING"
	case "COMPLETED":
		switch resultName {
		case "SUCCESSFUL":
			return "SUCCESSFUL"
		case "FAILED":
			return "FAILED"
		default:
			return "STOPPED"
		}
	default:
		return stateName
	}
}

// formatTime parses and formats an ISO 8601 timestamp into a readable format.
func formatTime(ts string) string {
	if ts == "" {
		return "N/A"
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04:05")
}

// formatRelativeTime returns a short relative time string (e.g. "2m ago", "1h ago", "3d ago").
func formatRelativeTime(ts string) string {
	if ts == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	d := time.Since(t)
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 02")
	}
}

// padToWidth pads a string (which may contain ANSI escape codes) to the
// target visual width by appending spaces.
func padToWidth(s string, w int) string {
	vw := lipgloss.Width(s)
	if vw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vw)
}

// formatDuration computes the pipeline duration from CreatedOn/CompletedOn
// timestamps, falling back to BuildSecondsUsed if timestamps are unavailable.
func formatDuration(createdOn, completedOn string, buildSecondsUsed int) string {
	if completedOn != "" {
		created, err1 := time.Parse(time.RFC3339, createdOn)
		completed, err2 := time.Parse(time.RFC3339, completedOn)
		if err1 == nil && err2 == nil {
			d := completed.Sub(created)
			if d > 0 {
				return formatDurationCompact(int(d.Seconds()))
			}
		}
	}
	if buildSecondsUsed > 0 {
		return formatDurationCompact(buildSecondsUsed)
	}
	return "—"
}

// formatDurationCompact formats a duration in seconds to a compact string.
func formatDurationCompact(seconds int) string {
	if seconds <= 0 {
		return "—"
	}
	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// formatDurationSeconds formats a duration in seconds to a human-readable string.
func formatDurationSeconds(seconds int) string {
	if seconds <= 0 {
		return "N/A"
	}
	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// renderStatusBadge renders a colored status badge string.
func renderStatusBadge(status string) string {
	switch status {
	case "IN_PROGRESS":
		return StatusInProgress.Render(" ◉ IN_PROGRESS ")
	case "PENDING":
		return StatusPending.Render(" ◌ PENDING ")
	case "SUCCESSFUL":
		return StatusSuccessful.Render(" ✓ SUCCESSFUL ")
	case "FAILED":
		return StatusFailed.Render(" ✗ FAILED ")
	case "STOPPED":
		return StatusStopped.Render(" ⊘ STOPPED ")
	default:
		return DimmedStyle.Render(" " + status + " ")
	}
}

// statusBadgeCompact returns a short colored status pill for the pipeline list.
func statusBadgeCompact(status string) string {
	switch status {
	case "IN_PROGRESS":
		return StatusInProgress.Render("●IN_PROG")
	case "PENDING":
		return StatusPending.Render("◌PENDING")
	case "SUCCESSFUL":
		return StatusSuccessful.Render("✓SUCCESS ")
	case "FAILED":
		return StatusFailed.Render("✗FAILED  ")
	case "STOPPED":
		return StatusStopped.Render("⊘STOPPED")
	default:
		return DimmedStyle.Render(status)
	}
}

// pipelineTypeLabel produces a short display label for the pipeline type
// in the format: branch:main, tag:v1.0, custom:abc, default.
func pipelineTypeLabel(target bitbucket.PipelineTarget) string {
	if target.Selector != nil {
		if target.Selector.Type == "custom" {
			return "custom:" + target.Selector.Pattern
		}
		return target.Selector.Type + ":" + target.Selector.Pattern
	}
	if target.RefType != "" {
		return target.RefType + ":" + target.RefName
	}
	return target.Type
}

// pipelineTypeIcon returns a simple icon for a pipeline type.
func pipelineTypeIcon(target bitbucket.PipelineTarget) string {
	if target.Selector != nil {
		return "⚙"
	}
	switch target.RefType {
	case "branch":
		return "⎇"
	case "tag":
		return "🏷"
	case "commit":
		return "◆"
	default:
		return "●"
	}
}

// clampCursor ensures the cursor stays within valid bounds.
func clampCursor(cursor, max int) int {
	if cursor < 0 {
		return 0
	}
	if max == 0 {
		return 0
	}
	if cursor >= max {
		return max - 1
	}
	return cursor
}

// renderScrollbar returns a vertical scrollbar string for the given position.
// height is the total number of visible rows, pos is the current offset,
// total is the total number of items.
func renderScrollbar(height, pos, total int) string {
	if total <= height || height < 1 {
		return ""
	}
	thumbSize := height * height / total
	if thumbSize < 1 {
		thumbSize = 1
	}
	thumbPos := pos * height / total
	if thumbPos+thumbSize > height {
		thumbPos = height - thumbSize
	}

	var sb strings.Builder
	for i := 0; i < height; i++ {
		if i >= thumbPos && i < thumbPos+thumbSize {
			sb.WriteString(ScrollThumbStyle.Render("█"))
		} else {
			sb.WriteString(ScrollTrackStyle.Render("│"))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// varBlockRe matches the "Pipeline variables:" block in step log output.
var varBlockRe = regexp.MustCompile(`(?m)^Pipeline variables:\n((?:[ \t]+\w[\w.]*:[ \t]+[^\n]*\n)*)`)

// ParsePipelineVariablesFromLog extracts pipeline variables from a step log
// that contains a "Pipeline variables:" block.
func ParsePipelineVariablesFromLog(logContent string) []bitbucket.PipelineVariable {
	m := varBlockRe.FindStringSubmatch(logContent)
	if m == nil {
		return nil
	}
	block := m[1]
	lines := strings.Split(block, "\n")
	var vars []bitbucket.PipelineVariable
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Split on first colon
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if key != "" {
			vars = append(vars, bitbucket.PipelineVariable{Key: key, Value: value})
		}
	}
	return vars
}
