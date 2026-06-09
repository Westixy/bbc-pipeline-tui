package bitbucket

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
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

// ListAllRepositories fetches every repository in a workspace by following
// pagination links until exhausted.
func (c *Client) ListAllRepositories(workspace string) ([]Repository, error) {
	path := fmt.Sprintf("/repositories/%s?pagelen=100", url.PathEscape(workspace))
	var page PaginatedRepositories
	if err := c.doGet(path, &page); err != nil {
		return nil, fmt.Errorf("list all repositories: %w", err)
	}
	all := append([]Repository{}, page.Values...)
	for page.Next != "" {
		var next PaginatedRepositories
		if err := c.GetFullURL(page.Next, &next); err != nil {
			return nil, fmt.Errorf("list all repositories next page: %w", err)
		}
		all = append(all, next.Values...)
		page = next
	}
	// Sort alphabetically by name/slug (case-insensitive)
	sort.Slice(all, func(i, j int) bool {
		pi := strings.ToLower(all[i].Name)
		pj := strings.ToLower(all[j].Name)
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(all[i].FullName) < strings.ToLower(all[j].FullName)
	})
	return all, nil
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
