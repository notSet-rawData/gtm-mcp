package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

type ImportToolInput struct {
	AccountID   string `json:"accountId" jsonschema:"description:Target GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:Target GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:Target workspace ID (entities will be created here)"`
	ExportJSON  string `json:"exportJson" jsonschema:"description:The JSON string from a previous export (either ui or api format)"`
	Format      string `json:"format,omitempty" jsonschema:"enum:ui,api,auto,description:Format of the input JSON. ui=SCREAMING_CASE (from GTM UI export). api=camelCase (from MCP export). auto=detect automatically. Default: auto"`
	DryRun      bool   `json:"dryRun,omitempty" jsonschema:"description:If true only analyze and return a plan without creating anything"`
	Confirm     bool   `json:"confirm,omitempty" jsonschema:"description:Must be true to actually import (safety guard)"`
}

type ImportContainerOutput struct {
	Success bool          `json:"success"`
	DryRun  bool          `json:"dryRun"`
	Message string        `json:"message"`
	Plan    *ImportPlan   `json:"plan,omitempty"`
	Result  *ImportResult `json:"result,omitempty"`
}

type ImportPlan struct {
	Folders         int `json:"folders"`
	Templates       int `json:"templates"`
	Variables       int `json:"variables"`
	Triggers        int `json:"triggers"`
	Tags            int `json:"tags"`
	Clients         int `json:"clients"`
	Transformations int `json:"transformations"`
	Total           int `json:"total"`
}

type ImportResult struct {
	FoldersCreated         int               `json:"foldersCreated"`
	TemplatesCreated       int               `json:"templatesCreated"`
	VariablesCreated       int               `json:"variablesCreated"`
	TriggersCreated        int               `json:"triggersCreated"`
	TagsCreated            int               `json:"tagsCreated"`
	ClientsCreated         int               `json:"clientsCreated"`
	TransformationsCreated int               `json:"transformationsCreated"`
	Errors                 []string          `json:"errors,omitempty"`
	IDMap                  map[string]string `json:"idMap,omitempty"`
}

func handleVersionImport(ctx context.Context, input VersionToolInput) (*mcp.CallToolResult, any, error) {
	if input.WorkspaceID == "" {
		return nil, nil, fmt.Errorf("workspaceId is required for import action")
	}

	importInput := ImportToolInput{
		AccountID:   input.AccountID,
		ContainerID: input.ContainerID,
		WorkspaceID: input.WorkspaceID,
		ExportJSON:  input.ExportJSON,
		Format:      input.Format,
		DryRun:      input.DryRun,
		Confirm:     input.Confirm,
	}

	return executeImport(ctx, importInput)
}

