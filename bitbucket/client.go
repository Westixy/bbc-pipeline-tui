package bitbucket

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	BaseURL    = "https://api.bitbucket.org/2.0"
	DefaultUA  = "bbc-pipeline-tui/1.0"
	TimeoutSec = 30
)

// Client handles authenticated requests to the Bitbucket API.
type Client struct {
	Username  string
	AppPass   string
	BaseURL   string
	UserAgent string
	hc        *http.Client
}

// NewClient creates a new Bitbucket API client with Basic Auth.
func NewClient(username, appPassword string) *Client {
	return &Client{
		Username:  username,
		AppPass:   appPassword,
		BaseURL:   BaseURL,
		UserAgent: DefaultUA,
		hc:        &http.Client{Timeout: TimeoutSec * time.Second},
	}
}

// authHeader returns the Basic Auth header value.
func (c *Client) authHeader() string {
	val := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.AppPass))
	return "Basic " + val
}

// doRequest performs an HTTP request and decodes the JSON response into dest.
func (c *Client) doRequest(method, path string, body io.Reader, dest interface{}) error {
	reqURL := c.BaseURL + path
	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("User-Agent", c.UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var apiErr APIError
		_ = json.Unmarshal(respBody, &apiErr)
		if apiErr.ErrorMsg.Message != "" {
			return fmt.Errorf("API error (%d): %s - %s", resp.StatusCode, apiErr.ErrorMsg.Key, apiErr.ErrorMsg.Message)
		}
		return fmt.Errorf("API error: %s (status %d)", strings.TrimSpace(string(respBody)), resp.StatusCode)
	}
	if dest != nil {
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}
		if err := json.Unmarshal(raw, dest); err != nil {
			return fmt.Errorf("decode response: %w\nbody: %s", err, string(raw[:min(len(raw), 500)]))
		}
	}
	return nil
}

// doGet performs a GET request and decodes the JSON into dest.
func (c *Client) doGet(path string, dest interface{}) error {
	return c.doRequest("GET", path, nil, dest)
}

// doGetText performs a GET request and returns the raw response body as a string.
func (c *Client) doGetText(path string) (string, error) {
	reqURL := c.BaseURL + path
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("API error: %s (status %d)", strings.TrimSpace(string(respBody)), resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	return string(raw), nil
}

// doPost performs a POST request with a JSON body and decodes the response into dest.
func (c *Client) doPost(path string, body interface{}, dest interface{}) error {
	var r io.Reader
	var reqBody string
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reqBody = string(data)
		r = strings.NewReader(reqBody)
	}
	err := c.doRequest("POST", path, r, dest)
	if err != nil && reqBody != "" {
		return fmt.Errorf("%w\nrequest body: %s", err, reqBody)
	}
	return err
}

