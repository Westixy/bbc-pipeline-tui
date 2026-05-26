package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	"github.com/charmbracelet/lipgloss"
)

// filteredPipelines returns pipelines that match the filter text.
// It filters by build number, branch name, type, or status.
func filteredPipelines(pipelines []bitbucket.Pipeline, filter string) []bitbucket.Pipeline {
	if filter == "" {
		return pipelines
	}
	lower := strings.ToLower(filter)
	var result []bitbucket.Pipeline
	for _, p := range pipelines {
		if strings.Contains(strings.ToLower(fmt.Sprintf("%d", p.BuildNumber)), lower) {
			result = append(result, p)
			continue
		}
		if strings.Contains(strings.ToLower(p.Target.RefName), lower) {
			result = append(result, p)
			continue
		}
		if strings.Contains(strings.ToLower(p.Target.Type), lower) {
			result = append(result, p)
			continue
		}
		if strings.Contains(strings.ToLower(p.State.Name), lower) {
			result = append(result, p)
			continue
		}
		if p.State.Result != nil && strings.Contains(strings.ToLower(p.State.Result.Name), lower) {
			result = append(result, p)
			continue
		}
		if p.Trigger.Name != "" && strings.Contains(strings.ToLower(p.Trigger.Name), lower) {
			result = append(result, p)
			continue
		}
	}
	return result
}

// truncate truncates a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// creatorName returns the display name of the pipeline creator, or "N/A" if nil.
func creatorName(c *bitbucket.Account) string {
	if c == nil {
		return "N/A"
	}
	return c.DisplayName
}

// statusStyle returns the appropriate style for a pipeline state.
func statusStyle(state bitbucket.PipelineState) string {
	switch state.Name {
	case "IN_PROGRESS":
		return "IN_PROGRESS"
	case "PENDING":
		return "PENDING"
	case "COMPLETED":
		if state.Result != nil {
			switch state.Result.Name {
			case "SUCCESSFUL":
				return "SUCCESSFUL"
			case "FAILED":
				return "FAILED"
			default:
				return "STOPPED"
			}
		}
		return "COMPLETED"
	default:
		return ""
	}
}

// stepStatusStyle returns the appropriate style name for a step state.
func stepStatusStyle(state bitbucket.PipelineStepState) string {
	switch state.Name {
	case "IN_PROGRESS":
		return "IN_PROGRESS"
	case "PENDING":
		return "PENDING"
	case "COMPLETED":
		if state.Result != nil {
			switch state.Result.Name {
			case "SUCCESSFUL":
				return "SUCCESSFUL"
			case "FAILED":
				return "FAILED"
			default:
				return "STOPPED"
			}
		}
		return "COMPLETED"
	default:
		return ""
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
				return formatDurationSeconds(int(d.Seconds()))
			}
		}
	}
	if buildSecondsUsed > 0 {
		return formatDurationSeconds(buildSecondsUsed)
	}
	return "N/A"
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
		return StatusInProgress.Render("◉ IN_PROGRESS")
	case "PENDING":
		return StatusPending.Render("◌ PENDING")
	case "SUCCESSFUL":
		return StatusSuccessful.Render("✓ SUCCESSFUL")
	case "FAILED":
		return StatusFailed.Render("✗ FAILED")
	case "STOPPED":
		return StatusStopped.Render("⊘ STOPPED")
	default:
		return DimmedStyle.Render(status)
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

// varBlockRe matches the "Pipeline variables:" block in step log output.
var varBlockRe = regexp.MustCompile(`(?m)^Pipeline variables:\n((?:\s+\w[\w.]*:\s.*\n?)*)`)

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