func executeImport(ctx context.Context, input ImportToolInput) (*mcp.CallToolResult, any, error) {
	if input.ExportJSON == "" {
		return nil, nil, fmt.Errorf("exportJson is required for import action")
	}

	var exportData map[string]interface{}
	if err := json.Unmarshal([]byte(input.ExportJSON), &exportData); err != nil {
		return nil, nil, fmt.Errorf("failed to parse exportJson: %w", err)
	}

	versionData, ok := getContainerVersionData(exportData)
	if !ok {
		return nil, nil, fmt.Errorf("exportJson must contain a 'containerVersion' object (standard GTM export format)")
	}

	format := input.Format
	if format == "" || format == "auto" {
		format = detectFormat(versionData)
	}

	if format == "ui" {
		convertEnumsToCamelCase(versionData)
		slog.Info("import: converted enums from SCREAMING_CASE to camelCase")
	}

	folders := extractMapArray(versionData, "folder")
	customTemplates := extractMapArray(versionData, "customTemplate")
	variables := extractMapArray(versionData, "variable")
	triggers := extractMapArray(versionData, "trigger")
	tags := extractMapArray(versionData, "tag")
	clients := extractMapArray(versionData, "client")
	transformations := extractMapArray(versionData, "transformation")

	plan := &ImportPlan{
		Folders:         len(folders),
		Templates:       len(customTemplates),
		Variables:       len(variables),
		Triggers:        len(triggers),
		Tags:            len(tags),
		Clients:         len(clients),
		Transformations: len(transformations),
		Total:           len(folders) + len(customTemplates) + len(variables) + len(triggers) + len(tags) + len(clients) + len(transformations),
	}

	if input.DryRun {
		return nil, ImportContainerOutput{
			Success: true,
			DryRun:  true,
			Message: fmt.Sprintf("Dry-run analysis: would import %d folders, %d templates, %d variables, %d triggers, %d tags, %d clients, %d transformations (%d total entities)",
				plan.Folders, plan.Templates, plan.Variables, plan.Triggers, plan.Tags, plan.Clients, plan.Transformations, plan.Total),
			Plan: plan,
		}, nil
	}

	if !input.Confirm {
		return nil, ImportContainerOutput{
			Success: false,
			Message: fmt.Sprintf("Import requires confirm: true. This will create %d entities in the target workspace. "+
				"Use dryRun: true first to review the plan.", plan.Total),
			Plan: plan,
		}, nil
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	result := &ImportResult{
		IDMap: make(map[string]string),
	}

	for _, f := range folders {
		name := getStringField(f, "name")
		oldID := getStringField(f, "folderId")
		if name == "" {
			result.Errors = append(result.Errors, "skipped folder with empty name")
			continue
		}

		folder, err := client.CreateFolder(tCtx, input.AccountID, input.ContainerID, input.WorkspaceID, name, "")
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("folder %q (id: %s): %v", name, oldID, err))
			continue
		}
		result.FoldersCreated++
		if oldID != "" {
			result.IDMap["folder:"+oldID] = folder.FolderID
		}
	}

	for _, tmpl := range customTemplates {
		name := getStringField(tmpl, "name")
		oldID := getStringField(tmpl, "templateId")
		templateData := getStringField(tmpl, "templateData")

		if name == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("skipped template with empty name (id: %s)", oldID))
			continue
		}

		path := BuildWorkspacePath(input.AccountID, input.ContainerID, input.WorkspaceID)
		tmplReq := &tagmanager.CustomTemplate{
			Name:         name,
			TemplateData: templateData,
		}

		created, err := client.Service.Accounts.Containers.Workspaces.Templates.Create(path, tmplReq).Context(tCtx).Do()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("template %q (id: %s): %v", name, oldID, err))
			continue
		}
		result.TemplatesCreated++
		if oldID != "" {
			result.IDMap["template:"+oldID] = created.TemplateId
		}
	}

	for _, v := range variables {
		name := getStringField(v, "name")
		varType := getStringField(v, "type")
		oldID := getStringField(v, "variableId")

		if name == "" || varType == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("skipped variable with empty name or type (id: %s)", oldID))
			continue
		}

		varInput := &VariableInput{
			Name:      name,
			Type:      varType,
			Parameter: extractParameters(v),
			Notes:     getStringField(v, "notes"),
		}

		if parentFolderID := getStringField(v, "parentFolderId"); parentFolderID != "" {
			if newID, ok := result.IDMap["folder:"+parentFolderID]; ok {
				varInput.ParentFolderID = newID
			}
		}

		created, err := client.CreateVariable(tCtx, input.AccountID, input.ContainerID, input.WorkspaceID, varInput)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("variable %q (id: %s): %v", name, oldID, err))
			continue
		}
		result.VariablesCreated++
		if oldID != "" {
			result.IDMap["variable:"+oldID] = created.VariableID
		}
	}

	for _, tr := range triggers {
		name := getStringField(tr, "name")
		trigType := getStringField(tr, "type")
		oldID := getStringField(tr, "triggerId")

		if name == "" || trigType == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("skipped trigger with empty name or type (id: %s)", oldID))
			continue
		}

		trigInput := &TriggerInput{
			Name:      name,
			Type:      trigType,
			Parameter: extractParameters(tr),
			Notes:     getStringField(tr, "notes"),
		}

		trigInput.Filter = extractConditions(tr, "filter")
		trigInput.AutoEventFilter = extractConditions(tr, "autoEventFilter")
		trigInput.CustomEventFilter = extractConditions(tr, "customEventFilter")

		if eventParam, ok := tr["eventName"].(map[string]interface{}); ok {
			p := mapToParameter(eventParam)
			trigInput.EventName = &p
		}

		created, err := client.CreateTrigger(tCtx, input.AccountID, input.ContainerID, input.WorkspaceID, trigInput)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("trigger %q (id: %s): %v", name, oldID, err))
			continue
		}
		result.TriggersCreated++
		if oldID != "" {
			result.IDMap["trigger:"+oldID] = created.TriggerID
		}
	}

	for _, tg := range tags {
		name := getStringField(tg, "name")
		tagType := getStringField(tg, "type")
		oldID := getStringField(tg, "tagId")

		if name == "" || tagType == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("skipped tag with empty name or type (id: %s)", oldID))
			continue
		}

		tagInput := &TagInput{
			Name:            name,
			Type:            tagType,
			Parameter:       extractParameters(tg),
			Notes:           getStringField(tg, "notes"),
			TagFiringOption: getStringField(tg, "tagFiringOption"),
			Paused:          boolPtr(getBoolField(tg, "paused")),
		}

		tagInput.FiringTriggerId = remapTriggerIDs(
			extractStringArray(tg, "firingTriggerId"),
			result.IDMap,
		)
		tagInput.BlockingTriggerId = remapTriggerIDs(
			extractStringArray(tg, "blockingTriggerId"),
			result.IDMap,
		)

		created, err := client.CreateTag(tCtx, input.AccountID, input.ContainerID, input.WorkspaceID, tagInput)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("tag %q (id: %s): %v", name, oldID, err))
			continue
		}
		result.TagsCreated++
		if oldID != "" {
			result.IDMap["tag:"+oldID] = created.TagID
		}
	}

	for _, cl := range clients {
		name := getStringField(cl, "name")
		clientType := getStringField(cl, "type")
		oldID := getStringField(cl, "clientId")

		if name == "" || clientType == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("skipped client with empty name or type (id: %s)", oldID))
			continue
		}

		clientInput := &ClientInput{
			Name:      name,
			Type:      clientType,
			Parameter: extractParameters(cl),
			Notes:     getStringField(cl, "notes"),
		}

		if p, ok := cl["priority"].(float64); ok {
			clientInput.Priority = int64(p)
		}

		created, err := client.CreateClient(tCtx, input.AccountID, input.ContainerID, input.WorkspaceID, clientInput)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("client %q (id: %s): %v", name, oldID, err))
			continue
		}
		result.ClientsCreated++
		if oldID != "" {
			result.IDMap["client:"+oldID] = created.ClientID
		}
	}

	for _, tr := range transformations {
		name := getStringField(tr, "name")
		transType := getStringField(tr, "type")
		oldID := getStringField(tr, "transformationId")

		if name == "" || transType == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("skipped transformation with empty name or type (id: %s)", oldID))
			continue
		}

		transInput := &TransformationInput{
			Name:      name,
			Type:      transType,
			Parameter: extractParameters(tr),
			Notes:     getStringField(tr, "notes"),
		}

		created, err := client.CreateTransformation(tCtx, input.AccountID, input.ContainerID, input.WorkspaceID, transInput)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("transformation %q (id: %s): %v", name, oldID, err))
			continue
		}
		result.TransformationsCreated++
		if oldID != "" {
			result.IDMap["transformation:"+oldID] = created.TransformationID
		}
	}

	totalCreated := result.FoldersCreated + result.TemplatesCreated + result.VariablesCreated + result.TriggersCreated + result.TagsCreated + result.ClientsCreated + result.TransformationsCreated
	message := fmt.Sprintf("Import complete: %d/%d entities created (%d folders, %d templates, %d variables, %d triggers, %d tags, %d clients, %d transformations)",
		totalCreated, plan.Total,
		result.FoldersCreated, result.TemplatesCreated, result.VariablesCreated,
		result.TriggersCreated, result.TagsCreated, result.ClientsCreated, result.TransformationsCreated)

	if len(result.Errors) > 0 {
		message += fmt.Sprintf(". %d errors encountered.", len(result.Errors))
	}

	return nil, ImportContainerOutput{
		Success: len(result.Errors) == 0,
		DryRun:  false,
		Message: message,
		Plan:    plan,
		Result:  result,
	}, nil
}

