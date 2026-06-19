package gtm

import (
	"encoding/json"
	"testing"
)

func TestConvertEnumsToScreamingCase(t *testing.T) {
	input := map[string]interface{}{
		"container": map[string]interface{}{
			"usageContext": []interface{}{"web"},
			"name":         "My Container",
		},
		"tag": []interface{}{
			map[string]interface{}{
				"name": "GA4 config",
				"type": "googtag", // tag type - should NOT be converted
				"parameter": []interface{}{
					map[string]interface{}{"key": "tagId", "type": "template", "value": "G-XXX"},
					map[string]interface{}{"key": "sendPageView", "type": "boolean", "value": "false"},
				},
				"tagFiringOption": "oncePerEvent",
				"monitoringMetadata": map[string]interface{}{
					"type": "map",
				},
				"consentSettings": map[string]interface{}{
					"consentStatus": "notSet",
				},
			},
		},
		"trigger": []interface{}{
			map[string]interface{}{
				"name": "Custom Event",
				"type": "customEvent",
				"customEventFilter": []interface{}{
					map[string]interface{}{
						"type": "matchRegex",
						"parameter": []interface{}{
							map[string]interface{}{"key": "arg0", "type": "template"},
							map[string]interface{}{"key": "arg1", "type": "template"},
						},
					},
				},
			},
		},
		"variable": []interface{}{
			map[string]interface{}{
				"name": "DLV test",
				"type": "v", // variable type - should NOT be converted
				"parameter": []interface{}{
					map[string]interface{}{"key": "name", "type": "template", "value": "test"},
					map[string]interface{}{"key": "dataLayerVersion", "type": "integer", "value": "2"},
				},
			},
		},
		"builtInVariable": []interface{}{
			map[string]interface{}{
				"type": "pageUrl",
				"name": "Page URL",
			},
			map[string]interface{}{
				"type": "event",
				"name": "Event",
			},
		},
	}

	convertEnumsToScreamingCase(input)

	out, _ := json.MarshalIndent(input, "", "  ")
	t.Logf("Result:\n%s", string(out))

	container := input["container"].(map[string]interface{})
	usageCtx := container["usageContext"].([]interface{})
	if usageCtx[0] != "WEB" {
		t.Errorf("usageContext: got %v, want WEB", usageCtx[0])
	}

	tags := input["tag"].([]interface{})
	tag0 := tags[0].(map[string]interface{})
	if tag0["type"] != "googtag" {
		t.Errorf("tag type: got %v, want googtag (should NOT be converted)", tag0["type"])
	}

	params := tag0["parameter"].([]interface{})
	param0 := params[0].(map[string]interface{})
	if param0["type"] != "TEMPLATE" {
		t.Errorf("param type[0]: got %v, want TEMPLATE", param0["type"])
	}

	param1 := params[1].(map[string]interface{})
	if param1["type"] != "BOOLEAN" {
		t.Errorf("param type[1]: got %v, want BOOLEAN", param1["type"])
	}

	if tag0["tagFiringOption"] != "ONCE_PER_EVENT" {
		t.Errorf("tagFiringOption: got %v, want ONCE_PER_EVENT", tag0["tagFiringOption"])
	}

	cs := tag0["consentSettings"].(map[string]interface{})
	if cs["consentStatus"] != "NOT_SET" {
		t.Errorf("consentStatus: got %v, want NOT_SET", cs["consentStatus"])
	}

	mm := tag0["monitoringMetadata"].(map[string]interface{})
	if mm["type"] != "MAP" {
		t.Errorf("monitoringMetadata type: got %v, want MAP", mm["type"])
	}

	triggers := input["trigger"].([]interface{})
	trigger0 := triggers[0].(map[string]interface{})
	if trigger0["type"] != "CUSTOM_EVENT" {
		t.Errorf("trigger type: got %v, want CUSTOM_EVENT", trigger0["type"])
	}

	filters := trigger0["customEventFilter"].([]interface{})
	filter0 := filters[0].(map[string]interface{})
	if filter0["type"] != "MATCH_REGEX" {
		t.Errorf("condition type: got %v, want MATCH_REGEX", filter0["type"])
	}

	vars := input["variable"].([]interface{})
	var0 := vars[0].(map[string]interface{})
	if var0["type"] != "v" {
		t.Errorf("variable type: got %v, want v (should NOT be converted)", var0["type"])
	}

	varParams := var0["parameter"].([]interface{})
	varParam1 := varParams[1].(map[string]interface{})
	if varParam1["type"] != "INTEGER" {
		t.Errorf("var param type: got %v, want INTEGER", varParam1["type"])
	}

	bivs := input["builtInVariable"].([]interface{})
	biv0 := bivs[0].(map[string]interface{})
	if biv0["type"] != "PAGE_URL" {
		t.Errorf("builtInVariable type[0]: got %v, want PAGE_URL", biv0["type"])
	}

	biv1 := bivs[1].(map[string]interface{})
	if biv1["type"] != "EVENT" {
		t.Errorf("builtInVariable type[1]: got %v, want EVENT", biv1["type"])
	}
}
