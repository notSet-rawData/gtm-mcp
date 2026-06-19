package gtm

import (
	"fmt"

	"google.golang.org/api/tagmanager/v2"
)

// --- Generic list entry operations for tags and variables ---
// These functions enable append/remove/list on any "list" type parameter
// inside a tag or variable's parameter array, without the caller needing
// to GET the full entity, parse 50KB+ of JSON, and re-send it.
// The MCP server handles the full GET→mutate→UPDATE cycle internally.

// ListEntry is a flat representation of a map-type entry inside a list parameter.
// Keys and values are extracted from the nested Parameter.Map structure.
type ListEntry struct {
	Fields map[string]string `json:"fields"`
}

// AppendListEntryOutput is the response for append_list_entry action.
type AppendListEntryOutput struct {
	Success          bool              `json:"success"`
	EntityType       string            `json:"entityType"`
	EntityID         string            `json:"entityId"`
	EntityName       string            `json:"entityName"`
	ListParameterKey string            `json:"listParameterKey"`
	Action           string            `json:"action"` // "appended", "skipped", "merged", "replaced"
	PreviousSize     int               `json:"previousSize"`
	CurrentSize      int               `json:"currentSize"`
	Entry            map[string]string `json:"entry,omitempty"`
	Fingerprint      string            `json:"fingerprint,omitempty"`
}

// RemoveListEntryOutput is the response for remove_list_entry action.
type RemoveListEntryOutput struct {
	Success          bool              `json:"success"`
	EntityType       string            `json:"entityType"`
	EntityID         string            `json:"entityId"`
	EntityName       string            `json:"entityName"`
	ListParameterKey string            `json:"listParameterKey"`
	Removed          bool              `json:"removed"`
	RemovedEntry     map[string]string `json:"removedEntry,omitempty"`
	PreviousSize     int               `json:"previousSize"`
	CurrentSize      int               `json:"currentSize"`
	Fingerprint      string            `json:"fingerprint,omitempty"`
	Message          string            `json:"message"`
}

// ListEntriesOutput is the response for list_entries action.
type ListEntriesOutput struct {
	EntityType       string              `json:"entityType"`
	EntityID         string              `json:"entityId"`
	EntityName       string              `json:"entityName"`
	ListParameterKey string              `json:"listParameterKey"`
	Total            int                 `json:"total"`
	Entries          []map[string]string `json:"entries"`
}

// findListParameter locates a parameter of type "list" by key name
// in any entity's parameter array. Works for tags, variables, triggers.
// Returns the index in the params slice and the list entries, or -1 and nil.
func findListParameter(params []Parameter, key string) (int, []Parameter) {
	for i, p := range params {
		if p.Key == key && p.Type == "list" {
			return i, p.List
		}
	}
	return -1, nil
}

// flattenEntry converts a nested map Parameter into a flat key→value map.
// Input:  Parameter{Type: "map", Map: [{Key: "hostname", Value: "43231"}, ...]}
// Output: {"hostname": "43231", ...}
func flattenEntry(entry Parameter) map[string]string {
	result := make(map[string]string, len(entry.Map))
	for _, field := range entry.Map {
		result[field.Key] = field.Value
	}
	return result
}

// flattenEntries converts a slice of map Parameters into flat ListEntry structs.
func flattenEntries(entries []Parameter) []ListEntry {
	result := make([]ListEntry, 0, len(entries))
	for _, e := range entries {
		if e.Type == "map" {
			result = append(result, ListEntry{Fields: flattenEntry(e)})
		}
	}
	return result
}

// isDuplicate checks if an entry already exists based on deduplication keys.
// Returns the index of the matching entry and whether it was found.
func isDuplicate(existing []Parameter, newEntry Parameter, deduplicateBy []string) (int, bool) {
	if len(deduplicateBy) == 0 {
		return -1, false // No dedup keys = always append
	}
	newFlat := flattenEntry(newEntry)
	for i, e := range existing {
		if e.Type != "map" {
			continue
		}
		flat := flattenEntry(e)
		match := true
		for _, key := range deduplicateBy {
			if flat[key] != newFlat[key] {
				match = false
				break
			}
		}
		if match {
			return i, true
		}
	}
	return -1, false
}

// MergeStrategy defines how to handle duplicate entries.
type MergeStrategy string

const (
	MergeSkip          MergeStrategy = "skip"           // Do nothing if duplicate found
	MergeExtendValue   MergeStrategy = "extend_value"   // Append to a field with comma separator (e.g. "G-AAA" → "G-AAA, G-BBB")
	MergeExtendPattern MergeStrategy = "extend_pattern" // Append to a field with pipe separator (e.g. "123" → "123|456")
	MergeReplace       MergeStrategy = "replace"        // Replace the existing entry entirely
)

