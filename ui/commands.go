package ui

import (
	"net/url"
	"strings"

	"github.com/bbc/infra-pipeline-ui/bitbucket"
	tea "github.com/charmbracelet/bubbletea"
)

// Messages for async operations
type pipelinesLoadedMsg struct {
	pipelines   []bitbucket.Pipeline
	nextPageURL string
	err         error
}

type nextPageLoadedMsg struct {
	pipelines   []bitbucket.Pipeline
	nextPageURL string
	err         error
}

type pipelineDetailLoadedMsg struct {
	pipeline *bitbucket.Pipeline
	steps    []bitbucket.PipelineStep
	err      error
}

type stepLogLoadedMsg struct {
	content  string
	stepName string
	err      error
}

type pipelineTriggeredMsg struct {
	pipeline *bitbucket.Pipeline
	err      error
}

type configVarsLoadedMsg struct {
	variables []bitbucket.PipelineVariable
	err       error
}

type logVarsLoadedMsg struct {
	variables []bitbucket.PipelineVariable
	err       error
}

type pipelinePreviewLoadedMsg struct {
	pipeline  *bitbucket.Pipeline
	steps     []bitbucket.PipelineStep
	variables []bitbucket.PipelineVariable
	err       error
}

// fetchPipelines loads the first page of pipelines for the active project.
func fetchPipelines(client *bitbucket.CachedClient, workspace, repoSlug string) tea.Cmd {
	return func() tea.Msg {
		params := &bitbucket.ListPipelinesParams{
			Pagelen: 25,
			Sort:    "-created_on",
		}
		result, err := client.ListPipelines(workspace, repoSlug, params)
		if err != nil {
			return pipelinesLoadedMsg{err: err}
		}
		return pipelinesLoadedMsg{
			pipelines:   result.Values,
			nextPageURL: result.Next,
		}
	}
}

// fetchNextPage loads the next page of pipelines using the next URL.
func fetchNextPage(client *bitbucket.CachedClient, nextURL string) tea.Cmd {
	return func() tea.Msg {
		if nextURL == "" {
			return nextPageLoadedMsg{}
		}
		// Bitbucket next URLs may not preserve sort; explicitly add it to avoid
		// getting the oldest pipelines on subsequent pages.
		if !strings.Contains(nextURL, "sort=") {
			if strings.Contains(nextURL, "?") {
				nextURL += "&sort=" + url.QueryEscape("-created_on")
			} else {
				nextURL += "?sort=" + url.QueryEscape("-created_on")
			}
		}
		var next bitbucket.PaginatedPipelines
		if err := client.GetFullURL(nextURL, &next); err != nil {
			return nextPageLoadedMsg{err: err}
		}
		return nextPageLoadedMsg{
			pipelines:   next.Values,
			nextPageURL: next.Next,
		}
	}
}

// fetchPipelineDetail loads a single pipeline and its steps.
func fetchPipelineDetail(client *bitbucket.CachedClient, workspace, repoSlug, pipelineUUID string) tea.Cmd {
	return func() tea.Msg {
		pipeline, err := client.GetPipeline(workspace, repoSlug, pipelineUUID)
		if err != nil {
			return pipelineDetailLoadedMsg{err: err}
		}
		stepsResult, err := client.ListPipelineSteps(workspace, repoSlug, pipelineUUID)
		if err != nil {
			return pipelineDetailLoadedMsg{pipeline: pipeline, err: err}
		}
		return pipelineDetailLoadedMsg{
			pipeline: pipeline,
			steps:    stepsResult.Values,
		}
	}
}

// fetchStepLog loads the log for a specific step.
func fetchStepLog(client *bitbucket.CachedClient, workspace, repoSlug, pipelineUUID, stepUUID, stepName string) tea.Cmd {
	return func() tea.Msg {
		content, err := client.GetStepLog(workspace, repoSlug, pipelineUUID, stepUUID)
		if err != nil {
			return stepLogLoadedMsg{stepName: stepName, err: err}
		}
		return stepLogLoadedMsg{
			content:  content,
			stepName: stepName,
		}
	}
}

// triggerPipelineCmd triggers a new pipeline run.
func triggerPipelineCmd(client *bitbucket.CachedClient, workspace, repoSlug string, req bitbucket.TriggerPipelineRequest) tea.Cmd {
	return func() tea.Msg {
		pipeline, err := client.TriggerPipeline(workspace, repoSlug, req)
		if err != nil {
			return pipelineTriggeredMsg{err: err}
		}
		return pipelineTriggeredMsg{pipeline: pipeline}
	}
}

// fetchConfigVars loads repository-level pipeline config variables.
func fetchConfigVars(client *bitbucket.CachedClient, workspace, repoSlug string) tea.Cmd {
	return func() tea.Msg {
		result, err := client.ListPipelineVariables(workspace, repoSlug)
		if err != nil {
			return configVarsLoadedMsg{err: err}
		}
		return configVarsLoadedMsg{variables: result.Values}
	}
}

// fetchLogVarsForPipeline fetches the first step's log and parses pipeline variables from it.
// This is used on the detail screen to get actual pipeline variables from the run log,
// since the API may not return them for custom pipelines.
func fetchLogVarsForPipeline(client *bitbucket.CachedClient, workspace, repoSlug, pipelineUUID string) tea.Cmd {
	return func() tea.Msg {
		stepsResult, err := client.ListPipelineSteps(workspace, repoSlug, pipelineUUID)
		if err != nil || len(stepsResult.Values) == 0 {
			if err != nil {
				return logVarsLoadedMsg{err: err}
			}
			return logVarsLoadedMsg{}
		}
		firstStep := stepsResult.Values[0]
		content, err := client.GetStepLog(workspace, repoSlug, pipelineUUID, firstStep.UUID)
		if err != nil {
			return logVarsLoadedMsg{err: err}
		}
		vars := ParsePipelineVariablesFromLog(content)
		return logVarsLoadedMsg{variables: vars}
	}
}

// fetchPipelinePreview loads a pipeline's detail, steps, and config vars for the
// split-panel preview on the list screen.
func fetchPipelinePreview(client *bitbucket.CachedClient, workspace, repoSlug, pipelineUUID string) tea.Cmd {
	return func() tea.Msg {
		pipeline, pErr := client.GetPipeline(workspace, repoSlug, pipelineUUID)
		if pErr != nil {
			return pipelinePreviewLoadedMsg{err: pErr}
		}
		stepsResult, sErr := client.ListPipelineSteps(workspace, repoSlug, pipelineUUID)
		var steps []bitbucket.PipelineStep
		if sErr == nil {
			steps = stepsResult.Values
		}
		// Parse pipeline variables from the first step's build log
		var vars []bitbucket.PipelineVariable
		if len(steps) > 0 {
			content, logErr := client.GetStepLog(workspace, repoSlug, pipelineUUID, steps[0].UUID)
			if logErr == nil {
				vars = ParsePipelineVariablesFromLog(content)
			}
		}
		return pipelinePreviewLoadedMsg{
			pipeline:  pipeline,
			steps:     steps,
			variables: vars,
		}
	}
}
