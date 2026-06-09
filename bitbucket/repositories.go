package bitbucket

import (
	"fmt"
	"net/url"
)

// PaginatedRepositories is the paginated list of repositories.
type PaginatedRepositories struct {
	PaginatedResponse
	Values []Repository `json:"values"`
}

// ListRepositories fetches all repositories in a workspace.
func (c *Client) ListRepositories(workspace string) (*PaginatedRepositories, error) {
	path := fmt.Sprintf("/repositories/%s?pagelen=50", url.PathEscape(workspace))
	var result PaginatedRepositories
	if err := c.doGet(path, &result); err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	return &result, nil
}

// ListRepositoriesNext fetches the next page of repositories from a paginated result.
func (c *Client) ListRepositoriesNext(result *PaginatedRepositories) (*PaginatedRepositories, error) {
	if result.Next == "" {
		return nil, nil
	}
	var next PaginatedRepositories
	if err := c.GetFullURL(result.Next, &next); err != nil {
		return nil, fmt.Errorf("fetch next page: %w", err)
	}
	return &next, nil
}