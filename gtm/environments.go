package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Environment is a simplified representation of a GTM environment.
type Environment struct {
	EnvironmentID    string `json:"environmentId"`
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	Type             string `json:"type"` // user, live, latest, workspace
	ContainerVersionID string `json:"containerVersionId,omitempty"`
	URL              string `json:"url,omitempty"`
	AuthorizationCode string `json:"authorizationCode,omitempty"`
	Path             string `json:"path"`
	Fingerprint      string `json:"fingerprint,omitempty"`
}

// ListEnvironments returns all environments in a container.
func (c *Client) ListEnvironments(ctx context.Context, accountID, containerID string) ([]Environment, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s", accountID, containerID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListEnvironmentsResponse, error) {
		return c.Service.Accounts.Containers.Environments.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return toEnvironments(resp.Environment), nil
}

// GetEnvironment returns a specific environment by ID.
func (c *Client) GetEnvironment(ctx context.Context, accountID, containerID, environmentID string) (*Environment, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/environments/%s",
		accountID, containerID, environmentID)

	env, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Environment, error) {
		return c.Service.Accounts.Containers.Environments.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	result := toEnvironment(env)
	return &result, nil
}

func toEnvironments(environments []*tagmanager.Environment) []Environment {
	result := make([]Environment, 0, len(environments))
	for _, e := range environments {
		result = append(result, toEnvironment(e))
	}
	return result
}

// CreateEnvironment creates a new user-defined environment in a container.
func (c *Client) CreateEnvironment(ctx context.Context, accountID, containerID string, input *EnvironmentInput) (*Environment, error) {
	parent := BuildContainerPath(accountID, containerID)

	env := &tagmanager.Environment{
		Name:               input.Name,
		Description:        input.Description,
		ContainerVersionId: input.ContainerVersionID,
	}
	if input.EnableDebug {
		env.EnableDebug = true
	}

	result, err := c.Service.Accounts.Containers.Environments.Create(parent, env).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	r := toEnvironment(result)
	return &r, nil
}

// UpdateEnvironment updates an existing environment. Fetches current fingerprint first.
func (c *Client) UpdateEnvironment(ctx context.Context, path string, input *EnvironmentInput) (*Environment, error) {
	current, err := c.Service.Accounts.Containers.Environments.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	env := &tagmanager.Environment{
		Name:               input.Name,
		Description:        input.Description,
		ContainerVersionId: input.ContainerVersionID,
		EnableDebug:        input.EnableDebug,
	}

	result, err := c.Service.Accounts.Containers.Environments.Update(path, env).Fingerprint(current.Fingerprint).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	r := toEnvironment(result)
	return &r, nil
}

// DeleteEnvironment deletes an environment from a container.
func (c *Client) DeleteEnvironment(ctx context.Context, path string) error {
	err := c.Service.Accounts.Containers.Environments.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

// ReauthorizeEnvironment generates a new authorization code for the environment.
func (c *Client) ReauthorizeEnvironment(ctx context.Context, path string) (*Environment, error) {
	current, err := c.Service.Accounts.Containers.Environments.Get(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	result, err := c.Service.Accounts.Containers.Environments.Reauthorize(path, current).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	r := toEnvironment(result)
	return &r, nil
}

func toEnvironment(e *tagmanager.Environment) Environment {
	return Environment{
		EnvironmentID:      e.EnvironmentId,
		Name:               e.Name,
		Description:        e.Description,
		Type:               e.Type,
		ContainerVersionID: e.ContainerVersionId,
		URL:                e.Url,
		AuthorizationCode:  e.AuthorizationCode,
		Path:               e.Path,
		Fingerprint:        e.Fingerprint,
	}
}
