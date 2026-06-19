package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

type GTMExportContainer struct {
	ExportFormatVersion int         `json:"exportFormatVersion"`
	ExportTime          string      `json:"exportTime"`
	ContainerVersion    interface{} `json:"containerVersion"`
}

var enumOverrides = map[string]string{
	"template": "TEMPLATE", "integer": "INTEGER", "boolean": "BOOLEAN",
	"list": "LIST", "map": "MAP", "init": "INIT", "always": "ALWAYS",
	"pageview": "PAGEVIEW", "click": "CLICK", "timer": "TIMER",
	"equals": "EQUALS", "contains": "CONTAINS", "greater": "GREATER",
	"less": "LESS", "unlimited": "UNLIMITED", "needed": "NEEDED",
	"referrer": "REFERRER", "event": "EVENT", "language": "LANGUAGE",
	"platform": "PLATFORM", "resolution": "RESOLUTION", "regex": "REGEX",
	"web": "WEB", "android": "ANDROID", "ios": "IOS",
	"amp": "AMP", "server": "SERVER",
}

var enumFields = map[string]bool{
	"type": true, "tagFiringOption": true, "consentStatus": true,
}

var enumArrayFields = map[string]bool{
	"usageContext": true, "containerContexts": true,
}

func camelToScreamingCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		if r >= 'a' && r <= 'z' {
			result.WriteByte(byte(r - 'a' + 'A'))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func convertEnumValue(s string) string {
	if mapped, exists := enumOverrides[s]; exists {
		return mapped
	}

	if s == strings.ToUpper(s) {
		return s
	}

	if strings.Contains(s, "_") {
		return s
	}

	for _, r := range s {
		if r >= '0' && r <= '9' {
			return s
		}
	}

	hasCamelCase := false
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			hasCamelCase = true
			break
		}
	}
	if !hasCamelCase {
		return s
	}

	return camelToScreamingCase(s)
}

func convertEnumsToScreamingCase(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		for key, child := range val {
			if enumFields[key] {
				if s, ok := child.(string); ok {
					val[key] = convertEnumValue(s)
				}
			} else if enumArrayFields[key] {
				if arr, ok := child.([]interface{}); ok {
					for i, item := range arr {
						if s, ok := item.(string); ok {
							arr[i] = convertEnumValue(s)
						}
					}
				}
			} else {
				val[key] = convertEnumsToScreamingCase(child)
			}
		}
		return val
	case []interface{}:
		for i, item := range val {
			val[i] = convertEnumsToScreamingCase(item)
		}
		return val
	default:
		return v
	}
}

var reverseEnumOverrides map[string]string

func init() {
	reverseEnumOverrides = make(map[string]string, len(enumOverrides))
	for camel, screaming := range enumOverrides {
		reverseEnumOverrides[screaming] = camel
	}
}

func screamingToCamelCase(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 0 {
		return s
	}
	var result strings.Builder
	for i, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		if i == 0 {
			result.WriteString(lower)
		} else {
			result.WriteByte(byte(lower[0] - 'a' + 'A'))
			result.WriteString(lower[1:])
		}
	}
	return result.String()
}

func reverseEnumValue(s string) string {
	if mapped, exists := reverseEnumOverrides[s]; exists {
		return mapped
	}

	if s != strings.ToUpper(s) {
		return s
	}

	for _, r := range s {
		if r >= '0' && r <= '9' {
			return s
		}
	}

	if !strings.Contains(s, "_") {
		return strings.ToLower(s)
	}

	return screamingToCamelCase(s)
}

func convertEnumsToCamelCase(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		for key, child := range val {
			if enumFields[key] {
				if s, ok := child.(string); ok {
					val[key] = reverseEnumValue(s)
				}
			} else if enumArrayFields[key] {
				if arr, ok := child.([]interface{}); ok {
					for i, item := range arr {
						if s, ok := item.(string); ok {
							arr[i] = reverseEnumValue(s)
						}
					}
				}
			} else {
				val[key] = convertEnumsToCamelCase(child)
			}
		}
		return val
	case []interface{}:
		for i, item := range val {
			val[i] = convertEnumsToCamelCase(item)
		}
		return val
	default:
		return v
	}
}
func (c *Client) GetContainerVersionRaw(ctx context.Context, accountID, containerID, versionID string) (json.RawMessage, error) {
	return c.getContainerVersionRawInternal(ctx, accountID, containerID, versionID, true)
}

