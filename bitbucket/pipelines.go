package bitbucket

import (
	"fmt"
	"net/url"
	"strings"
)

// ListPipelines fetches pipelines for the given workspace/repo with optional params.
func (c *Client) ListPipelines(workspace, repoSlug string, params *ListPipelinesParams) (*PaginatedPipelines, error) {
	basePath := BuildPipelinePath(workspace, repoSlug, "pipelines")
	q := url.Values{}
	if params != nil {
		if params.Pagelen > 0 {
			q.Set("pagelen", fmt.Sprintf("%d", params.Pagelen))
		}
		if params.Page > 0 {
			q.Set("page", fmt.Sprintf("%d", params.Page))
		}
		if params.Sort != "" {
			q.Set("sort", params.Sort)
		}
		if params.Fields != "" {
			q.Set("fields", params.Fields)
		}
		if params.Filter != "" {
			q.Set("q", params.Filter)
		}
	}
	path := basePath
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var result PaginatedPipelines
	if err := c.doGet(path, &result); err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}
	return &result, nil
}

// ListPipelinesNext fetches the next page of pipelines from a paginated result,
// ensuring the sort parameter from the original request is preserved.
func (c *Client) ListPipelinesNext(result *PaginatedPipelines, params *ListPipelinesParams) (*PaginatedPipelines, error) {
	if result.Next == "" {
		return nil, nil
	}
	nextURL := result.Next
	// Bitbucket next URLs may not preserve query parameters; explicitly add sort if provided.
	if params != nil && params.Sort != "" && !strings.Contains(nextURL, "sort=") {
		if strings.Contains(nextURL, "?") {
			nextURL += "&sort=" + url.QueryEscape(params.Sort)
		} else {
			nextURL += "?sort=" + url.QueryEscape(params.Sort)
		}
	}
	var next PaginatedPipelines
	if err := c.GetFullURL(nextURL, &next); err != nil {
		return nil, fmt.Errorf("fetch next page: %w", err)
	}
	return &next, nil
}

// GetPipeline fetches a single pipeline by UUID.
func (c *Client) GetPipeline(workspace, repoSlug, pipelineUUID string) (*Pipeline, error) {
	path := BuildPipelinePath(workspace, repoSlug, "pipelines/"+url.PathEscape(pipelineUUID))
	var result Pipeline
	if err := c.doGet(path, &result); err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}
	return &result, nil
}

// ListPipelineSteps fetches all steps for a given pipeline.
func (c *Client) ListPipelineSteps(workspace, repoSlug, pipelineUUID string) (*PaginatedSteps, error) {
	path := BuildPipelinePath(workspace, repoSlug, "pipelines/"+url.PathEscape(pipelineUUID)+"/steps")
	var result PaginatedSteps
	if err := c.doGet(path, &result); err != nil {
		return nil, fmt.Errorf("list steps: %w", err)
	}
	return &result, nil
}

// GetStepLog fetches the log content for a specific step.
// The Bitbucket API returns the log as plain text, not JSON.
func (c *Client) GetStepLog(workspace, repoSlug, pipelineUUID, stepUUID string) (string, error) {
	path := BuildPipelinePath(workspace, repoSlug,
		"pipelines/"+url.PathEscape(pipelineUUID)+"/steps/"+url.PathEscape(stepUUID)+"/log")
	logContent, err := c.doGetText(path)
	if err != nil {
		return "", fmt.Errorf("get step log: %w", err)
	}
	return logContent, nil
}

// TriggerPipeline triggers a new pipeline run for the given workspace/repo.
func (c *Client) TriggerPipeline(workspace, repoSlug string, req TriggerPipelineRequest) (*Pipeline, error) {
	path := BuildPipelinePath(workspace, repoSlug, "pipelines")
	var result Pipeline
	if err := c.doPost(path, req, &result); err != nil {
		return nil, fmt.Errorf("trigger pipeline: %w", err)
	}
	return &result, nil
}

// ListPipelineVariables fetches repository-level pipeline config variables.
func (c *Client) ListPipelineVariables(workspace, repoSlug string) (*PaginatedPipelineVariables, error) {
	path := BuildPipelinePath(workspace, repoSlug, "pipelines_config/variables")
	var result PaginatedPipelineVariables
	if err := c.doGet(path, &result); err != nil {
		return nil, fmt.Errorf("list pipeline variables: %w", err)
	}
	return &result, nil
}

// GetPipelineStep fetches a single step for a pipeline.
func (c *Client) GetPipelineStep(workspace, repoSlug, pipelineUUID, stepUUID string) (*PipelineStep, error) {
	path := BuildPipelinePath(workspace, repoSlug,
		"pipelines/"+url.PathEscape(pipelineUUID)+"/steps/"+url.PathEscape(stepUUID))
	var step PipelineStep
	if err := c.doGet(path, &step); err != nil {
		return nil, fmt.Errorf("get pipeline step: %w", err)
	}
	return &step, nil
}

// StopPipeline stops a running pipeline.
func (c *Client) StopPipeline(workspace, repoSlug, pipelineUUID string) error {
	path := BuildPipelinePath(workspace, repoSlug, "pipelines/"+url.PathEscape(pipelineUUID)+"/stopPipeline")
	if err := c.doPost(path, nil, nil); err != nil {
		return fmt.Errorf("stop pipeline: %w", err)
	}
	return nil
}
