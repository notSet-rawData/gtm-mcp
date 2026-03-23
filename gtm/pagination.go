package gtm

import "reflect"

// =============================================================================
// Pagination & Compact Mode
// =============================================================================

// ListParams holds optional pagination and compact parameters for list actions.
type ListParams struct {
	Limit   int  // 0 = no limit (return all)
	Offset  int  // 0-based offset
	Compact bool // true = return only essential fields
}

// PaginationMeta contains metadata for paginated responses.
type PaginationMeta struct {
	Total   int  `json:"total"`
	Limit   int  `json:"limit,omitempty"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"hasMore"`
}

// extractListParams reads limit, offset, compact from the args map and removes
// them so they don't interfere with resource-specific arg parsing.
// Default: compact=true, limit=0 (all), offset=0.
func extractListParams(args map[string]interface{}) ListParams {
	params := ListParams{Compact: true} // compact by default

	if v, ok := args["limit"]; ok {
		params.Limit = toInt(v)
		delete(args, "limit")
	}
	if v, ok := args["offset"]; ok {
		params.Offset = toInt(v)
		delete(args, "offset")
	}
	if v, ok := args["compact"]; ok {
		params.Compact = toBool(v)
		delete(args, "compact")
	}

	return params
}

// toInt converts interface{} (float64 from JSON) to int.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

// toBool converts interface{} to bool.
func toBool(v interface{}) bool {
	switch b := v.(type) {
	case bool:
		return b
	default:
		return true // default to compact
	}
}

// =============================================================================
// Compact types — only essential fields for list responses
// =============================================================================

type CompactTag struct {
	TagID  string `json:"tagId"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Paused bool   `json:"paused,omitempty"`
}

type CompactTrigger struct {
	TriggerID      string `json:"triggerId"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	ParentFolderID string `json:"parentFolderId,omitempty"`
}

type CompactVariable struct {
	VariableID string `json:"variableId"`
	Name       string `json:"name"`
	Type       string `json:"type"`
}

