package github

import (
	gh "github.com/google/go-github/v68/github"
)

// EnableVulnerabilityAlerts enables Dependabot vulnerability alerts on a repo.
func (c *Client) EnableVulnerabilityAlerts(owner, repo string) error {
	_, err := c.Repositories.EnableVulnerabilityAlerts(c.ctx, owner, repo)
	return err
}

// DisableVulnerabilityAlerts disables Dependabot vulnerability alerts on a repo.
func (c *Client) DisableVulnerabilityAlerts(owner, repo string) error {
	_, err := c.Repositories.DisableVulnerabilityAlerts(c.ctx, owner, repo)
	return err
}

// GetVulnerabilityAlerts checks if Dependabot vulnerability alerts are enabled.
func (c *Client) GetVulnerabilityAlerts(owner, repo string) (bool, error) {
	enabled, _, err := c.Repositories.GetVulnerabilityAlerts(c.ctx, owner, repo)
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// UpdateSecurityAndAnalysis updates security features (secret scanning, etc.) on a repo.
func (c *Client) UpdateSecurityAndAnalysis(owner, repoName string, secretScanning, codeScanningEnabled bool) error {
	securityAndAnalysis := &gh.SecurityAndAnalysis{}

	if secretScanning {
		securityAndAnalysis.SecretScanning = &gh.SecretScanning{Status: gh.Ptr("enabled")}
		securityAndAnalysis.SecretScanningPushProtection = &gh.SecretScanningPushProtection{Status: gh.Ptr("enabled")}
	} else {
		securityAndAnalysis.SecretScanning = &gh.SecretScanning{Status: gh.Ptr("disabled")}
		securityAndAnalysis.SecretScanningPushProtection = &gh.SecretScanningPushProtection{Status: gh.Ptr("disabled")}
	}

	if codeScanningEnabled {
		securityAndAnalysis.AdvancedSecurity = &gh.AdvancedSecurity{Status: gh.Ptr("enabled")}
	}

	_, _, err := c.Repositories.Edit(c.ctx, owner, repoName, &gh.Repository{
		SecurityAndAnalysis: securityAndAnalysis,
	})
	return err
}
