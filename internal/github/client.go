package github

import (
	"context"
	"fmt"
	"os"

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
		return nil, fmt.Errorf("GITHUB_TOKEN or GH_TOKEN environment variable must be set")
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