func (c *Client) GetContainerVersionRawAPI(ctx context.Context, accountID, containerID, versionID string) (json.RawMessage, error) {
	return c.getContainerVersionRawInternal(ctx, accountID, containerID, versionID, false)
}

func (c *Client) getContainerVersionRawInternal(ctx context.Context, accountID, containerID, versionID string, normalizeEnums bool) (json.RawMessage, error) {
	if c.HTTPClient != nil {
		data, err := c.exportViaInternalAPI(ctx, accountID, containerID, versionID, normalizeEnums)
		if err == nil {
			return data, nil
		}
		slog.Warn("partialexport API failed, falling back to public API", "error", err)
	}

	return c.exportViaPublicAPI(ctx, accountID, containerID, versionID, normalizeEnums)
}

func (c *Client) exportViaInternalAPI(ctx context.Context, accountID, containerID, versionID string, normalizeEnums bool) (json.RawMessage, error) {
	url := fmt.Sprintf(
		"https://tagmanager.google.com/api/accounts/%s/containers/%s/versions/%s/partialexport?hl=en",
		accountID, containerID, versionID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("partialexport returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	slog.Debug("partialexport response", "size", len(body), "content_type", resp.Header.Get("Content-Type"))

	jsonBody := body
	if len(jsonBody) > 4 && string(jsonBody[:4]) == ")]}" {
		jsonBody = jsonBody[4:]
		for len(jsonBody) > 0 && (jsonBody[0] == '\'' || jsonBody[0] == '\n' || jsonBody[0] == '\r' || jsonBody[0] == ' ') {
			jsonBody = jsonBody[1:]
		}
		slog.Debug("stripped anti-XSS prefix from partialexport response")
	}

	var wrapper struct {
		Default struct {
			ExportedContainerJSON string `json:"exportedContainerJson"`
		} `json:"default"`
	}
	if err := json.Unmarshal(jsonBody, &wrapper); err != nil {
		preview := string(jsonBody)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		slog.Warn("partialexport wrapper parse failed", "error", err, "body_preview", preview)
		return nil, fmt.Errorf("failed to parse partialexport wrapper: %w", err)
	}

	exportJSON := wrapper.Default.ExportedContainerJSON
	if exportJSON == "" {
		return nil, fmt.Errorf("partialexport returned empty exportedContainerJson")
	}

	if !json.Valid([]byte(exportJSON)) {
		return nil, fmt.Errorf("partialexport exportedContainerJson is not valid JSON")
	}

	if normalizeEnums {
		var exportMap map[string]interface{}
		if err := json.Unmarshal([]byte(exportJSON), &exportMap); err != nil {
			slog.Warn("skipping enum conversion for partialexport", "error", err)
			return json.RawMessage(exportJSON), nil
		}
		convertEnumsToScreamingCase(exportMap)

		normalizedJSON, err := json.MarshalIndent(exportMap, "", "    ")
		if err != nil {
			return json.RawMessage(exportJSON), nil
		}

		slog.Info("partialexport succeeded", "json_size", len(normalizedJSON), "normalized", true)
		return json.RawMessage(normalizedJSON), nil
	}

	slog.Info("partialexport succeeded", "json_size", len(exportJSON), "normalized", false)
	return json.RawMessage(exportJSON), nil
}

var knownContainerFeatureFlags = []string{
	"supportBuiltInVariables",
	"supportClients",
	"supportEnvironments",
	"supportFolders",
	"supportGtagConfigs",
	"supportTags",
	"supportTemplates",
	"supportTransformations",
	"supportTriggers",
	"supportUserPermissions",
	"supportVariables",
	"supportVersions",
	"supportWorkspaces",
	"supportZones",
}

func ensureContainerFeatureFlags(versionMap map[string]interface{}) {
	containerRaw, ok := versionMap["container"]
	if !ok {
		return
	}
	container, ok := containerRaw.(map[string]interface{})
	if !ok {
		return
	}
	featuresRaw, ok := container["features"]
	if !ok {
		features := make(map[string]interface{}, len(knownContainerFeatureFlags))
		for _, flag := range knownContainerFeatureFlags {
			features[flag] = false
		}
		container["features"] = features
		return
	}
	features, ok := featuresRaw.(map[string]interface{})
	if !ok {
		return
	}
	for _, flag := range knownContainerFeatureFlags {
		if _, exists := features[flag]; !exists {
			features[flag] = false
		}
	}
}

func (c *Client) exportViaPublicAPI(ctx context.Context, accountID, containerID, versionID string, normalizeEnums bool) (json.RawMessage, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/versions/%s",
		accountID, containerID, versionID)

	result, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ContainerVersion, error) {
		return c.Service.Accounts.Containers.Versions.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	intermediate, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal container version: %w", err)
	}

	var versionMap map[string]interface{}
	if err := json.Unmarshal(intermediate, &versionMap); err != nil {
		return nil, fmt.Errorf("failed to parse container version JSON: %w", err)
	}

	ensureContainerFeatureFlags(versionMap)

	if normalizeEnums {
		convertEnumsToScreamingCase(versionMap)
	}

	export := GTMExportContainer{
		ExportFormatVersion: 2,
		ExportTime:          time.Now().UTC().Format("2006-01-02 15:04:05"),
		ContainerVersion:    versionMap,
	}

	data, err := json.MarshalIndent(export, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal export container: %w", err)
	}

	return data, nil
}

