package github

import (
	"net/http"

	gh "github.com/google/go-github/v68/github"
)

// GetTeam fetches a team by slug. Returns nil if not found.
func (c *Client) GetTeam(org, slug string) (*gh.Team, error) {
	team, resp, err := c.Teams.GetTeamBySlug(c.ctx, org, slug)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	return team, nil
}

// CreateTeam creates a new team in the organization.
func (c *Client) CreateTeam(org, name, description, permission string) (*gh.Team, error) {
	team, _, err := c.Teams.CreateTeam(c.ctx, org, gh.NewTeam{
		Name:        name,
		Description: gh.Ptr(description),
		Permission:  gh.Ptr(permission),
	})
	return team, err
}

// UpdateTeam updates a team's settings.
func (c *Client) UpdateTeam(org, slug, name, description, permission string) (*gh.Team, error) {
	team, _, err := c.Teams.EditTeamBySlug(c.ctx, org, slug, gh.NewTeam{
		Name:        name,
		Description: gh.Ptr(description),
		Permission:  gh.Ptr(permission),
	}, false)
	return team, err
}

// ListTeamMembers lists all members of a team.
func (c *Client) ListTeamMembers(org, slug string) ([]*gh.User, error) {
	var all []*gh.User
	opts := &gh.TeamListTeamMembersOptions{ListOptions: gh.ListOptions{PerPage: 100}}
	for {
		members, resp, err := c.Teams.ListTeamMembersBySlug(c.ctx, org, slug, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, members...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// ListOrgTeams lists all teams in an organization.
func (c *Client) ListOrgTeams(org string) ([]*gh.Team, error) {
	var all []*gh.Team
	opts := &gh.ListOptions{PerPage: 100}
	for {
		teams, resp, err := c.Teams.ListTeams(c.ctx, org, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, teams...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// AddTeamMember adds a user to a team.
func (c *Client) AddTeamMember(org, slug, username string) error {
	_, _, err := c.Teams.AddTeamMembershipBySlug(c.ctx, org, slug, username, nil)
	return err
}

// RemoveTeamMember removes a user from a team.
func (c *Client) RemoveTeamMember(org, slug, username string) error {
	_, err := c.Teams.RemoveTeamMembershipBySlug(c.ctx, org, slug, username)
	return err
}
