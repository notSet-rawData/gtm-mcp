package gtm

import (
	"context"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

type Account struct {
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}

func (c *Client) ListAccounts(ctx context.Context) ([]Account, error) {
	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListAccountsResponse, error) {
		return c.Service.Accounts.List().Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return toAccounts(resp.Account), nil
}

func toAccounts(accounts []*tagmanager.Account) []Account {
	result := make([]Account, 0, len(accounts))
	for _, a := range accounts {
		result = append(result, Account{
			AccountID: a.AccountId,
			Name:      a.Name,
			Path:      a.Path,
		})
	}
	return result
}
