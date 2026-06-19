package gtm

import (
	"testing"
)

func TestValidateNumericID(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		wantErr bool
		errMsg  string
	}{
		{"valid numeric", "account ID", "12345", false, ""},
		{"valid single digit", "account ID", "0", false, ""},
		{"valid large number", "container ID", "9999999999", false, ""},
		{"empty string", "account ID", "", true, "account ID is required"},
		{"whitespace only", "account ID", "   ", true, "account ID is required"},
		{"alpha string", "account ID", "abc", true, `account ID must be numeric (got "abc")`},
		{"path traversal", "account ID", "../etc", true, `account ID must be numeric (got "../etc")`},
		{"mixed alphanumeric", "account ID", "123abc", true, `account ID must be numeric (got "123abc")`},
		{"with spaces", "container ID", "123 456", true, `container ID must be numeric (got "123 456")`},
		{"negative number", "workspace ID", "-1", true, `workspace ID must be numeric (got "-1")`},
		{"decimal", "tag ID", "12.5", true, `tag ID must be numeric (got "12.5")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNumericID(tt.field, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if err.Error() != tt.errMsg {
					t.Fatalf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateWorkspacePath(t *testing.T) {
	tests := []struct {
		name        string
		accountID   string
		containerID string
		workspaceID string
		wantErr     bool
	}{
		{"valid", "123", "456", "789", false},
		{"empty account", "", "456", "789", true},
		{"non-numeric account", "abc", "456", "789", true},
		{"non-numeric container", "123", "abc", "789", true},
		{"non-numeric workspace", "123", "456", "abc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkspacePath(tt.accountID, tt.containerID, tt.workspaceID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateContainerPath(t *testing.T) {
	tests := []struct {
		name        string
		accountID   string
		containerID string
		wantErr     bool
	}{
		{"valid", "123", "456", false},
		{"empty account", "", "456", true},
		{"path traversal", "../../etc", "456", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContainerPath(tt.accountID, tt.containerID)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateTagInput(t *testing.T) {
	tests := []struct {
		name             string
		tagName          string
		tagType          string
		firingTriggerIDs []string
		wantErr          bool
	}{
		{"valid", "My Tag", "html", []string{"1"}, false},
		{"empty name", "", "html", []string{"1"}, true},
		{"empty type", "Tag", "", []string{"1"}, true},
		{"no triggers", "Tag", "html", []string{}, true},
		{"empty trigger ID", "Tag", "html", []string{""}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTagInput(tt.tagName, tt.tagType, tt.firingTriggerIDs)
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestBuildWorkspacePath(t *testing.T) {
	got := BuildWorkspacePath("123", "456", "789")
	want := "accounts/123/containers/456/workspaces/789"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildContainerPath(t *testing.T) {
	got := BuildContainerPath("123", "456")
	want := "accounts/123/containers/456"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestValidateVariableReferences_AllValid(t *testing.T) {
	vars := []Variable{
		{VariableID: "1", Name: "DLV - pageName", Parameter: map[string]any{"value": "{{Page URL}}"}},
		{VariableID: "2", Name: "MyVar", Parameter: map[string]any{"value": "{{DLV - pageName}}"}},
	}
	tags := []Tag{
		{TagID: "10", Name: "GA4 Config", Parameter: map[string]any{"html": "{{MyVar}}"}},
	}
	triggers := []Trigger{}

	result := ValidateVariableReferences(tags, triggers, vars)
	if !result.Valid {
		t.Fatalf("expected valid, got: %s", result.Summary)
	}
}

func TestValidateVariableReferences_BrokenInVariable(t *testing.T) {
	vars := []Variable{
		{VariableID: "1", Name: "ESV - Global", Parameter: map[string]any{"value": "{{DLV - oldName}}"}},
	}

	result := ValidateVariableReferences(nil, nil, vars)
	if result.Valid {
		t.Fatal("expected invalid but got valid")
	}
	if len(result.BrokenReferences) != 1 {
		t.Fatalf("expected 1 broken ref, got %d", len(result.BrokenReferences))
	}
	if result.BrokenReferences[0].Reference != "DLV - oldName" {
		t.Fatalf("expected ref 'DLV - oldName', got %q", result.BrokenReferences[0].Reference)
	}
	if result.BrokenReferences[0].EntityType != "variable" {
		t.Fatalf("expected entityType 'variable', got %q", result.BrokenReferences[0].EntityType)
	}
}

func TestValidateVariableReferences_BuiltInNotBroken(t *testing.T) {
	vars := []Variable{
		{VariableID: "1", Name: "URL Var", Parameter: map[string]any{"value": "{{Page URL}}"}},
	}

	result := ValidateVariableReferences(nil, nil, vars)
	if !result.Valid {
		t.Fatalf("built-in 'Page URL' should not be broken: %s", result.Summary)
	}
}

func TestValidateVariableReferences_NewBuiltIns2024(t *testing.T) {
	vars := []Variable{
		{VariableID: "1", Name: "Session Var", Parameter: map[string]any{
			"a": "{{Client ID}}", "b": "{{Session ID}}", "c": "{{Session Number}}",
		}},
	}

	result := ValidateVariableReferences(nil, nil, vars)
	if !result.Valid {
		t.Fatalf("2024 built-ins should not be broken: %s", result.Summary)
	}
}

func TestValidateVariableReferences_BrokenInTriggerFilter(t *testing.T) {
	vars := []Variable{
		{VariableID: "1", Name: "MyVar"},
	}
	triggers := []Trigger{
		{
			TriggerID: "5",
			Name:      "Click Trigger",
			Filter: []any{
				map[string]any{"type": "equals", "parameter": []any{
					map[string]any{"key": "arg0", "value": "{{NonExistent}}"},
				}},
			},
		},
	}

	result := ValidateVariableReferences(nil, triggers, vars)
	if result.Valid {
		t.Fatal("expected invalid — trigger filter has broken ref")
	}
	if len(result.BrokenReferences) != 1 {
		t.Fatalf("expected 1 broken ref, got %d", len(result.BrokenReferences))
	}
	if result.BrokenReferences[0].EntityType != "trigger" {
		t.Fatalf("expected entityType 'trigger', got %q", result.BrokenReferences[0].EntityType)
	}
}

func TestValidateVariableReferences_BrokenInTag(t *testing.T) {
	vars := []Variable{}
	tags := []Tag{
		{TagID: "10", Name: "Custom HTML", Parameter: map[string]any{
			"html": "<script>var x = '{{Missing Var}}';</script>",
		}},
	}

	result := ValidateVariableReferences(tags, nil, vars)
	if result.Valid {
		t.Fatal("expected invalid — tag has broken ref")
	}
	if result.BrokenReferences[0].Reference != "Missing Var" {
		t.Fatalf("expected ref 'Missing Var', got %q", result.BrokenReferences[0].Reference)
	}
}

func TestValidateVariableReferences_Deduplication(t *testing.T) {
	vars := []Variable{
		{VariableID: "1", Name: "ESV", Parameter: map[string]any{
			"a": "{{Ghost}}", "b": "{{Ghost}}", "c": "{{Ghost}}",
		}},
	}

	result := ValidateVariableReferences(nil, nil, vars)
	if result.Valid {
		t.Fatal("expected invalid")
	}
	if len(result.BrokenReferences) != 1 {
		t.Fatalf("expected 1 deduplicated broken ref, got %d", len(result.BrokenReferences))
	}
}

func TestExtractRefsFromAny_String(t *testing.T) {
	refs := extractRefsFromAny("Hello {{Var1}} and {{Var2}}")
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
}

func TestExtractRefsFromAny_Nil(t *testing.T) {
	refs := extractRefsFromAny(nil)
	if len(refs) != 0 {
		t.Fatalf("expected 0 refs, got %d", len(refs))
	}
}

func TestExtractRefsFromAny_NestedMap(t *testing.T) {
	data := map[string]any{
		"level1": map[string]any{
			"level2": "value is {{DeepVar}}",
		},
	}
	refs := extractRefsFromAny(data)
	if len(refs) != 1 || refs[0] != "DeepVar" {
		t.Fatalf("expected [DeepVar], got %v", refs)
	}
}

func TestExtractRefsFromAny_Slice(t *testing.T) {
	data := []any{"{{A}}", "no ref", "{{B}}"}
	refs := extractRefsFromAny(data)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d: %v", len(refs), refs)
	}
}
