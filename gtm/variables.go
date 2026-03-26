package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Variable is a simplified representation of a GTM variable.
type Variable struct {
	VariableID         string   `json:"variableId"`
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	Path               string   `json:"path"`
	Notes              string   `json:"notes,omitempty"`
	ParentFolderID     string   `json:"parentFolderId,omitempty"`
	Fingerprint        string   `json:"fingerprint,omitempty"`
	ScheduleStartMs    int64    `json:"scheduleStartMs,omitempty"`
	ScheduleEndMs      int64    `json:"scheduleEndMs,omitempty"`
	EnablingTriggerId  []string `json:"enablingTriggerId,omitempty"`
	DisablingTriggerId []string `json:"disablingTriggerId,omitempty"`
	FormatValue        any      `json:"formatValue,omitempty"`
	// Parameter contains variable configuration (lookup tables, JavaScript code, data layer variable names, etc.).
	Parameter any `json:"parameter,omitempty"`
}

// ListVariables returns all variables in a workspace.
func (c *Client) ListVariables(ctx context.Context, accountID, containerID, workspaceID string) ([]Variable, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListVariablesResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.Variables.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return toVariables(resp.Variable), nil
}

// GetVariable returns a specific variable by ID.
func (c *Client) GetVariable(ctx context.Context, accountID, containerID, workspaceID, variableID string) (*Variable, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/variables/%s",
		accountID, containerID, workspaceID, variableID)

	v, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Variable, error) {
		return c.Service.Accounts.Containers.Workspaces.Variables.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	result := toVariable(v)
	return &result, nil
}

func toVariable(v *tagmanager.Variable) Variable {
	variable := Variable{
		VariableID:         v.VariableId,
		Name:               v.Name,
		Type:               v.Type,
		Path:               v.Path,
		Notes:              v.Notes,
		ParentFolderID:     v.ParentFolderId,
		Fingerprint:        v.Fingerprint,
		ScheduleStartMs:    v.ScheduleStartMs,
		ScheduleEndMs:      v.ScheduleEndMs,
		EnablingTriggerId:  v.EnablingTriggerId,
		DisablingTriggerId: v.DisablingTriggerId,
	}
	if v.FormatValue != nil {
		variable.FormatValue = v.FormatValue
	}
	if len(v.Parameter) > 0 {
		variable.Parameter = v.Parameter
	}
	return variable
}

func toVariables(variables []*tagmanager.Variable) []Variable {
	result := make([]Variable, 0, len(variables))
	for _, v := range variables {
		result = append(result, toVariable(v))
	}
	return result
}
