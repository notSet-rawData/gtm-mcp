package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

type Trigger struct {
	TriggerID          string `json:"triggerId"`
	Name               string `json:"name"`
	Type               string `json:"type"`
	Path               string `json:"path"`
	ParentFolderID     string `json:"parentFolderId,omitempty"`
	Notes              string `json:"notes,omitempty"`
	Fingerprint        string `json:"fingerprint,omitempty"`
	EventName          any    `json:"eventName,omitempty"`
	WaitForTags        any    `json:"waitForTags,omitempty"`
	CheckValidation    any    `json:"checkValidation,omitempty"`
	WaitForTagsTimeout any    `json:"waitForTagsTimeout,omitempty"`
	UniqueTriggerId    any    `json:"uniqueTriggerId,omitempty"`
	Filter             any    `json:"filter,omitempty"`            // For pageview triggers
	AutoEventFilter    any    `json:"autoEventFilter,omitempty"`   // For click/form triggers
	CustomEventFilter  any    `json:"customEventFilter,omitempty"` // For customEvent triggers
	Parameter          any    `json:"parameter,omitempty"`
}

func (c *Client) ListTriggers(ctx context.Context, accountID, containerID, workspaceID string) ([]Trigger, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListTriggersResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.Triggers.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return toTriggers(resp.Trigger), nil
}

func (c *Client) GetTrigger(ctx context.Context, accountID, containerID, workspaceID, triggerID string) (*Trigger, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/triggers/%s",
		accountID, containerID, workspaceID, triggerID)

	t, err := retryWithBackoff(ctx, 3, func() (*tagmanager.Trigger, error) {
		return c.Service.Accounts.Containers.Workspaces.Triggers.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	triggers := toTriggers([]*tagmanager.Trigger{t})
	return &triggers[0], nil
}

func toTriggers(triggers []*tagmanager.Trigger) []Trigger {
	result := make([]Trigger, 0, len(triggers))
	for _, t := range triggers {
		trigger := Trigger{
			TriggerID:      t.TriggerId,
			Name:           t.Name,
			Type:           t.Type,
			Path:           t.Path,
			ParentFolderID: t.ParentFolderId,
			Notes:          t.Notes,
			Fingerprint:    t.Fingerprint,
		}
		if t.EventName != nil {
			trigger.EventName = t.EventName
		}
		if t.WaitForTags != nil {
			trigger.WaitForTags = t.WaitForTags
		}
		if t.CheckValidation != nil {
			trigger.CheckValidation = t.CheckValidation
		}
		if t.WaitForTagsTimeout != nil {
			trigger.WaitForTagsTimeout = t.WaitForTagsTimeout
		}
		if t.UniqueTriggerId != nil {
			trigger.UniqueTriggerId = t.UniqueTriggerId
		}
		if len(t.Filter) > 0 {
			trigger.Filter = t.Filter
		}
		if len(t.AutoEventFilter) > 0 {
			trigger.AutoEventFilter = t.AutoEventFilter
		}
		if len(t.CustomEventFilter) > 0 {
			trigger.CustomEventFilter = t.CustomEventFilter
		}
		if len(t.Parameter) > 0 {
			trigger.Parameter = t.Parameter
		}
		result = append(result, trigger)
	}
	return result
}
