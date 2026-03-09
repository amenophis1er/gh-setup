package github

import (
	"net/http"

	"github.com/amenophis1er/gh-setup/internal/config"
	gh "github.com/google/go-github/v68/github"
)

// GetBranchProtection fetches branch protection for a branch. Returns nil if not set.
func (c *Client) GetBranchProtection(owner, repo, branch string) (*gh.Protection, error) {
	prot, resp, err := c.Repositories.GetBranchProtection(c.ctx, owner, repo, branch)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	return prot, nil
}

// ApplyBranchProtection sets branch protection rules based on the config.
func (c *Client) ApplyBranchProtection(owner, repo, branch string, bp config.BranchProtection) error {
	req := buildProtectionRequest(bp)
	_, _, err := c.Repositories.UpdateBranchProtection(c.ctx, owner, repo, branch, req)
	return err
}

// RemoveBranchProtection removes all branch protection from a branch.
func (c *Client) RemoveBranchProtection(owner, repo, branch string) error {
	_, err := c.Repositories.RemoveBranchProtection(c.ctx, owner, repo, branch)
	return err
}

func buildProtectionRequest(bp config.BranchProtection) *gh.ProtectionRequest {
	req := &gh.ProtectionRequest{
		AllowForcePushes: gh.Ptr(bp.AllowForcePush),
		AllowDeletions:   gh.Ptr(bp.AllowDeletions),
	}

	if bp.RequirePR {
		req.RequiredPullRequestReviews = &gh.PullRequestReviewsEnforcementRequest{
			RequiredApprovingReviewCount: bp.RequiredApprovals,
			DismissStaleReviews:         bp.DismissStaleReviews,
		}
	}

	if bp.RequireStatusChecks {
		checks := make([]*gh.RequiredStatusCheck, len(bp.StatusChecks))
		for i, name := range bp.StatusChecks {
			checks[i] = &gh.RequiredStatusCheck{Context: name}
		}
		req.RequiredStatusChecks = &gh.RequiredStatusChecks{
			Strict: bp.RequireUpToDate,
			Checks: &checks,
		}
	}

	if bp.EnforceAdmins {
		req.EnforceAdmins = true
	}

	return req
}