func getContainerVersionData(exportData map[string]interface{}) (map[string]interface{}, bool) {
	if cv, ok := exportData["containerVersion"].(map[string]interface{}); ok {
		return cv, true
	}
	if _, hasTag := exportData["tag"]; hasTag {
		return exportData, true
	}
	if _, hasTrigger := exportData["trigger"]; hasTrigger {
		return exportData, true
	}
	return nil, false
}

func detectFormat(versionData map[string]interface{}) string {
	if bivs, ok := versionData["builtInVariable"].([]interface{}); ok && len(bivs) > 0 {
		if biv, ok := bivs[0].(map[string]interface{}); ok {
			if t, ok := biv["type"].(string); ok {
				if t == strings.ToUpper(t) && strings.Contains(t, "_") {
					return "ui"
				}
				return "api"
			}
		}
	}
	if triggers, ok := versionData["trigger"].([]interface{}); ok && len(triggers) > 0 {
		if tr, ok := triggers[0].(map[string]interface{}); ok {
			if t, ok := tr["type"].(string); ok {
				if t == strings.ToUpper(t) {
					return "ui"
				}
				return "api"
			}
		}
	}
	return "api"
}

func extractMapArray(parent map[string]interface{}, key string) []map[string]interface{} {
	arr, ok := parent[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]map[string]interface{}, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result
}

