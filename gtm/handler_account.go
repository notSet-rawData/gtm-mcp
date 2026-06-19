package gtm

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AccountToolInput struct {
	Action string `json:"action" jsonschema:"enum:list,description:Operation to perform on accounts"`
}

func handleListAccounts(ctx context.Context) (*mcp.CallToolResult, any, error) {
	client, err := getClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	accounts, err := client.ListAccounts(ctx)
	if err != nil {
		return nil, nil, err
	}

	return nil, ListAccountsOutput{Accounts: accounts}, nil
}
