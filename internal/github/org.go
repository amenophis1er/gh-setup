package github

import (
	"net/http"

	gh "github.com/google/go-github/v68/github"
)

// GetOrg fetches an organization by name. Returns nil if not found.
func (c *Client) GetOrg(name string) (*gh.Organization, error) {
	org, resp, err := c.Organizations.Get(c.ctx, name)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	return org, nil
}

// IsOrganization checks if the given name is an organization.
func (c *Client) IsOrganization(name string) (bool, error) {
	org, err := c.GetOrg(name)
	if err != nil {
		return false, err
	}
	return org != nil, nil
}

// CreateOrg is a placeholder — GitHub API doesn't support org creation via REST.
// Organizations must be created manually via github.com.
func (c *Client) CreateOrg(_ string) error {
	return nil
}

// UpdateOrgSettings updates organization-level settings.
func (c *Client) UpdateOrgSettings(name string, org *gh.Organization) (*gh.Organization, error) {
	updated, _, err := c.Organizations.Edit(c.ctx, name, org)
	return updated, err
}
