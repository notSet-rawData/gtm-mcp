package gtm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// BrokenReference represents a broken variable reference found in the workspace.
type BrokenReference struct {
	EntityType string `json:"entityType"` // "variable", "tag", "trigger"
	EntityName string `json:"entityName"`
	EntityID   string `json:"entityId"`
	Reference  string `json:"reference"` // The broken {{Variable Name}} reference
}

// ValidationResult contains the results of a pre-publish validation.
type ValidationResult struct {
	Valid            bool              `json:"valid"`
	BrokenReferences []BrokenReference `json:"brokenReferences,omitempty"`
	Summary          string            `json:"summary"`
}

// variableRefPattern matches GTM variable references like {{Variable Name}}
var variableRefPattern = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

// builtInVariableNames contains GTM built-in variables that don't need to exist as custom variables.
// Updated with 2024-2025 additions (Client ID, Session ID, Session Number).
// [vs. NLM:datalayer-gtm]
var builtInVariableNames = map[string]bool{
	// Page
	"Page URL": true, "Page Hostname": true, "Page Path": true,
	"Referrer": true,
	// Utilities
	"Event": true, "Container ID": true, "Container Version": true,
	"Random Number": true, "HTML ID": true, "Debug Mode": true,
	"Environment Name": true,
	// 2024-2025 new built-ins
	"Client ID": true, "Session ID": true, "Session Number": true,
	// Internal pseudo-variables (used by customEvent triggers, never user-declared)
	"_event": true,
	// Clicks
	"Click Element": true, "Click Classes": true, "Click ID": true,
	"Click Target": true, "Click URL": true, "Click Text": true,
	// Forms
	"Form Element": true, "Form Classes": true, "Form ID": true,
	"Form Target": true, "Form URL": true, "Form Text": true,
	// Errors
	"Error Message": true, "Error URL": true, "Error Line": true,
	// History
	"New History Fragment": true, "Old History Fragment": true,
	"New History State": true, "Old History State": true, "History Source": true,
	// Scroll
	"Scroll Depth Threshold": true, "Scroll Depth Units": true,
	"Scroll Direction": true,
	// Visibility
	"Element Visibility": true, "Percent Visible": true, "On-Screen Duration": true,
	// Video
	"Video Provider": true, "Video Status": true, "Video URL": true,
	"Video Title": true, "Video Duration": true, "Video Current Time": true,
	"Video Percent": true, "Video Visible": true,
	// Mobile / App
	"Advertiser Tracking Enabled": true, "Advertising Tracking Enabled": true,
	"App ID": true, "App Name": true, "App Version Code": true,
	"App Version Name": true, "Language": true, "Platform": true,
	"SDK Version": true, "Device Name": true, "Resolution": true,
	"OS Version": true, "IDFA": true,
}

// ValidateVariableReferences checks all workspace entities for broken variable references.
// This acts as a guardrail before create_version to prevent compiler errors.
func ValidateVariableReferences(tags []Tag, triggers []Trigger, variables []Variable) *ValidationResult {
	// Build set of known variable names
	knownVars := make(map[string]bool)
	for _, v := range variables {
		knownVars[v.Name] = true
	}

	var broken []BrokenReference

	// Check variables — scan Parameter field
	for _, v := range variables {
		refs := extractRefsFromAny(v.Parameter)
		for _, ref := range refs {
			if !isKnownRef(ref, knownVars) {
				broken = append(broken, BrokenReference{
					EntityType: "variable",
					EntityName: v.Name,
					EntityID:   v.VariableID,
					Reference:  ref,
				})
			}
		}
	}

	// Check tags — scan Parameter field
	for _, t := range tags {
		refs := extractRefsFromAny(t.Parameter)
		for _, ref := range refs {
			if !isKnownRef(ref, knownVars) {
				broken = append(broken, BrokenReference{
					EntityType: "tag",
					EntityName: t.Name,
					EntityID:   t.TagID,
					Reference:  ref,
				})
			}
		}
	}

	// Check triggers — scan Parameter AND all filter fields
	// [vs. NLM:datalayer-gtm] Triggers use Filter, AutoEventFilter, CustomEventFilter
	// in addition to Parameter, all of which can contain {{Variable Name}} refs.
	for _, tr := range triggers {
		var allRefs []string
		allRefs = append(allRefs, extractRefsFromAny(tr.Parameter)...)
		allRefs = append(allRefs, extractRefsFromAny(tr.Filter)...)
		allRefs = append(allRefs, extractRefsFromAny(tr.AutoEventFilter)...)
		allRefs = append(allRefs, extractRefsFromAny(tr.CustomEventFilter)...)

		for _, ref := range allRefs {
			if !isKnownRef(ref, knownVars) {
				broken = append(broken, BrokenReference{
					EntityType: "trigger",
					EntityName: tr.Name,
					EntityID:   tr.TriggerID,
					Reference:  ref,
				})
			}
		}
	}

	// Deduplicate
	broken = deduplicateRefs(broken)

	if len(broken) == 0 {
		return &ValidationResult{
			Valid:   true,
			Summary: "All variable references are valid.",
		}
	}

	// Group by entity for readable summary
	byEntity := make(map[string][]string)
	for _, b := range broken {
		key := fmt.Sprintf("%s '%s'", b.EntityType, b.EntityName)
		byEntity[key] = append(byEntity[key], b.Reference)
	}

	var lines []string
	for entity, refs := range byEntity {
		lines = append(lines, fmt.Sprintf("- %s → broken refs: %s", entity, strings.Join(refs, ", ")))
	}

	return &ValidationResult{
		Valid:            false,
		BrokenReferences: broken,
		Summary: fmt.Sprintf("Found %d broken variable references:\n%s",
			len(broken), strings.Join(lines, "\n")),
	}
}

// isKnownRef checks if a reference is a known custom variable or GTM built-in.
func isKnownRef(ref string, knownVars map[string]bool) bool {
	return knownVars[ref] || builtInVariableNames[ref]
}

// extractRefsFromAny recursively extracts {{Variable Name}} references from any parameter structure.
func extractRefsFromAny(v any) []string {
	if v == nil {
		return nil
	}

	var refs []string

	switch val := v.(type) {
	case string:
		matches := variableRefPattern.FindAllStringSubmatch(val, -1)
		for _, m := range matches {
			refs = append(refs, m[1])
		}
	case map[string]any:
		for _, child := range val {
			refs = append(refs, extractRefsFromAny(child)...)
		}
	case []any:
		for _, child := range val {
			refs = append(refs, extractRefsFromAny(child)...)
		}
	default:
		// Fallback: JSON marshal for complex types (e.g., Google API structs)
		data, err := json.Marshal(val)
		if err == nil {
			matches := variableRefPattern.FindAllStringSubmatch(string(data), -1)
			for _, m := range matches {
				refs = append(refs, m[1])
			}
		}
	}

	return refs
}

func deduplicateRefs(refs []BrokenReference) []BrokenReference {
	seen := make(map[string]bool)
	var result []BrokenReference
	for _, r := range refs {
		key := r.EntityType + "|" + r.EntityID + "|" + r.Reference
		if !seen[key] {
			seen[key] = true
			result = append(result, r)
		}
	}
	return result
}
