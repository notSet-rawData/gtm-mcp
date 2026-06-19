package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DestinationToolInput struct {
	Action                           string `json:"action" jsonschema:"enum:list,get,link,description:Operation to perform on destinations"`
	AccountID                        string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID                      string `json:"containerId" jsonschema:"description:The GTM container ID"`
	DestinationID                    string `json:"destinationId,omitempty" jsonschema:"description:Destination ID (required for get and link)"`
	AllowUserPermissionFeatureUpdate bool   `json:"allowUserPermissionFeatureUpdate,omitempty" jsonschema:"description:If true, allows user permission feature update during linking (optional for link)"`
}

type DestinationInfo struct {
	DestinationID string `json:"destinationId"`
	Name          string `json:"name,omitempty"`
	Path          string `json:"path,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	TagManagerUrl string `json:"tagManagerUrl,omitempty"`
}

type ListDestinationsOutput struct {
	Destinations []DestinationInfo `json:"destinations"`
}

type GetDestinationOutput struct {
	Destination DestinationInfo `json:"destination"`
}

type LinkDestinationOutput struct {
	Success     bool            `json:"success"`
	Destination DestinationInfo `json:"destination"`
	Message     string          `json:"message"`
}

func handleDestinationList(ctx context.Context, input DestinationToolInput) (*mcp.CallToolResult, any, error) {
	client, err := getClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := fmt.Sprintf("accounts/%s/containers/%s", input.AccountID, input.ContainerID)
	resp, err := client.Service.Accounts.Containers.Destinations.List(parent).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	destinations := make([]DestinationInfo, 0)
	if resp != nil && resp.Destination != nil {
		for _, d := range resp.Destination {
			destinations = append(destinations, DestinationInfo{
				DestinationID: d.DestinationId,
				Name:          d.Name,
				Path:          d.Path,
				Fingerprint:   d.Fingerprint,
				TagManagerUrl: d.TagManagerUrl,
			})
		}
	}

	return nil, ListDestinationsOutput{Destinations: destinations}, nil
}

func handleDestinationGet(ctx context.Context, input DestinationToolInput) (*mcp.CallToolResult, any, error) {
	if input.DestinationID == "" {
		return nil, nil, fmt.Errorf("destinationId is required for get action")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("accounts/%s/containers/%s/destinations/%s", input.AccountID, input.ContainerID, input.DestinationID)
	d, err := client.Service.Accounts.Containers.Destinations.Get(path).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	return nil, GetDestinationOutput{
		Destination: DestinationInfo{
			DestinationID: d.DestinationId,
			Name:          d.Name,
			Path:          d.Path,
			Fingerprint:   d.Fingerprint,
			TagManagerUrl: d.TagManagerUrl,
		},
	}, nil
}

func handleDestinationLink(ctx context.Context, input DestinationToolInput) (*mcp.CallToolResult, any, error) {
	if input.DestinationID == "" {
		return nil, nil, fmt.Errorf("destinationId is required for link action")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := fmt.Sprintf("accounts/%s/containers/%s", input.AccountID, input.ContainerID)
	call := client.Service.Accounts.Containers.Destinations.Link(parent).
		DestinationId(input.DestinationID).
		Context(tCtx)

	if input.AllowUserPermissionFeatureUpdate {
		call = call.AllowUserPermissionFeatureUpdate(true)
	}

	d, err := call.Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	return nil, LinkDestinationOutput{
		Success: true,
		Destination: DestinationInfo{
			DestinationID: d.DestinationId,
			Name:          d.Name,
			Path:          d.Path,
			Fingerprint:   d.Fingerprint,
			TagManagerUrl: d.TagManagerUrl,
		},
		Message: fmt.Sprintf("Destination %s linked successfully", input.DestinationID),
	}, nil
}