func (c *Client) CreateVersion(ctx context.Context, accountID, containerID, workspaceID string, input *VersionInput) (*CreatedVersion, error) {
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)

	req := &tagmanager.CreateContainerVersionRequestVersionOptions{
		Name:  input.Name,
		Notes: input.Notes,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.CreateVersion(parent, req).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	if result == nil || result.ContainerVersion == nil {
		if result != nil && result.SyncStatus != nil && !result.SyncStatus.SyncError {
			return nil, fmt.Errorf("no version created - workspace sync required, merge pending changes first")
		}
		return nil, fmt.Errorf("no version created - workspace may have no changes")
	}

	cv := &CreatedVersion{
		VersionID:     result.ContainerVersion.ContainerVersionId,
		Name:          result.ContainerVersion.Name,
		Path:          result.ContainerVersion.Path,
		CompilerError: result.CompilerError,
	}

	return cv, nil
}

func (c *Client) PublishVersion(ctx context.Context, accountID, containerID, versionID string) (*PublishedVersion, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/versions/%s",
		accountID, containerID, versionID)

	var result *tagmanager.PublishContainerVersionResponse
	var err error

	for attempt := 0; attempt < 3; attempt++ {
		result, err = c.Service.Accounts.Containers.Versions.Publish(path).Context(ctx).Do()
		if err == nil {
			break
		}

		errMsg := err.Error()
		if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "Not Found") {
			if attempt < 2 {
				select {
				case <-time.After(2 * time.Second):
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				continue
			}
		}

		return nil, mapGoogleError(err)
	}

	if result == nil || result.ContainerVersion == nil {
		return nil, fmt.Errorf("publish returned empty response")
	}

	return &PublishedVersion{
		VersionID: result.ContainerVersion.ContainerVersionId,
		Name:      result.ContainerVersion.Name,
		Path:      result.ContainerVersion.Path,
	}, nil
}

