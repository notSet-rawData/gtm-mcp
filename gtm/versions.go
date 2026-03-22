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

// GTMExportContainer is the top-level structure for GTM container export/import JSON.
type GTMExportContainer struct {
	ExportFormatVersion int         `json:"exportFormatVersion"`
	ExportTime          string      `json:"exportTime"`
	ContainerVersion    interface{} `json:"containerVersion"`
}

// enumOverrides maps known edge cases where generic camelCase→SCREAMING_CASE
// conversion would produce incorrect results. The generic algorithm handles
// the vast majority of cases automatically.
var enumOverrides = map[string]string{
	// Single-word values that need to be uppercased but have no camelCase boundary
	"template": "TEMPLATE", "integer": "INTEGER", "boolean": "BOOLEAN",
	"list": "LIST", "map": "MAP", "init": "INIT", "always": "ALWAYS",
	"pageview": "PAGEVIEW", "click": "CLICK", "timer": "TIMER",
	"equals": "EQUALS", "contains": "CONTAINS", "greater": "GREATER",
	"less": "LESS", "unlimited": "UNLIMITED", "needed": "NEEDED",
	"referrer": "REFERRER", "event": "EVENT", "language": "LANGUAGE",
	"platform": "PLATFORM", "resolution": "RESOLUTION", "regex": "REGEX",
	// Multi-word where the generic algorithm might not produce the right output
	"web": "WEB", "android": "ANDROID", "ios": "IOS",
	"amp": "AMP", "server": "SERVER",
}

// Fields whose values should be converted from camelCase to SCREAMING_CASE.
var enumFields = map[string]bool{
	"type": true, "tagFiringOption": true, "consentStatus": true,
}

// Fields whose array values should be converted from camelCase to SCREAMING_CASE.
var enumArrayFields = map[string]bool{
	"usageContext": true, "containerContexts": true,
}

// camelToScreamingCase converts a camelCase string to SCREAMING_SNAKE_CASE.
// Examples: "requestPath" → "REQUEST_PATH", "customEvent" → "CUSTOM_EVENT",
// "domReady" → "DOM_READY", "youTubeVideo" → "YOU_TUBE_VIDEO".
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

// convertEnumValue converts a single enum value to SCREAMING_CASE.
// Priority: 1) manual overrides, 2) skip non-enum values, 3) generic algorithm.
func convertEnumValue(s string) string {
	// 1. Check manual overrides first (handles single-word and edge cases)
	if mapped, exists := enumOverrides[s]; exists {
		return mapped
	}

	// 2. Already SCREAMING_CASE? (e.g. "CONTAINER_VERSION", "TEMPLATE") — leave as-is
	if s == strings.ToUpper(s) {
		return s
	}

	// 3. Contains underscores → template/type IDs (e.g. "cvt_198845464_347") — leave as-is
	if strings.Contains(s, "_") {
		return s
	}

	// 4. Contains digits → type IDs (e.g. "gaawc" is fine, but "ga4" wouldn't convert well) — leave as-is
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return s
		}
	}

	// 5. Detect camelCase: must have at least one lowercase→uppercase transition
	hasCamelCase := false
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			hasCamelCase = true
			break
		}
	}
	if !hasCamelCase {
		// Pure lowercase, unknown — not in overrides, not camelCase
		// Leave as-is (e.g. "gaawc", custom type IDs)
		return s
	}

	// 6. It's camelCase — auto-convert via generic algorithm
	return camelToScreamingCase(s)
}

// convertEnumsToScreamingCase recursively walks a JSON structure and converts
// known enum fields from camelCase (API format) to SCREAMING_CASE (import format).
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

// =============================================================================
// Inverse conversion: SCREAMING_CASE → camelCase (for import from GTM UI format)
// =============================================================================

// reverseEnumOverrides is the inverse of enumOverrides, auto-generated at init.
var reverseEnumOverrides map[string]string

func init() {
	reverseEnumOverrides = make(map[string]string, len(enumOverrides))
	for camel, screaming := range enumOverrides {
		reverseEnumOverrides[screaming] = camel
	}
}