type CompactClient struct {
	ClientID string `json:"clientId"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

type CompactTransformation struct {
	TransformationID string `json:"transformationId"`
	Name             string `json:"name"`
	Type             string `json:"type"`
}

type CompactFolder struct {
	FolderID string `json:"folderId"`
	Name     string `json:"name"`
}

type CompactTemplate struct {
	TemplateID string `json:"templateId"`
	Name       string `json:"name"`
	Type       string `json:"type"`
}

// =============================================================================
// Compact converters
// =============================================================================

func tagsToCompact(tags []Tag) []CompactTag {
	out := make([]CompactTag, len(tags))
	for i, t := range tags {
		out[i] = CompactTag{TagID: t.TagID, Name: t.Name, Type: t.Type, Paused: t.Paused}
	}
	return out
}

func triggersToCompact(triggers []Trigger) []CompactTrigger {
	out := make([]CompactTrigger, len(triggers))
	for i, t := range triggers {
		out[i] = CompactTrigger{TriggerID: t.TriggerID, Name: t.Name, Type: t.Type, ParentFolderID: t.ParentFolderID}
	}
	return out
}

func variablesToCompact(variables []Variable) []CompactVariable {
	out := make([]CompactVariable, len(variables))
	for i, v := range variables {
		out[i] = CompactVariable{VariableID: v.VariableID, Name: v.Name, Type: v.Type}
	}
	return out
}

func clientsToCompact(clients []ClientInfo) []CompactClient {
	out := make([]CompactClient, len(clients))
	for i, c := range clients {
		out[i] = CompactClient{ClientID: c.ClientID, Name: c.Name, Type: c.Type}
	}
	return out
}

func transformationsToCompact(transformations []TransformationInfo) []CompactTransformation {
	out := make([]CompactTransformation, len(transformations))
	for i, t := range transformations {
		out[i] = CompactTransformation{TransformationID: t.TransformationID, Name: t.Name, Type: t.Type}
	}
	return out
}

func foldersToCompact(folders []Folder) []CompactFolder {
	out := make([]CompactFolder, len(folders))
	for i, f := range folders {
		out[i] = CompactFolder{FolderID: f.FolderID, Name: f.Name}
	}
	return out
}

func templatesToCompact(templates []TemplateInfo) []CompactTemplate {
	out := make([]CompactTemplate, len(templates))
	for i, t := range templates {
		out[i] = CompactTemplate{TemplateID: t.TemplateID, Name: t.Name, Type: t.Type}
	}
	return out
}

// =============================================================================
// Paginated output wrappers
// =============================================================================

// PaginatedOutput wraps any list output with pagination metadata.
type PaginatedOutput struct {
	Items      interface{}     `json:"items"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

// applyListOptimizations applies compact mode and pagination to list outputs.
// It detects known output types and transforms them accordingly.
func applyListOptimizations(output any, params ListParams) any {
	if output == nil {
		return output
	}

	switch out := output.(type) {
	case ListTagsOutput:
		return applyToSlice(out.Tags, params, tagsToCompact, "tags")
	case ListTriggersOutput:
		return applyToSlice(out.Triggers, params, triggersToCompact, "triggers")
	case ListVariablesOutput:
		return applyToSlice(out.Variables, params, variablesToCompact, "variables")
	case ListClientsOutput:
		return applyToSlice(out.Clients, params, clientsToCompact, "clients")
	case ListTransformationsOutput:
		return applyToSlice(out.Transformations, params, transformationsToCompact, "transformations")
	case ListFoldersOutput:
		return applyToSlice(out.Folders, params, foldersToCompact, "folders")
	case ListTemplatesOutput:
		return applyToSlice(out.Templates, params, templatesToCompact, "templates")
	default:
		// For types without compact support (accounts, workspaces, environments, etc.),
		// still apply pagination if requested.
		return applyPaginationToUnknown(output, params)
	}
}

// applyToSlice applies compact conversion and pagination to a typed slice.
func applyToSlice[T any, C any](items []T, params ListParams, compactFn func([]T) []C, key string) map[string]any {
	total := len(items)

	// Apply pagination to full items first
	paginated, meta := paginate(items, params)

	if params.Compact {
		compact := compactFn(paginated)
		result := map[string]any{key: compact}
		if meta != nil {
			result["pagination"] = meta
		}
		return result
	}

	result := map[string]any{key: paginated}
	if meta != nil {
		_ = total // suppress unused
		result["pagination"] = meta
	}
	return result
}

// paginate applies limit/offset to a slice and returns the paginated slice + metadata.
func paginate[T any](items []T, params ListParams) ([]T, *PaginationMeta) {
	total := len(items)

	// No pagination requested
	if params.Limit <= 0 && params.Offset <= 0 {
		return items, nil
	}

	// Apply offset
	offset := params.Offset
	if offset > total {
		offset = total
	}
	items = items[offset:]

	// Apply limit
	limit := params.Limit
	if limit <= 0 {
		limit = len(items) // no limit but with offset
	}
	hasMore := len(items) > limit
	if limit < len(items) {
		items = items[:limit]
	}

	meta := &PaginationMeta{
		Total:   total,
		Limit:   params.Limit,
		Offset:  params.Offset,
		HasMore: hasMore,
	}

	return items, meta
}

// applyPaginationToUnknown applies pagination to unknown output types using reflection.
// It looks for the first slice field in the struct and paginates it.
func applyPaginationToUnknown(output any, params ListParams) any {
	if params.Limit <= 0 && params.Offset <= 0 {
		return output // nothing to do
	}

	v := reflect.ValueOf(output)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return output
	}

	// Find the first slice field
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() == reflect.Slice {
			total := field.Len()
			offset := params.Offset
			if offset > total {
				offset = total
			}
			limit := params.Limit
			if limit <= 0 {
				limit = total - offset
			}
			end := offset + limit
			if end > total {
				end = total
			}
			sliced := field.Slice(offset, end)

			// Build result map
			fieldName := v.Type().Field(i).Tag.Get("json")
			if fieldName == "" {
				fieldName = v.Type().Field(i).Name
			}
			// Strip omitempty
			if idx := len(fieldName) - 1; idx > 0 {
				for j := 0; j < len(fieldName); j++ {
					if fieldName[j] == ',' {
						fieldName = fieldName[:j]
						break
					}
				}
			}

			result := map[string]any{
				fieldName: sliced.Interface(),
				"pagination": &PaginationMeta{
					Total:   total,
					Limit:   params.Limit,
					Offset:  params.Offset,
					HasMore: end < total,
				},
			}
			return result
		}
	}

	return output
}
