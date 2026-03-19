package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// TemplateToolInput is the unified input for the template tool.
type TemplateToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,get,create,update,delete,import,revert,description:Operation to perform on custom templates"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	// Fields for get/update/delete:
	TemplateID string `json:"templateId,omitempty" jsonschema:"description:Template ID (required for get, update, delete)"`
	// Fields for create/update:
	Name         string `json:"name,omitempty" jsonschema:"description:Template name (required for create)"`
	TemplateData string `json:"templateData,omitempty" jsonschema:"description:Template code in .tpl format (required for create, optional for update)"`
	// Fields for import:
	GalleryOwner string `json:"galleryOwner,omitempty" jsonschema:"description:Gallery template owner (required for import, e.g. 'GoogleAnalytics')"`
	GalleryRepo  string `json:"galleryRepository,omitempty" jsonschema:"description:Gallery template repository (required for import, e.g. 'gtm-cookie-solution')"`
	GallerySha   string `json:"gallerySha,omitempty" jsonschema:"description:Gallery template SHA version (optional for import)"`
	// Fields for delete:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for revert)"`
}


func handleTemplateList(ctx context.Context, input TemplateToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	cacheKey := WorkspaceCacheKey(wc.AccountID, wc.ContainerID, wc.WorkspaceID, "template_list")
	if cached, ok := globalCache.Get(cacheKey); ok {
		return nil, cached, nil
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := wc.WorkspacePath()
	resp, err := retryWithBackoff(tCtx, 3, func() (*tagmanager.ListTemplatesResponse, error) {
		return wc.Client.Service.Accounts.Containers.Workspaces.Templates.List(parent).Context(tCtx).Do()
	})
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	templates := make([]TemplateInfo, 0)
	if resp.Template != nil {
		for _, t := range resp.Template {
			info := TemplateInfo{
				TemplateID:    t.TemplateId,
				Name:          t.Name,
				TagManagerUrl: t.TagManagerUrl,
			}
			if t.GalleryReference != nil && t.GalleryReference.GalleryTemplateId != "" {
				info.Type = fmt.Sprintf("cvt_%s", t.GalleryReference.GalleryTemplateId)
				info.GalleryReference = &GalleryReferenceInfo{
					Owner:             t.GalleryReference.Owner,
					Repository:        t.GalleryReference.Repository,
					Version:           t.GalleryReference.Version,
					GalleryTemplateId: t.GalleryReference.GalleryTemplateId,
				}
			} else {
				info.Type = fmt.Sprintf("cvt_%s_%s", wc.ContainerID, t.TemplateId)
			}
			templates = append(templates, info)
		}
	}

	out := ListTemplatesOutput{Templates: templates}
	globalCache.Set(cacheKey, out)
	return nil, out, nil
}

func handleTemplateGet(ctx context.Context, input TemplateToolInput) (*mcp.CallToolResult, any, error) {
	if input.TemplateID == "" {
		return nil, nil, fmt.Errorf("templateId is required for get action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/templates/%s", wc.WorkspacePath(), input.TemplateID)

	template, err := retryWithBackoff(tCtx, 3, func() (*tagmanager.CustomTemplate, error) {
		return wc.Client.Service.Accounts.Containers.Workspaces.Templates.Get(path).Context(tCtx).Do()
	})
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	output := GetTemplateOutput{
		TemplateID:    template.TemplateId,
		Name:          template.Name,
		TemplateData:  template.TemplateData,
		Path:          template.Path,
		Fingerprint:   template.Fingerprint,
		TagManagerUrl: template.TagManagerUrl,
	}

	if template.GalleryReference != nil && template.GalleryReference.GalleryTemplateId != "" {
		output.Type = fmt.Sprintf("cvt_%s", template.GalleryReference.GalleryTemplateId)
		output.GalleryReference = &GalleryReferenceInfo{
			Owner:             template.GalleryReference.Owner,
			Repository:        template.GalleryReference.Repository,
			Version:           template.GalleryReference.Version,
			GalleryTemplateId: template.GalleryReference.GalleryTemplateId,
		}
	} else {
		output.Type = fmt.Sprintf("cvt_%s_%s", wc.ContainerID, template.TemplateId)
	}

	return nil, output, nil
}

func handleTemplateCreate(ctx context.Context, input TemplateToolInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return nil, nil, fmt.Errorf("name is required for create action")
	}
	if input.TemplateData == "" {
		return nil, nil, fmt.Errorf("templateData is required for create action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := wc.WorkspacePath()
	template := &tagmanager.CustomTemplate{
		Name:         input.Name,
		TemplateData: input.TemplateData,
	}

	created, err := wc.Client.Service.Accounts.Containers.Workspaces.Templates.Create(parent, template).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, CreateTemplateOutput{
		Success:       true,
		TemplateID:    created.TemplateId,
		Name:          created.Name,
		Type:          fmt.Sprintf("cvt_%s_%s", wc.ContainerID, created.TemplateId),
		Path:          created.Path,
		TagManagerUrl: created.TagManagerUrl,
		Message:       fmt.Sprintf("Template '%s' created. Use type 'cvt_%s_%s' when creating tags.", created.Name, wc.ContainerID, created.TemplateId),
	}, nil
}

func handleTemplateUpdate(ctx context.Context, input TemplateToolInput) (*mcp.CallToolResult, any, error) {
	if input.TemplateID == "" {
		return nil, nil, fmt.Errorf("templateId is required for update action")
	}
	if input.Name == "" && input.TemplateData == "" {
		return nil, nil, fmt.Errorf("at least one of name or templateData must be provided")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/templates/%s", wc.WorkspacePath(), input.TemplateID)

	// Get current template for fingerprint and defaults
	current, err := wc.Client.Service.Accounts.Containers.Workspaces.Templates.Get(path).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	template := &tagmanager.CustomTemplate{
		Name:         current.Name,
		TemplateData: current.TemplateData,
		Fingerprint:  current.Fingerprint,
	}

	if input.Name != "" {
		template.Name = input.Name
	}
	if input.TemplateData != "" {
		template.TemplateData = input.TemplateData
	}

	updated, err := wc.Client.Service.Accounts.Containers.Workspaces.Templates.Update(path, template).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	templateType := fmt.Sprintf("cvt_%s_%s", wc.ContainerID, updated.TemplateId)
	if updated.GalleryReference != nil && updated.GalleryReference.GalleryTemplateId != "" {
		templateType = fmt.Sprintf("cvt_%s", updated.GalleryReference.GalleryTemplateId)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, UpdateTemplateOutput{
		Success:       true,
		TemplateID:    updated.TemplateId,
		Name:          updated.Name,
		Type:          templateType,
		Path:          updated.Path,
		Fingerprint:   updated.Fingerprint,
		TagManagerUrl: updated.TagManagerUrl,
		Message:       fmt.Sprintf("Template '%s' updated successfully", updated.Name),
	}, nil
}

func handleTemplateDelete(ctx context.Context, input TemplateToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteTemplateOutput{
			Success: false,
			Message: "Deletion requires confirm: true. Templates in use by tags cannot be deleted.",
		}, nil
	}
	if input.TemplateID == "" {
		return nil, nil, fmt.Errorf("templateId is required for delete action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/templates/%s", wc.WorkspacePath(), input.TemplateID)
	if err := wc.Client.Service.Accounts.Containers.Workspaces.Templates.Delete(path).Context(tCtx).Do(); err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, DeleteTemplateOutput{
		Success: true,
		Message: fmt.Sprintf("Template %s deleted successfully", input.TemplateID),
	}, nil
}

func handleTemplateImport(ctx context.Context, input TemplateToolInput) (*mcp.CallToolResult, any, error) {
	if input.GalleryOwner == "" {
		return nil, nil, fmt.Errorf("galleryOwner is required for import action")
	}
	if input.GalleryRepo == "" {
		return nil, nil, fmt.Errorf("galleryRepository is required for import action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := wc.WorkspacePath()
	call := wc.Client.Service.Accounts.Containers.Workspaces.Templates.ImportFromGallery(parent).
		GalleryOwner(input.GalleryOwner).
		GalleryRepository(input.GalleryRepo).
		AcknowledgePermissions(true)

	if input.GallerySha != "" {
		call = call.GallerySha(input.GallerySha)
	}

	template, err := call.Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	result := TemplateInfo{
		TemplateID:    template.TemplateId,
		Name:          template.Name,
		TagManagerUrl: template.TagManagerUrl,
	}

	if template.GalleryReference != nil && template.GalleryReference.GalleryTemplateId != "" {
		result.Type = fmt.Sprintf("cvt_%s", template.GalleryReference.GalleryTemplateId)
		result.GalleryReference = &GalleryReferenceInfo{
			Owner:             template.GalleryReference.Owner,
			Repository:        template.GalleryReference.Repository,
			Version:           template.GalleryReference.Version,
			GalleryTemplateId: template.GalleryReference.GalleryTemplateId,
		}
	} else {
		result.Type = fmt.Sprintf("cvt_%s_%s", wc.ContainerID, template.TemplateId)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, ImportGalleryTemplateOutput{
		Success:  true,
		Template: result,
		Message:  fmt.Sprintf("Template '%s' imported. Use type '%s' when creating tags.", template.Name, result.Type),
	}, nil
}

func handleTemplateRevert(ctx context.Context, input TemplateToolInput) (*mcp.CallToolResult, any, error) {
	if input.TemplateID == "" {
		return nil, nil, fmt.Errorf("templateId is required for revert action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/templates/%s", wc.WorkspacePath(), input.TemplateID)
	call := wc.Client.Service.Accounts.Containers.Workspaces.Templates.Revert(path).Context(tCtx)
	if input.Fingerprint != "" {
		call = call.Fingerprint(input.Fingerprint)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, RevertOutput{
		Success: true,
		Message: fmt.Sprintf("Template %s reverted to latest published version", input.TemplateID),
		Entity:  resp.Template,
	}, nil
}