// screamingToCamelCase converts a SCREAMING_SNAKE_CASE string to camelCase.
// Examples: "REQUEST_PATH" → "requestPath", "CUSTOM_EVENT" → "customEvent",
// "YOU_TUBE_VIDEO" → "youTubeVideo".
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
			// Capitalize first letter of subsequent parts
			result.WriteByte(byte(lower[0] - 'a' + 'A'))
			result.WriteString(lower[1:])
		}
	}
	return result.String()
}

// reverseEnumValue converts a single SCREAMING_CASE enum value to camelCase.
// Priority: 1) manual reverse overrides, 2) skip non-SCREAMING values, 3) generic algorithm.
func reverseEnumValue(s string) string {
	// 1. Check reverse overrides first
	if mapped, exists := reverseEnumOverrides[s]; exists {
		return mapped
	}

	// 2. Already camelCase or lowercase? Leave as-is
	if s != strings.ToUpper(s) {
		return s
	}

	// 3. Contains digits? Probably a type ID — leave as-is
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return s
		}
	}

	// 4. No underscores? Single word like "MAP" — already handled by overrides.
	// If not in overrides, lowercase it as a safe fallback
	if !strings.Contains(s, "_") {
		return strings.ToLower(s)
	}

	// 5. It's SCREAMING_CASE — auto-convert
	return screamingToCamelCase(s)
}

// convertEnumsToCamelCase recursively walks a JSON structure and converts
// known enum fields from SCREAMING_CASE (UI export format) to camelCase (API format).
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
// in the official GTM export format, compatible with GTM UI import.
// Enum values are converted from camelCase to SCREAMING_CASE.
func (c *Client) GetContainerVersionRaw(ctx context.Context, accountID, containerID, versionID string) (json.RawMessage, error) {
	return c.getContainerVersionRawInternal(ctx, accountID, containerID, versionID, true)
}

// GetContainerVersionRawAPI retrieves a container version as raw JSON
// in the native API format (camelCase), suitable for programmatic import via MCP.
// No enum conversion is applied.
func (c *Client) GetContainerVersionRawAPI(ctx context.Context, accountID, containerID, versionID string) (json.RawMessage, error) {
	return c.getContainerVersionRawInternal(ctx, accountID, containerID, versionID, false)
}

// getContainerVersionRawInternal is the shared implementation for both export formats.
// When normalizeEnums is true, camelCase enum values are converted to SCREAMING_CASE.
func (c *Client) getContainerVersionRawInternal(ctx context.Context, accountID, containerID, versionID string, normalizeEnums bool) (json.RawMessage, error) {
	// Try the internal GTM partialexport API first (returns native export format)
	if c.HTTPClient != nil {
		data, err := c.exportViaInternalAPI(ctx, accountID, containerID, versionID, normalizeEnums)
		if err == nil {
			return data, nil
		}
		slog.Warn("partialexport API failed, falling back to public API", "error", err)
	}

	// Fallback: public API
	return c.exportViaPublicAPI(ctx, accountID, containerID, versionID, normalizeEnums)
}

