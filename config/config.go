package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Project represents a Bitbucket workspace and repository pair.
type Project struct {
	Workspace string `yaml:"workspace" json:"workspace"`
	RepoSlug  string `yaml:"repo_slug" json:"repo_slug"`
}

// Config holds the application configuration.
type Config struct {
	Username string    `yaml:"username"`
	AppPass  string    `yaml:"app_password"`
	Projects []Project `yaml:"projects"`
}

// DefaultPath returns the default config file path.
//
// The path can be overridden with the BBC_CONFIG environment variable; this is
// used by the container image, where the config is mounted at /config.yml.
func DefaultPath() (string, error) {
	if env := os.Getenv("BBC_CONFIG"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".config", "bbc-pipeline-tui.yml"), nil
}

// Exists checks if the config file exists at the default path.
func Exists() bool {
	path, err := DefaultPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Load reads the config from the default path.
func Load() (*Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads the config from the given path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to the default path.
func (c *Config) Save() error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

// SaveTo writes the config to the given path.
func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// AddProject appends a workspace/repo_slug pair if not already present.
// Returns true if the project was added.
func (c *Config) AddProject(workspace, repoSlug string) bool {
	if c.HasProject(workspace, repoSlug) {
		return false
	}
	c.Projects = append(c.Projects, Project{Workspace: workspace, RepoSlug: repoSlug})
	return true
}

// RemoveProject removes a workspace/repo_slug pair by matching both fields.
// Returns true if a project was removed.
func (c *Config) RemoveProject(workspace, repoSlug string) bool {
	for i, p := range c.Projects {
		if p.Workspace == workspace && p.RepoSlug == repoSlug {
			c.Projects = append(c.Projects[:i], c.Projects[i+1:]...)
			return true
		}
	}
	return false
}

// HasProject checks if a workspace/repo_slug pair exists in the config.
func (c *Config) HasProject(workspace, repoSlug string) bool {
	for _, p := range c.Projects {
		if p.Workspace == workspace && p.RepoSlug == repoSlug {
			return true
		}
	}
	return false
}

// Validate checks that the config has the required fields.
func (c *Config) Validate() error {
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	if c.AppPass == "" {
		return fmt.Errorf("app password is required")
	}
	if len(c.Projects) == 0 {
		return fmt.Errorf("at least one project is required")
	}
	for i, p := range c.Projects {
		if p.Workspace == "" {
			return fmt.Errorf("project %d: workspace is required", i+1)
		}
		if p.RepoSlug == "" {
			return fmt.Errorf("project %d: repo_slug is required", i+1)
		}
	}
	return nil
}