func (c *Client) GetWorkspaceStatus(ctx context.Context, accountID, containerID, workspaceID string) (*WorkspaceStatus, error) {
	path := BuildWorkspacePath(accountID, containerID, workspaceID)

	status, err := retryWithBackoff(ctx, 3, func() (*tagmanager.GetWorkspaceStatusResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.GetStatus(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &WorkspaceStatus{
		HasChanges:    len(status.WorkspaceChange) > 0,
		HasConflicts:  len(status.MergeConflict) > 0,
		ChangeCount:   len(status.WorkspaceChange),
		ConflictCount: len(status.MergeConflict),
	}, nil
}

type PublishedVersion struct {
	VersionID string `json:"containerVersionId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}

type WorkspaceStatus struct {
	HasChanges    bool `json:"hasChanges"`
	HasConflicts  bool `json:"hasConflicts"`
	ChangeCount   int  `json:"changeCount"`
	ConflictCount int  `json:"conflictCount"`
}

type ContainerVersionDetail struct {
	VersionID        string               `json:"containerVersionId"`
	Name             string               `json:"name,omitempty"`
	Description      string               `json:"description,omitempty"`
	Path             string               `json:"path"`
	Fingerprint      string               `json:"fingerprint,omitempty"`
	Tags             []Tag                `json:"tag,omitempty"`
	Triggers         []Trigger            `json:"trigger,omitempty"`
	Variables        []Variable           `json:"variable,omitempty"`
	Folders          []Folder             `json:"folder,omitempty"`
	CustomTemplates  []TemplateInfo       `json:"customTemplate,omitempty"`
	Clients          []ClientInfo         `json:"client,omitempty"`
	Transformations  []TransformationInfo `json:"transformation,omitempty"`
	BuiltInVariables []BuiltInVariable    `json:"builtInVariable,omitempty"`
}

func (c *Client) GetContainerVersion(ctx context.Context, accountID, containerID, versionID string) (*ContainerVersionDetail, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/versions/%s",
		accountID, containerID, versionID)

	result, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ContainerVersion, error) {
		return c.Service.Accounts.Containers.Versions.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	detail := &ContainerVersionDetail{
		VersionID:   result.ContainerVersionId,
		Name:        result.Name,
		Description: result.Description,
		Path:        result.Path,
		Fingerprint: result.Fingerprint,
	}

	if result.Tag != nil {
		detail.Tags = toTags(result.Tag)
	}

	if result.Trigger != nil {
		detail.Triggers = toTriggers(result.Trigger)
	}

	if result.Variable != nil {
		detail.Variables = toVariables(result.Variable)
	}

	if result.Folder != nil {
		detail.Folders = make([]Folder, 0, len(result.Folder))
		for _, f := range result.Folder {
			detail.Folders = append(detail.Folders, Folder{
				FolderID: f.FolderId,
				Name:     f.Name,
				Path:     f.Path,
			})
		}
	}

	if result.CustomTemplate != nil {
		detail.CustomTemplates = make([]TemplateInfo, 0, len(result.CustomTemplate))
		for _, t := range result.CustomTemplate {
			info := TemplateInfo{
				TemplateID: t.TemplateId,
				Name:       t.Name,
			}
			if t.GalleryReference != nil {
				info.GalleryReference = &GalleryReferenceInfo{
					Owner:      t.GalleryReference.Owner,
					Repository: t.GalleryReference.Repository,
					Version:    t.GalleryReference.Version,
				}
			}
			detail.CustomTemplates = append(detail.CustomTemplates, info)
		}
	}

	if result.Client != nil {
		detail.Clients = toClients(result.Client)
	}

	if result.Transformation != nil {
		detail.Transformations = toTransformations(result.Transformation)
	}

	if result.BuiltInVariable != nil {
		detail.BuiltInVariables = make([]BuiltInVariable, 0, len(result.BuiltInVariable))
		for _, bv := range result.BuiltInVariable {
			detail.BuiltInVariables = append(detail.BuiltInVariables, BuiltInVariable{
				Name: bv.Name,
				Type: bv.Type,
				Path: bv.Path,
			})
		}
	}

	return detail, nil
}

func (c *Client) ListVersionHeaders(ctx context.Context, accountID, containerID string) ([]VersionInfo, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s", accountID, containerID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListContainerVersionsResponse, error) {
		return c.Service.Accounts.Containers.VersionHeaders.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	versions := make([]VersionInfo, 0)
	if resp.ContainerVersionHeader != nil {
		for _, v := range resp.ContainerVersionHeader {
			versions = append(versions, VersionInfo{
				VersionID:          v.ContainerVersionId,
				Name:               v.Name,
				Deleted:            v.Deleted,
				NumTags:            v.NumTags,
				NumTriggers:        v.NumTriggers,
				NumVars:            v.NumVariables,
				NumCustomTemplates: v.NumCustomTemplates,
				Path:               v.Path,
			})
		}
	}

	return versions, nil
}

func (c *Client) SetLatestVersion(ctx context.Context, accountID, containerID, versionID string) (*PublishedVersion, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/versions/%s",
		accountID, containerID, versionID)

	result, err := c.Service.Accounts.Containers.Versions.SetLatest(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &PublishedVersion{
		VersionID: result.ContainerVersionId,
		Name:      result.Name,
		Path:      result.Path,
	}, nil
}
