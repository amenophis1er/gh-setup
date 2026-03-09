package github

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/nacl/box"

	gh "github.com/google/go-github/v68/github"
)

// SetOrgSecret creates or updates an organization-level secret.
func (c *Client) SetOrgSecret(org, name, value string) error {
	key, keyID, err := c.getOrgPublicKey(org)
	if err != nil {
		return fmt.Errorf("getting org public key: %w", err)
	}

	encrypted, err := encryptSecret(key, value)
	if err != nil {
		return fmt.Errorf("encrypting secret: %w", err)
	}

	_, err = c.Actions.CreateOrUpdateOrgSecret(c.ctx, org, &gh.EncryptedSecret{
		Name:           name,
		KeyID:          keyID,
		EncryptedValue: encrypted,
		Visibility:     "all",
	})
	return err
}

// SetRepoSecret creates or updates a repository-level secret.
func (c *Client) SetRepoSecret(owner, repo, name, value string) error {
	key, keyID, err := c.getRepoPublicKey(owner, repo)
	if err != nil {
		return fmt.Errorf("getting repo public key: %w", err)
	}

	encrypted, err := encryptSecret(key, value)
	if err != nil {
		return fmt.Errorf("encrypting secret: %w", err)
	}

	_, err = c.Actions.CreateOrUpdateRepoSecret(c.ctx, owner, repo, &gh.EncryptedSecret{
		Name:           name,
		KeyID:          keyID,
		EncryptedValue: encrypted,
	})
	return err
}

func (c *Client) getOrgPublicKey(org string) (string, string, error) {
	key, _, err := c.Actions.GetOrgPublicKey(c.ctx, org)
	if err != nil {
		return "", "", err
	}
	return key.GetKey(), key.GetKeyID(), nil
}

func (c *Client) getRepoPublicKey(owner, repo string) (string, string, error) {
	key, _, err := c.Actions.GetRepoPublicKey(c.ctx, owner, repo)
	if err != nil {
		return "", "", err
	}
	return key.GetKey(), key.GetKeyID(), nil
}

func encryptSecret(publicKey, secretValue string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return "", fmt.Errorf("decoding public key: %w", err)
	}

	var recipientKey [32]byte
	copy(recipientKey[:], decoded)

	encrypted, err := box.SealAnonymous(nil, []byte(secretValue), &recipientKey, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("encrypting: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}
