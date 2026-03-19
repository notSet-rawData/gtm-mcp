package gtm

import (
	"testing"
)

// Helper to build a regex table parameter with entries
func buildTestRegexTable(entries ...[]struct{ key, value string }) Parameter {
	p := Parameter{
		Type: "list",
		Key:  "regexTable",
	}
	for _, fields := range entries {
		entry := Parameter{Type: "map"}
		for _, f := range fields {
			entry.Map = append(entry.Map, Parameter{
				Type:  "template",
				Key:   f.key,
				Value: f.value,
			})
		}
		p.List = append(p.List, entry)
	}
	return p
}

func TestFindRegexTable(t *testing.T) {
	params := []Parameter{
		{Type: "template", Key: "inputsTable", Value: ""},
		{Type: "list", Key: "regexTable", List: []Parameter{
			{Type: "map", Map: []Parameter{
				{Type: "template", Key: "pattern", Value: "^test$"},
				{Type: "template", Key: "output", Value: "123"},
			}},
		}},
		{Type: "template", Key: "defaultValue", Value: "0"},
	}

	idx, entries := findRegexTable(params)
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestFindRegexTable_NotFound(t *testing.T) {
	params := []Parameter{
		{Type: "template", Key: "value", Value: "hello"},
	}

	idx, entries := findRegexTable(params)
	if idx != -1 {
		t.Fatalf("expected index -1, got %d", idx)
	}
	if entries != nil {
		t.Fatal("expected nil entries")
	}
}

func TestParseRegexEntry(t *testing.T) {
	entry := Parameter{
		Type: "map",
		Map: []Parameter{
			{Type: "template", Key: "pattern", Value: "^11983_599902$"},
			{Type: "template", Key: "output", Value: "1345093174093364"},
			{Type: "template", Key: "inputsTable", Value: ""},
		},
	}

	le := parseRegexEntry(entry)
	if le.Pattern != "^11983_599902$" {
		t.Fatalf("expected pattern '^11983_599902$', got %q", le.Pattern)
	}
	if le.Output != "1345093174093364" {
		t.Fatalf("expected output '1345093174093364', got %q", le.Output)
	}
	if le.Extra == nil || le.Extra["inputsTable"] != "" {
		t.Fatalf("expected extra with inputsTable key, got %v", le.Extra)
	}
}

func TestParseRegexEntry_MinimalFields(t *testing.T) {
	entry := Parameter{
		Type: "map",
		Map: []Parameter{
			{Type: "template", Key: "pattern", Value: "^test$"},
			{Type: "template", Key: "output", Value: "val"},
		},
	}

	le := parseRegexEntry(entry)
	if le.Pattern != "^test$" || le.Output != "val" {
		t.Fatalf("unexpected values: %+v", le)
	}
	if le.Extra != nil {
		t.Fatalf("expected nil extra, got %v", le.Extra)
	}
}

func TestBuildRegexEntry(t *testing.T) {
	entry := LookupEntry{
		Pattern: "^new$",
		Output:  "newval",
	}
	existingKeys := []string{"pattern", "output", "inputsTable"}

	p := buildRegexEntry(entry, existingKeys)

	if p.Type != "map" {
		t.Fatalf("expected type 'map', got %q", p.Type)
	}
	if len(p.Map) != 3 {
		t.Fatalf("expected 3 map fields, got %d", len(p.Map))
	}

	// Verify key ordering matches existing
	if p.Map[0].Key != "pattern" || p.Map[0].Value != "^new$" {
		t.Fatalf("first field should be pattern=^new$, got %s=%s", p.Map[0].Key, p.Map[0].Value)
	}
	if p.Map[1].Key != "output" || p.Map[1].Value != "newval" {
		t.Fatalf("second field should be output=newval, got %s=%s", p.Map[1].Key, p.Map[1].Value)
	}
	if p.Map[2].Key != "inputsTable" || p.Map[2].Value != "" {
		t.Fatalf("third field should be inputsTable='', got %s=%s", p.Map[2].Key, p.Map[2].Value)
	}
}

func TestBuildRegexEntry_NoExistingKeys(t *testing.T) {
	entry := LookupEntry{
		Pattern: "^test$",
		Output:  "val",
	}

	p := buildRegexEntry(entry, nil)
	if len(p.Map) != 2 {
		t.Fatalf("expected 2 map fields, got %d", len(p.Map))
	}
	if p.Map[0].Key != "pattern" || p.Map[1].Key != "output" {
		t.Fatal("expected pattern then output")
	}
}

func TestExtractExistingKeys(t *testing.T) {
	entries := []Parameter{
		{Type: "map", Map: []Parameter{
			{Type: "template", Key: "pattern"},
			{Type: "template", Key: "output"},
			{Type: "template", Key: "inputsTable"},
		}},
	}

	keys := extractExistingKeys(entries)
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if keys[0] != "pattern" || keys[1] != "output" || keys[2] != "inputsTable" {
		t.Fatalf("unexpected keys: %v", keys)
	}
}

func TestExtractExistingKeys_Empty(t *testing.T) {
	keys := extractExistingKeys(nil)
	if keys != nil {
		t.Fatalf("expected nil, got %v", keys)
	}
}

func TestMergeEntries_NoDuplicates(t *testing.T) {
	existing := []Parameter{
		{Type: "map", Map: []Parameter{
			{Type: "template", Key: "pattern", Value: "^old$"},
			{Type: "template", Key: "output", Value: "oldval"},
		}},
	}

	newEntries := []LookupEntry{
		{Pattern: "^new1$", Output: "val1"},
		{Pattern: "^new2$", Output: "val2"},
	}

	added, duplicates, merged := mergeEntries(existing, newEntries)

	if len(added) != 2 {
		t.Fatalf("expected 2 added, got %d", len(added))
	}
	if len(duplicates) != 0 {
		t.Fatalf("expected 0 duplicates, got %d", len(duplicates))
	}
	if len(merged) != 3 {
		t.Fatalf("expected 3 merged entries, got %d", len(merged))
	}
}

func TestMergeEntries_WithDuplicates(t *testing.T) {
	existing := []Parameter{
		{Type: "map", Map: []Parameter{
			{Type: "template", Key: "pattern", Value: "^existing$"},
			{Type: "template", Key: "output", Value: "existval"},
		}},
	}

	newEntries := []LookupEntry{
		{Pattern: "^existing$", Output: "newval"},  // duplicate
		{Pattern: "^fresh$", Output: "freshval"},    // new
	}

	added, duplicates, merged := mergeEntries(existing, newEntries)

	if len(added) != 1 {
		t.Fatalf("expected 1 added, got %d", len(added))
	}
	if len(duplicates) != 1 {
		t.Fatalf("expected 1 duplicate, got %d", len(duplicates))
	}
	if duplicates[0].Pattern != "^existing$" {
		t.Fatalf("expected duplicate pattern '^existing$', got %q", duplicates[0].Pattern)
	}
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged entries, got %d", len(merged))
	}
}