func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBoolField(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func extractParameters(entity map[string]interface{}) []Parameter {
	paramArr, ok := entity["parameter"].([]interface{})
	if !ok {
		return nil
	}
	result := make([]Parameter, 0, len(paramArr))
	for _, p := range paramArr {
		if pm, ok := p.(map[string]interface{}); ok {
			result = append(result, mapToParameter(pm))
		}
	}
	return result
}

func mapToParameter(m map[string]interface{}) Parameter {
	p := Parameter{
		Type:  getStringField(m, "type"),
		Key:   getStringField(m, "key"),
		Value: getStringField(m, "value"),
	}
	if listArr, ok := m["list"].([]interface{}); ok {
		for _, item := range listArr {
			if pm, ok := item.(map[string]interface{}); ok {
				p.List = append(p.List, mapToParameter(pm))
			}
		}
	}
	if mapArr, ok := m["map"].([]interface{}); ok {
		for _, item := range mapArr {
			if pm, ok := item.(map[string]interface{}); ok {
				p.Map = append(p.Map, mapToParameter(pm))
			}
		}
	}
	return p
}

func extractConditions(trigger map[string]interface{}, key string) []Condition {
	condArr, ok := trigger[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]Condition, 0, len(condArr))
	for _, c := range condArr {
		if cm, ok := c.(map[string]interface{}); ok {
			cond := Condition{
				Type:      getStringField(cm, "type"),
				Parameter: extractParameters(cm),
			}
			result = append(result, cond)
		}
	}
	return result
}

func extractStringArray(m map[string]interface{}, key string) []string {
	arr, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func remapTriggerIDs(oldIDs []string, idMap map[string]string) []string {
	if len(oldIDs) == 0 {
		return nil
	}
	result := make([]string, 0, len(oldIDs))
	for _, oldID := range oldIDs {
		if newID, ok := idMap["trigger:"+oldID]; ok {
			result = append(result, newID)
		} else {
			result = append(result, oldID)
		}
	}
	return result
}

var _ = tagmanager.Tag{}
