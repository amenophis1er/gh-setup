package github

import (
	"net/http"

	gh "github.com/google/go-github/v68/github"
)

// GetRepo fetches a repository. Returns nil if not found.
func (c *Client) GetRepo(owner, name string) (*gh.Repository, error) {
	repo, resp, err := c.Repositories.Get(c.ctx, owner, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	return repo, nil
}

// CreateRepo creates a new repository under the given owner.
// If owner is an org, it creates under the org; otherwise under the authenticated user.
func (c *Client) CreateRepo(owner string, isOrg bool, repo *gh.Repository) (*gh.Repository, error) {
	var created *gh.Repository
	var err error

	if isOrg {
		created, _, err = c.Repositories.Create(c.ctx, owner, repo)
	} else {
		created, _, err = c.Repositories.Create(c.ctx, "", repo)
	}
	return created, err
}

// UpdateRepo updates an existing repository's settings.
func (c *Client) UpdateRepo(owner, name string, repo *gh.Repository) (*gh.Repository, error) {
	updated, _, err := c.Repositories.Edit(c.ctx, owner, name, repo)
	return updated, err
}

// SetTopics replaces the topics on a repository.
func (c *Client) SetTopics(owner, name string, topics []string) error {
	_, _, err := c.Repositories.ReplaceAllTopics(c.ctx, owner, name, topics)
	return err
}

// CreateOrUpdateFile creates or updates a file in a repository.
func (c *Client) CreateOrUpdateFile(owner, repo, path, message string, content []byte, sha *string) error {
	opts := &gh.RepositoryContentFileOptions{
		Message: gh.Ptr(message),
		Content: content,
	}
	if sha != nil {
		opts.SHA = sha
	}
	_, _, err := c.Repositories.CreateFile(c.ctx, owner, repo, path, opts)
	return err
}

// GetFileContent retrieves file content and its SHA. Returns empty string and nil SHA if not found.
func (c *Client) GetFileContent(owner, repo, path string) (string, *string, error) {
	file, _, resp, err := c.Repositories.GetContents(c.ctx, owner, repo, path, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return "", nil, nil
		}
		return "", nil, err
	}
	if file == nil {
		return "", nil, nil
	}
	content, err := file.GetContent()
	if err != nil {
		return "", nil, err
	}
	return content, file.SHA, nil
}