func TestMergeEntries_DuplicatesWithinNewEntries(t *testing.T) {
	existing := []Parameter{}

	newEntries := []LookupEntry{
		{Pattern: "^same$", Output: "val1"},
		{Pattern: "^same$", Output: "val2"}, // duplicate within batch
	}

	added, duplicates, merged := mergeEntries(existing, newEntries)

	if len(added) != 1 {
		t.Fatalf("expected 1 added, got %d", len(added))
	}
	if len(duplicates) != 1 {
		t.Fatalf("expected 1 duplicate, got %d", len(duplicates))
	}
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged entry, got %d", len(merged))
	}
}

func TestRemoveEntries(t *testing.T) {
	existing := []Parameter{
		{Type: "map", Map: []Parameter{
			{Type: "template", Key: "pattern", Value: "^keep$"},
			{Type: "template", Key: "output", Value: "keepval"},
		}},
		{Type: "map", Map: []Parameter{
			{Type: "template", Key: "pattern", Value: "^remove$"},
			{Type: "template", Key: "output", Value: "removeval"},
		}},
		{Type: "map", Map: []Parameter{
			{Type: "template", Key: "pattern", Value: "^also_keep$"},
			{Type: "template", Key: "output", Value: "alsoval"},
		}},
	}

	removed, remaining := removeEntries(existing, []string{"^remove$"})

	if len(removed) != 1 {
		t.Fatalf("expected 1 removed, got %d", len(removed))
	}
	if removed[0].Pattern != "^remove$" {
		t.Fatalf("expected removed pattern '^remove$', got %q", removed[0].Pattern)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 remaining, got %d", len(remaining))
	}
}

func TestRemoveEntries_PatternNotFound(t *testing.T) {
	existing := []Parameter{
		{Type: "map", Map: []Parameter{
			{Type: "template", Key: "pattern", Value: "^keep$"},
			{Type: "template", Key: "output", Value: "keepval"},
		}},
	}

	removed, remaining := removeEntries(existing, []string{"^nonexistent$"})

	if len(removed) != 0 {
		t.Fatalf("expected 0 removed, got %d", len(removed))
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(remaining))
	}
}

func TestRoundTrip_ParseBuild(t *testing.T) {
	// Verify that parse → build preserves data
	original := Parameter{
		Type: "map",
		Map: []Parameter{
			{Type: "template", Key: "pattern", Value: "^test_123$"},
			{Type: "template", Key: "output", Value: "AW-123456"},
			{Type: "template", Key: "inputsTable", Value: ""},
		},
	}

	parsed := parseRegexEntry(original)
	existingKeys := []string{"pattern", "output", "inputsTable"}
	rebuilt := buildRegexEntry(parsed, existingKeys)

	if len(rebuilt.Map) != len(original.Map) {
		t.Fatalf("expected %d fields, got %d", len(original.Map), len(rebuilt.Map))
	}

	for i, field := range original.Map {
		if rebuilt.Map[i].Key != field.Key || rebuilt.Map[i].Value != field.Value {
			t.Fatalf("field %d mismatch: expected %s=%q, got %s=%q",
				i, field.Key, field.Value, rebuilt.Map[i].Key, rebuilt.Map[i].Value)
		}
	}
}
