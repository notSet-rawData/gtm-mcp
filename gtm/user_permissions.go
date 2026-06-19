package gtm

import (
	"context"
	"fmt"
	"strings"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

func (c *Client) ListUserPermissions(ctx context.Context, accountID string) ([]UserPermission, error) {
	parent := fmt.Sprintf("accounts/%s", accountID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListUserPermissionsResponse, error) {
		return c.Service.Accounts.UserPermissions.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}
	if resp == nil {
		return []UserPermission{}, nil
	}

	return toUserPermissions(resp.UserPermission), nil
}

func (c *Client) GetUserPermission(ctx context.Context, accountID, permissionID string) (*UserPermission, error) {
	path := BuildUserPermissionPath(accountID, permissionID)

	perm, err := retryWithBackoff(ctx, 3, func() (*tagmanager.UserPermission, error) {
		return c.Service.Accounts.UserPermissions.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	result := toUserPermission(perm)
	return &result, nil
}

func (c *Client) CreateUserPermission(ctx context.Context, accountID string, input *UserPermissionInput) (*UserPermission, error) {
	parent := fmt.Sprintf("accounts/%s", accountID)

	perm := &tagmanager.UserPermission{
		EmailAddress:    input.EmailAddress,
		AccountAccess:   toAPIAccountAccess(input.AccountAccess),
		ContainerAccess: toAPIContainerAccess(input.ContainerAccess),
	}

	result, err := c.Service.Accounts.UserPermissions.Create(parent, perm).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	r := toUserPermission(result)
	return &r, nil
}

func (c *Client) UpdateUserPermission(ctx context.Context, path string, input *UserPermissionInput) (*UserPermission, error) {
	perm := &tagmanager.UserPermission{
		EmailAddress:    input.EmailAddress,
		AccountAccess:   toAPIAccountAccess(input.AccountAccess),
		ContainerAccess: toAPIContainerAccess(input.ContainerAccess),
	}

	result, err := c.Service.Accounts.UserPermissions.Update(path, perm).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	r := toUserPermission(result)
	return &r, nil
}

func (c *Client) DeleteUserPermission(ctx context.Context, path string) error {
	err := c.Service.Accounts.UserPermissions.Delete(path).Context(ctx).Do()
	return mapGoogleError(err)
}

func toUserPermissions(perms []*tagmanager.UserPermission) []UserPermission {
	result := make([]UserPermission, 0, len(perms))
	for _, p := range perms {
		result = append(result, toUserPermission(p))
	}
	return result
}

func toUserPermission(p *tagmanager.UserPermission) UserPermission {
	permID := ""
	if parts := strings.Split(p.Path, "/"); len(parts) >= 4 {
		permID = parts[len(parts)-1]
	}

	up := UserPermission{
		PermissionID: permID,
		EmailAddress: p.EmailAddress,
		AccountID:    p.AccountId,
		Path:         p.Path,
	}

	if p.AccountAccess != nil {
		up.AccountAccess = &AccountAccess{
			Permission: p.AccountAccess.Permission,
		}
	}

	if len(p.ContainerAccess) > 0 {
		ca := make([]ContainerAccess, 0, len(p.ContainerAccess))
		for _, c := range p.ContainerAccess {
			ca = append(ca, ContainerAccess{
				ContainerID: c.ContainerId,
				Permission:  c.Permission,
			})
		}
		up.ContainerAccess = ca
	}

	return up
}

func toAPIAccountAccess(aa *AccountAccess) *tagmanager.AccountAccess {
	if aa == nil {
		return nil
	}
	return &tagmanager.AccountAccess{
		Permission: aa.Permission,
	}
}

func toAPIContainerAccess(ca []ContainerAccess) []*tagmanager.ContainerAccess {
	if len(ca) == 0 {
		return nil
	}
	result := make([]*tagmanager.ContainerAccess, len(ca))
	for i, c := range ca {
		result[i] = &tagmanager.ContainerAccess{
			ContainerId: c.ContainerID,
			Permission:  c.Permission,
		}
	}
	return result
}
