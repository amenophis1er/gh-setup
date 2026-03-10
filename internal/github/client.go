package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	gh "github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// Client wraps the go-github client with convenience methods.
type Client struct {
	*gh.Client
	ctx context.Context
}

// NewClient creates an authenticated GitHub client.
// It reads the token from GITHUB_TOKEN or GH_TOKEN environment variables.
func NewClient() (*Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		// Fall back to gh CLI auth token (available when running as a gh extension)
		if out, err := exec.Command("gh", "auth", "token").Output(); err == nil {
			token = strings.TrimSpace(string(out))
		}
	}
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN or GH_TOKEN environment variable must be set, or authenticate via: gh auth login")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		Client: gh.NewClient(tc),
		ctx:    ctx,
	}, nil
}

// Ctx returns the client's context.
func (c *Client) Ctx() context.Context {
	return c.ctx
}
