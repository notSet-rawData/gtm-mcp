package gtm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type BrokenReference struct {
	EntityType string `json:"entityType"` // "variable", "tag", "trigger"
	EntityName string `json:"entityName"`
	EntityID   string `json:"entityId"`
	Reference  string `json:"reference"` // The broken {{Variable Name}} reference
}

type ValidationResult struct {
	Valid            bool              `json:"valid"`
	BrokenReferences []BrokenReference `json:"brokenReferences,omitempty"`
	Summary          string            `json:"summary"`
}

var variableRefPattern = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

var builtInVariableNames = map[string]bool{
	"Page URL": true, "Page Hostname": true, "Page Path": true,
	"Referrer": true,
	"Event":    true, "Container ID": true, "Container Version": true,
	"Random Number": true, "HTML ID": true, "Debug Mode": true,
	"Environment Name": true,
	"Client ID":        true, "Session ID": true, "Session Number": true,
	"_event":        true,
	"Click Element": true, "Click Classes": true, "Click ID": true,
	"Click Target": true, "Click URL": true, "Click Text": true,
	"Form Element": true, "Form Classes": true, "Form ID": true,
	"Form Target": true, "Form URL": true, "Form Text": true,
	"Error Message": true, "Error URL": true, "Error Line": true,
	"New History Fragment": true, "Old History Fragment": true,
	"New History State": true, "Old History State": true, "History Source": true,
	"Scroll Depth Threshold": true, "Scroll Depth Units": true,
	"Scroll Direction":   true,
	"Element Visibility": true, "Percent Visible": true, "On-Screen Duration": true,
	"Video Provider": true, "Video Status": true, "Video URL": true,
	"Video Title": true, "Video Duration": true, "Video Current Time": true,
	"Video Percent": true, "Video Visible": true,
	"Advertiser Tracking Enabled": true, "Advertising Tracking Enabled": true,
	"App ID": true, "App Name": true, "App Version Code": true,
	"App Version Name": true, "Language": true, "Platform": true,
	"SDK Version": true, "Device Name": true, "Resolution": true,
	"OS Version": true, "IDFA": true,
}

func ValidateVariableReferences(tags []Tag, triggers []Trigger, variables []Variable) *ValidationResult {
	knownVars := make(map[string]bool)
	for _, v := range variables {
		knownVars[v.Name] = true
	}

	var broken []BrokenReference

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

	broken = deduplicateRefs(broken)

	if len(broken) == 0 {
		return &ValidationResult{
			Valid:   true,
			Summary: "All variable references are valid.",
		}
	}

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

func isKnownRef(ref string, knownVars map[string]bool) bool {
	return knownVars[ref] || builtInVariableNames[ref]
}

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