// exportViaInternalAPI calls the GTM UI's internal partialexport endpoint.
// The response format is: )]}'{"default":{"exportedContainerJson":"<escaped JSON string>"}}
// Google internal APIs prefix responses with )]}' as anti-XSS protection.
// We strip the prefix, parse the wrapper, and extract the inner JSON string.
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

	// Strip Google's anti-XSS prefix )]}'  (with optional newline)
	jsonBody := body
	if len(jsonBody) > 4 && string(jsonBody[:4]) == ")]}" {
		// Skip )]}' and any following whitespace/newline
		jsonBody = jsonBody[4:]
		for len(jsonBody) > 0 && (jsonBody[0] == '\'' || jsonBody[0] == '\n' || jsonBody[0] == '\r' || jsonBody[0] == ' ') {
			jsonBody = jsonBody[1:]
		}
		slog.Debug("stripped anti-XSS prefix from partialexport response")
	}

	// Parse the wrapper: {"default":{"exportedContainerJson":"..."}}
	var wrapper struct {
		Default struct {
			ExportedContainerJSON string `json:"exportedContainerJson"`
		} `json:"default"`
	}
	if err := json.Unmarshal(jsonBody, &wrapper); err != nil {
		// Log first bytes for debugging
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

	// Validate the extracted JSON is valid
	if !json.Valid([]byte(exportJSON)) {
		return nil, fmt.Errorf("partialexport exportedContainerJson is not valid JSON")
	}

	// Optionally normalize enums from camelCase to SCREAMING_CASE
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

// exportViaPublicAPI uses the official GTM API v2, optionally converting enums.
func (c *Client) exportViaPublicAPI(ctx context.Context, accountID, containerID, versionID string, normalizeEnums bool) (json.RawMessage, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/versions/%s",
		accountID, containerID, versionID)

	result, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ContainerVersion, error) {
		return c.Service.Accounts.Containers.Versions.Get(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	// Marshal the Google API struct to JSON
	intermediate, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal container version: %w", err)
	}

	// Unmarshal into generic map for wrapping (and optional enum conversion)
	var versionMap map[string]interface{}
	if err := json.Unmarshal(intermediate, &versionMap); err != nil {
		return nil, fmt.Errorf("failed to parse container version JSON: %w", err)
	}

	if normalizeEnums {
		convertEnumsToScreamingCase(versionMap)
	}

	// Wrap in export container format
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

// CreateVersion creates a new container version from a workspace.
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

// PublishVersion publishes a container version to make it live.
// Retries up to 3 times with a 2s delay on 404 errors (eventual consistency after create).
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

		// Retry on 404 — version may not have propagated yet after create
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

// GetWorkspaceStatus checks if a workspace has changes to publish.
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

// PublishedVersion represents the result of publishing a version.
type PublishedVersion struct {
	VersionID string `json:"containerVersionId"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}

// WorkspaceStatus represents the status of a workspace.
type WorkspaceStatus struct {
	HasChanges    bool `json:"hasChanges"`
	HasConflicts  bool `json:"hasConflicts"`
	ChangeCount   int  `json:"changeCount"`
	ConflictCount int  `json:"conflictCount"`
}

// ContainerVersionDetail represents a full container version with all entities.
type ContainerVersionDetail struct {
	VersionID   string `json:"containerVersionId"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint,omitempty"`
	// Full entity snapshots at this version
	Tags             []Tag               `json:"tag,omitempty"`
	Triggers         []Trigger           `json:"trigger,omitempty"`
	Variables        []Variable          `json:"variable,omitempty"`
	Folders          []Folder            `json:"folder,omitempty"`
	CustomTemplates  []TemplateInfo      `json:"customTemplate,omitempty"`
	Clients          []ClientInfo        `json:"client,omitempty"`
	Transformations  []TransformationInfo `json:"transformation,omitempty"`
	BuiltInVariables []BuiltInVariable   `json:"builtInVariable,omitempty"`
}

// GetContainerVersion retrieves a full container version with all entities.
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

	// Convert tags
	if result.Tag != nil {
		detail.Tags = toTags(result.Tag)
	}

	// Convert triggers
	if result.Trigger != nil {
		detail.Triggers = toTriggers(result.Trigger)
	}

	// Convert variables
	if result.Variable != nil {
		detail.Variables = toVariables(result.Variable)
	}

	// Convert folders
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

	// Convert custom templates
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

	// Convert clients (server-side containers)
	if result.Client != nil {
		detail.Clients = toClients(result.Client)
	}

	// Convert transformations (server-side containers)
	if result.Transformation != nil {
		detail.Transformations = toTransformations(result.Transformation)
	}

	// Convert built-in variables
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

// ListVersionHeaders returns version headers (lightweight metadata) for a container.
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

// SetLatestVersion sets a specific version as the latest (live) version — used for rollback.
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