// MergeConfig specifies how to merge when a duplicate is found.
type MergeConfig struct {
	Strategy  MergeStrategy `json:"strategy"`           // How to merge
	FieldKey  string        `json:"fieldKey,omitempty"` // Which field to extend (for extend_value/extend_pattern)
	Separator string        `json:"-"`                  // Determined by strategy
}

// appendListEntry appends a new entry to a list parameter, handling deduplication
// and optional merge strategies. Returns the action taken and updated entries.
func appendListEntry(entries []Parameter, newEntry Parameter, deduplicateBy []string, merge *MergeConfig) (action string, updatedEntries []Parameter, detail string) {
	idx, found := isDuplicate(entries, newEntry, deduplicateBy)

	if !found {
		// Simple append
		return "appended", append(entries, newEntry), ""
	}

	// Duplicate found — apply merge strategy
	if merge == nil || merge.Strategy == MergeSkip || merge.Strategy == "" {
		flat := flattenEntry(entries[idx])
		return "skipped", entries, fmt.Sprintf("entry already exists with matching keys: %v", flat)
	}

	switch merge.Strategy {
	case MergeExtendValue, MergeExtendPattern:
		if merge.FieldKey == "" {
			return "error", entries, "merge strategy requires fieldKey"
		}
		separator := ", "
		if merge.Strategy == MergeExtendPattern {
			separator = "|"
		}
		// Find the field to extend in the existing entry
		newFlat := flattenEntry(newEntry)
		newValue, hasNew := newFlat[merge.FieldKey]
		if !hasNew || newValue == "" {
			return "skipped", entries, fmt.Sprintf("new entry has no value for field %q", merge.FieldKey)
		}
		// Check if value already present
		for j, field := range entries[idx].Map {
			if field.Key == merge.FieldKey {
				existingValue := field.Value
				// Check if already contains the new value
				if containsValue(existingValue, newValue, separator) {
					return "skipped", entries, fmt.Sprintf("value %q already present in field %q", newValue, merge.FieldKey)
				}
				entries[idx].Map[j].Value = existingValue + separator + newValue
				return "merged", entries, fmt.Sprintf("extended field %q: %q → %q", merge.FieldKey, existingValue, entries[idx].Map[j].Value)
			}
		}
		return "error", entries, fmt.Sprintf("field %q not found in existing entry", merge.FieldKey)

	case MergeReplace:
		entries[idx] = newEntry
		return "replaced", entries, "existing entry replaced"

	default:
		return "error", entries, fmt.Sprintf("unknown merge strategy: %q", merge.Strategy)
	}
}

// containsValue checks if a separator-delimited string already contains a value.
func containsValue(existing, value, separator string) bool {
	if existing == value {
		return true
	}
	// Split by separator and check each part (trimmed)
	parts := splitTrimmed(existing, separator)
	for _, part := range parts {
		if part == value {
			return true
		}
	}
	return false
}

// splitTrimmed splits a string by separator and trims whitespace from each part.
func splitTrimmed(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			part := trimSpace(s[start:i])
			if part != "" {
				result = append(result, part)
			}
			start = i + len(sep)
		}
	}
	part := trimSpace(s[start:])
	if part != "" {
		result = append(result, part)
	}
	return result
}

// trimSpace trims leading and trailing spaces from a string.
func trimSpace(s string) string {
	start := 0
	for start < len(s) && s[start] == ' ' {
		start++
	}
	end := len(s)
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}

// removeListEntry removes entries from a list parameter that match the given keys.
// Returns the removed entries and the remaining list.
func removeListEntry(entries []Parameter, matchBy map[string]string) (removed []ListEntry, remaining []Parameter) {
	if len(matchBy) == 0 {
		return nil, entries
	}
	for _, e := range entries {
		if e.Type != "map" {
			remaining = append(remaining, e)
			continue
		}
		flat := flattenEntry(e)
		match := true
		for k, v := range matchBy {
			if flat[k] != v {
				match = false
				break
			}
		}
		if match {
			removed = append(removed, ListEntry{Fields: flat})
		} else {
			remaining = append(remaining, e)
		}
	}
	return removed, remaining
}

// extractEntryKeys returns the ordered list of map keys from the first entry.
// Used to preserve field ordering when the caller inspects results.
func extractEntryKeys(entries []Parameter) []string {
	for _, e := range entries {
		if e.Type == "map" && len(e.Map) > 0 {
			keys := make([]string, 0, len(e.Map))
			for _, field := range e.Map {
				keys = append(keys, field.Key)
			}
			return keys
		}
	}
	return nil
}

// getTagParams extracts the Parameter slice from a Tag.
// Tag.Parameter is typed as `any` but stores []*tagmanager.Parameter.
func getTagParams(t *Tag) []Parameter {
	apiParams, ok := t.Parameter.([]*tagmanager.Parameter)
	if !ok || len(apiParams) == 0 {
		return nil
	}
	result := make([]Parameter, 0, len(apiParams))
	for _, p := range apiParams {
		result = append(result, apiParamToParameter(p))
	}
	return result
}
