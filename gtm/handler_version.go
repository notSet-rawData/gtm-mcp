package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// VersionToolInput is the unified input for the version tool.
type VersionToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,get,create,publish,compare,find_by_date,set_latest,export,import,description:Operation to perform on versions"`
	// Format for export/import: "ui" (SCREAMING_CASE) or "api" (camelCase) or "auto" (detect)
	Format string `json:"format,omitempty" jsonschema:"enum:ui,api,auto,description:Format. ui = SCREAMING_CASE (GTM UI). api = camelCase (MCP/API). auto = detect. Default: ui for export - auto for import"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	// Fields used by create:
	WorkspaceID string `json:"workspaceId,omitempty" jsonschema:"description:Workspace ID (required for create)"`
	// Fields for get/publish/set_latest/export:
	VersionID string `json:"versionId,omitempty" jsonschema:"description:Version ID (required for get, publish, set_latest, export)"`
	// Fields for create:
	Name string `json:"name,omitempty" jsonschema:"description:Version name (required for create)"`
	// Fields for compare:
	BaseVersionID   string `json:"baseVersionId,omitempty" jsonschema:"description:Base version ID for comparison (required for compare, called versionIdA)"`
	TargetVersionID string `json:"targetVersionId,omitempty" jsonschema:"description:Target version ID for comparison (required for compare, called versionIdB)"`
	// Fields for find_by_date:
	Date string `json:"date,omitempty" jsonschema:"description:Date in YYYY-MM-DD format to find which version was active (required for find_by_date)"`
	// Fields for publish:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true for publish or import (safety guard)"`
	// Fields for import:
	ExportJSON string `json:"exportJson,omitempty" jsonschema:"description:JSON string from a previous export (required for import)"`
	DryRun     bool   `json:"dryRun,omitempty" jsonschema:"description:If true only analyze and return a plan without creating anything (for import)"`
	// Fields for export:
	OutputPath string `json:"outputPath,omitempty" jsonschema:"description:Local filesystem path where the export JSON will be saved. The MCP server runs on the user's machine so this writes directly to their local disk. Example: /home/user/Downloads/export.json. If omitted the file is saved to the user's home directory automatically."`
}


func handleVersionList(ctx context.Context, input VersionToolInput) (*mcp.CallToolResult, any, error) {
	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	headers, err := cc.Client.ListVersionHeaders(tCtx, cc.AccountID, cc.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	return nil, ListVersionsOutput{Versions: headers}, nil
}

func handleVersionGet(ctx context.Context, input VersionToolInput) (*mcp.CallToolResult, any, error) {
	if input.VersionID == "" {
		return nil, nil, fmt.Errorf("versionId is required for get action")
	}

	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	version, err := cc.Client.GetContainerVersion(tCtx, cc.AccountID, cc.ContainerID, input.VersionID)
	if err != nil {
		return nil, nil, err
	}

	return nil, GetVersionOutput{Version: *version}, nil
}

func handleVersionCreate(ctx context.Context, input VersionToolInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return nil, nil, fmt.Errorf("name is required for create action")
	}
	if input.WorkspaceID == "" {
		return nil, nil, fmt.Errorf("workspaceId is required for create action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	versionInput := &VersionInput{
		Name: input.Name,
	}

	version, err := wc.Client.CreateVersion(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, versionInput)
	if err != nil {
		return nil, nil, err
	}

	return nil, CreateVersionOutput{
		Success: true,
		Version: *version,
		Message: fmt.Sprintf("Version '%s' created successfully", input.Name),
	}, nil
}

func handleVersionPublish(ctx context.Context, input VersionToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, PublishVersionOutput{
			Success: false,
			Message: "Publishing requires confirm: true. WARNING: This will make the version LIVE and affect real users/traffic.",
		}, nil
	}
	if input.VersionID == "" {
		return nil, nil, fmt.Errorf("versionId is required for publish action")
	}

	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	version, err := cc.Client.PublishVersion(tCtx, cc.AccountID, cc.ContainerID, input.VersionID)
	if err != nil {
		return nil, nil, err
	}

	return nil, PublishVersionOutput{
		Success: true,
		Version: *version,
		Message: fmt.Sprintf("Version %s published successfully — now LIVE", input.VersionID),
	}, nil
}