// GetFullURL fetches a full URL (used for pagination next/prev links).
func (c *Client) GetFullURL(fullURL string, dest interface{}) error {
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("API error: %s (status %d)", strings.TrimSpace(string(respBody)), resp.StatusCode)
	}
	if dest != nil {
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}
		if err := json.Unmarshal(raw, dest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// APIError represents a Bitbucket API error response.
type APIError struct {
	ErrorMsg struct {
		Key     string `json:"key"`
		Message string `json:"message"`
	} `json:"error"`
	Type string `json:"type"`
}

// PaginatedResponse is the common pagination wrapper.
type PaginatedResponse struct {
	Page    *int    `json:"page"`
	Pagelen int     `json:"pagelen"`
	Size    *int    `json:"size"`
	Next    string  `json:"next"`
	Prev    string  `json:"previous"`
}

// PaginatedPipelines is the paginated list of pipelines.
type PaginatedPipelines struct {
	PaginatedResponse
	Values []Pipeline `json:"values"`
}

// PaginatedSteps is the paginated list of pipeline steps.
type PaginatedSteps struct {
	PaginatedResponse
	Values []PipelineStep `json:"values"`
}

// PaginatedPipelineVariables is the paginated list of pipeline config variables.
type PaginatedPipelineVariables struct {
	PaginatedResponse
	Values []PipelineVariable `json:"values"`
}

// Object is the base object embedded in all Bitbucket resources.
type Object struct {
	Type string `json:"type"`
}

// Account represents a Bitbucket user/account.
type Account struct {
	Object
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name"`
	CreatedOn   string `json:"created_on"`
}

// Repository represents a Bitbucket repository.
type Repository struct {
	Object
	UUID        string  `json:"uuid"`
	FullName    string  `json:"full_name"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	IsPrivate   bool    `json:"is_private"`
	SCM         string  `json:"scm"`
	Owner       Account `json:"owner"`
}

// Commit represents a git commit reference.
type Commit struct {
	Hash  string `json:"hash"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// PipelineSelector represents a pipeline definition selector (custom/default).
type PipelineSelector struct {
	Type    string `json:"type"`
	Pattern string `json:"pattern"`
}

// PipelineState holds the pipeline state info.
type PipelineState struct {
	Name   string `json:"name"`
	Result *struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"result,omitempty"`
	Stage *struct {
		Name string `json:"name"`
	} `json:"stage,omitempty"`
}

// PipelineTrigger represents what triggered the pipeline.
type PipelineTrigger struct {
	Name string `json:"name"`
}

// PipelineVariable is a variable defined on a pipeline run.
type PipelineVariable struct {
	UUID    string `json:"uuid,omitempty"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Secured bool   `json:"secured,omitempty"`
}

// PipelineTarget holds the target (branch, commit, selector) of the pipeline.
type PipelineTarget struct {
	Type     string                  `json:"type"`
	RefType  string                  `json:"ref_type,omitempty"`
	RefName  string                  `json:"ref_name,omitempty"`
	Commit   *Commit                 `json:"commit,omitempty"`
	Selector *PipelineSelector       `json:"selector,omitempty"`
}

// PipelineCommand is a command in a step.
type PipelineCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// PipelineImage is the Docker image used in a step.
type PipelineImage struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// PipelineStep represents a step within a pipeline.
type PipelineStep struct {
	Object
	UUID           string            `json:"uuid"`
	StartedOn      string            `json:"started_on"`
	CompletedOn    string            `json:"completed_on"`
	State          PipelineStepState `json:"state"`
	Image          PipelineImage     `json:"image"`
	SetupCommands  []PipelineCommand `json:"setup_commands"`
	ScriptCommands []PipelineCommand `json:"script_commands"`
	Name           string            `json:"name"`
	// Helper field populated after fetching
	MaxTime *int `json:"max_time,omitempty"`
	// Duration in seconds, computed
	Duration int `json:"duration"`
}

// PipelineStepState is the state of a pipeline step.
type PipelineStepState struct {
	Name   string `json:"name"`
	Result *struct {
		Name string `json:"name"`
	} `json:"result,omitempty"`
}

// Pipeline represents a Bitbucket pipeline run.
type Pipeline struct {
	Object
	UUID                 string            `json:"uuid"`
	BuildNumber          int               `json:"build_number"`
	Creator              *Account          `json:"creator"`
	Repository           *Repository       `json:"repository"`
	Target               PipelineTarget    `json:"target"`
	Trigger              PipelineTrigger   `json:"trigger"`
	State                PipelineState     `json:"state"`
	Variables            []PipelineVariable `json:"variables"`
	CreatedOn            string            `json:"created_on"`
	CompletedOn          string            `json:"completed_on"`
	BuildSecondsUsed     int               `json:"build_seconds_used"`
}

// ListPipelinesParams holds optional query parameters for listing pipelines.
type ListPipelinesParams struct {
	Pagelen  int
	Page     int
	Sort     string // e.g., "-created_on"
	Fields   string
}

// BuildPipelinePath builds the API path for a workspace/repo resource.
func BuildPipelinePath(workspace, repoSlug, resource string) string {
	return fmt.Sprintf("/repositories/%s/%s/%s", url.PathEscape(workspace), url.PathEscape(repoSlug), resource)
}

// TriggerPipelineRequest is the request body for triggering a new pipeline.
type TriggerPipelineRequest struct {
	Target    PipelineTarget     `json:"target"`
	Variables []PipelineVariable `json:"variables,omitempty"`
}