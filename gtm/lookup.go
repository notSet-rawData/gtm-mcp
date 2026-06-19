package gtm

type LookupEntry struct {
	Pattern string            `json:"pattern"`
	Output  string            `json:"output"`
	Extra   map[string]string `json:"extra,omitempty"` // Additional map fields (e.g. inputsTable keys)
}

func findRegexTable(params []Parameter) (int, []Parameter) {
	for i, p := range params {
		if p.Key == "regexTable" && p.Type == "list" {
			return i, p.List
		}
	}
	return -1, nil
}

func parseRegexEntry(entry Parameter) LookupEntry {
	le := LookupEntry{
		Extra: make(map[string]string),
	}
	for _, field := range entry.Map {
		switch field.Key {
		case "pattern":
			le.Pattern = field.Value
		case "output":
			le.Output = field.Value
		default:
			le.Extra[field.Key] = field.Value
		}
	}
	if len(le.Extra) == 0 {
		le.Extra = nil
	}
	return le
}

func buildRegexEntry(entry LookupEntry, existingKeys []string) Parameter {
	p := Parameter{
		Type: "map",
	}

	if len(existingKeys) > 0 {
		keySet := make(map[string]bool)
		for _, k := range existingKeys {
			keySet[k] = true
			var value string
			switch k {
			case "pattern":
				value = entry.Pattern
			case "output":
				value = entry.Output
			default:
				if entry.Extra != nil {
					value = entry.Extra[k]
				}
			}
			p.Map = append(p.Map, Parameter{
				Type:  "template",
				Key:   k,
				Value: value,
			})
		}
		if entry.Extra != nil {
			for k, v := range entry.Extra {
				if !keySet[k] {
					p.Map = append(p.Map, Parameter{
						Type:  "template",
						Key:   k,
						Value: v,
					})
				}
			}
		}
	} else {
		p.Map = append(p.Map, Parameter{
			Type:  "template",
			Key:   "pattern",
			Value: entry.Pattern,
		})
		p.Map = append(p.Map, Parameter{
			Type:  "template",
			Key:   "output",
			Value: entry.Output,
		})
		if entry.Extra != nil {
			for k, v := range entry.Extra {
				p.Map = append(p.Map, Parameter{
					Type:  "template",
					Key:   k,
					Value: v,
				})
			}
		}
	}

	return p
}

func extractExistingKeys(entries []Parameter) []string {
	if len(entries) == 0 {
		return nil
	}
	keys := make([]string, 0, len(entries[0].Map))
	for _, field := range entries[0].Map {
		keys = append(keys, field.Key)
	}
	return keys
}

func mergeEntries(existing []Parameter, newEntries []LookupEntry) (added []LookupEntry, duplicates []LookupEntry, merged []Parameter) {
	existingPatterns := make(map[string]bool, len(existing))
	for _, entry := range existing {
		le := parseRegexEntry(entry)
		existingPatterns[le.Pattern] = true
	}

	merged = make([]Parameter, len(existing))
	copy(merged, existing)

	existingKeys := extractExistingKeys(existing)

	for _, newEntry := range newEntries {
		if existingPatterns[newEntry.Pattern] {
			duplicates = append(duplicates, newEntry)
		} else {
			added = append(added, newEntry)
			merged = append(merged, buildRegexEntry(newEntry, existingKeys))
			existingPatterns[newEntry.Pattern] = true // Prevent duplicates within newEntries too
		}
	}

	return added, duplicates, merged
}

func removeEntries(existing []Parameter, patterns []string) (removed []LookupEntry, remaining []Parameter) {
	patternSet := make(map[string]bool, len(patterns))
	for _, p := range patterns {
		patternSet[p] = true
	}

	for _, entry := range existing {
		le := parseRegexEntry(entry)
		if patternSet[le.Pattern] {
			removed = append(removed, le)
		} else {
			remaining = append(remaining, entry)
		}
	}

	return removed, remaining
}
