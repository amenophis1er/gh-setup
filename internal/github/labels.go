package github

import (
	"github.com/amenophis1er/gh-setup/internal/config"
	gh "github.com/google/go-github/v68/github"
)

// ListLabels fetches all labels for a repository.
func (c *Client) ListLabels(owner, repo string) ([]*gh.Label, error) {
	var all []*gh.Label
	opts := &gh.ListOptions{PerPage: 100}
	for {
		labels, resp, err := c.Issues.ListLabels(c.ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, labels...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// DeleteLabel deletes a label from a repository.
func (c *Client) DeleteLabel(owner, repo, name string) error {
	_, err := c.Issues.DeleteLabel(c.ctx, owner, repo, name)
	return err
}

// CreateLabel creates a label in a repository.
func (c *Client) CreateLabel(owner, repo string, label config.Label) error {
	_, _, err := c.Issues.CreateLabel(c.ctx, owner, repo, &gh.Label{
		Name:        gh.Ptr(label.Name),
		Color:       gh.Ptr(label.Color),
		Description: gh.Ptr(label.Description),
	})
	return err
}

// UpdateLabel updates an existing label.
func (c *Client) UpdateLabel(owner, repo, currentName string, label config.Label) error {
	_, _, err := c.Issues.EditLabel(c.ctx, owner, repo, currentName, &gh.Label{
		Name:        gh.Ptr(label.Name),
		Color:       gh.Ptr(label.Color),
		Description: gh.Ptr(label.Description),
	})
	return err
}