func handleVersionCompare(ctx context.Context, input VersionToolInput) (*mcp.CallToolResult, any, error) {
	if input.BaseVersionID == "" || input.TargetVersionID == "" {
		return nil, nil, fmt.Errorf("baseVersionId and targetVersionId are required for compare action")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	vA, err := client.GetContainerVersion(tCtx, input.AccountID, input.ContainerID, input.BaseVersionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get base version %s: %w", input.BaseVersionID, err)
	}

	vB, err := client.GetContainerVersion(tCtx, input.AccountID, input.ContainerID, input.TargetVersionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get target version %s: %w", input.TargetVersionID, err)
	}

	// Compare using existing diffEntities helper and map builders
	tagChanges := diffEntities(tagMap(vA.Tags), tagMap(vB.Tags))
	trigChanges := diffEntities(triggerMap(vA.Triggers), triggerMap(vB.Triggers))
	varChanges := diffEntities(variableMap(vA.Variables), variableMap(vB.Variables))

	added := 0
	modified := 0
	deleted := 0
	for _, c := range append(append(tagChanges, trigChanges...), varChanges...) {
		switch c.Change {
		case "added":
			added++
		case "modified":
			modified++
		case "deleted":
			deleted++
		}
	}

	summary := fmt.Sprintf(
		"Version %s → %s: %d added, %d modified, %d deleted (tags: %d changes, triggers: %d changes, variables: %d changes)",
		input.BaseVersionID, input.TargetVersionID, added, modified, deleted,
		len(tagChanges), len(trigChanges), len(varChanges),
	)

	return nil, CompareVersionsOutput{
		VersionA:    input.BaseVersionID,
		VersionB:    input.TargetVersionID,
		TagChanges:  tagChanges,
		TrigChanges: trigChanges,
		VarChanges:  varChanges,
		Summary:     summary,
	}, nil
}

func handleVersionFindByDate(ctx context.Context, input VersionToolInput) (*mcp.CallToolResult, any, error) {
	if input.Date == "" {
		return nil, nil, fmt.Errorf("date is required for find_by_date action (format: YYYY-MM-DD)")
	}

	targetDate, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid date format %q, expected YYYY-MM-DD: %w", input.Date, err)
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	headers, err := client.ListVersionHeaders(tCtx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list versions: %w", err)
	}

	if len(headers) == 0 {
		return nil, FindVersionByDateOutput{
			TargetDate: input.Date,
			Message:    "No versions found in this container",
		}, nil
	}

	// Binary search through version headers
	low, high := 0, len(headers)-1
	bestIdx := -1
	apiCalls := 0

	for low <= high {
		mid := (low + high) / 2
		apiCalls++

		version, err := client.GetContainerVersion(tCtx, input.AccountID, input.ContainerID, headers[mid].VersionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get version %s: %w", headers[mid].VersionID, err)
		}

		versionTime, err := fingerprintToTime(version.Fingerprint)
		if err != nil {
			return nil, nil, fmt.Errorf("version %s fingerprint error: %w", version.VersionID, err)
		}

		if versionTime.Before(targetDate.Add(24 * time.Hour)) {
			bestIdx = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	if bestIdx == -1 {
		return nil, FindVersionByDateOutput{
			TargetDate: input.Date,
			Message:    fmt.Sprintf("No version was active on %s (all versions are newer)", input.Date),
			APICalls:   apiCalls,
		}, nil
	}

	apiCalls++
	bestVersion, err := client.GetContainerVersion(tCtx, input.AccountID, input.ContainerID, headers[bestIdx].VersionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get best version: %w", err)
	}

	versionTime, _ := fingerprintToTime(bestVersion.Fingerprint)

	return nil, FindVersionByDateOutput{
		TargetDate: input.Date,
		Version: VersionDateInfo{
			VersionID:   bestVersion.VersionID,
			Name:        bestVersion.Name,
			Fingerprint: bestVersion.Fingerprint,
			Timestamp:   versionTime.UTC().Format(time.RFC3339),
			Path:        bestVersion.Path,
		},
		Message:  fmt.Sprintf("Version %s ('%s') was active on %s", bestVersion.VersionID, bestVersion.Name, input.Date),
		APICalls: apiCalls,
	}, nil
}

func handleVersionSetLatest(ctx context.Context, input VersionToolInput) (*mcp.CallToolResult, any, error) {
	if input.VersionID == "" {
		return nil, nil, fmt.Errorf("versionId is required for set_latest action")
	}

	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	version, err := cc.Client.SetLatestVersion(tCtx, cc.AccountID, cc.ContainerID, input.VersionID)
	if err != nil {
		return nil, nil, err
	}

	return nil, SetLatestVersionOutput{
		Success: true,
		Version: *version,
		Message: fmt.Sprintf("Version %s set as latest", input.VersionID),
	}, nil
}

func handleVersionExport(ctx context.Context, input VersionToolInput) (*mcp.CallToolResult, any, error) {
	if input.VersionID == "" {
		return nil, nil, fmt.Errorf("versionId is required for export action")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Default to "ui" format
	format := input.Format
	if format == "" {
		format = "ui"
	}

	tCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var rawJSON json.RawMessage

	switch format {
	case "api":
		rawJSON, err = client.GetContainerVersionRawAPI(tCtx, input.AccountID, input.ContainerID, input.VersionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to export version %s: %w", input.VersionID, err)
		}
	default: // "ui"
		rawJSON, err = client.GetContainerVersionRaw(tCtx, input.AccountID, input.ContainerID, input.VersionID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to export version %s: %w", input.VersionID, err)
		}
	}

	// ALWAYS write clean JSON to disk — avoids MCP response wrapping issues
	outputPath := input.OutputPath
	if outputPath == "" {
		// Auto-generate path in user's home directory for easy access
		home, _ := os.UserHomeDir()
		if home == "" {
			home = os.TempDir()
		}
		outputPath = fmt.Sprintf("%s/GTM-export-%s_v%s.json", home, input.ContainerID, input.VersionID)
	}

	// Pretty-print the JSON for readability
	var prettyJSON []byte
	var raw interface{}
	if err := json.Unmarshal(rawJSON, &raw); err == nil {
		prettyJSON, _ = json.MarshalIndent(raw, "", "    ")
	} else {
		prettyJSON = rawJSON
	}

	if err := os.WriteFile(outputPath, prettyJSON, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to write export to %s: %w", outputPath, err)
	}

	msg := fmt.Sprintf("EXPORT COMPLETE. The file has been saved to the user's LOCAL machine at:\n%s\n\nFile size: %d bytes | Format: %s\nThis file is ready for GTM UI import (Admin → Import Container).\nIMPORTANT: This file is on the user's local filesystem, NOT in the cloud. The user can open it directly from that path.", outputPath, len(prettyJSON), format)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}, nil, nil
}
