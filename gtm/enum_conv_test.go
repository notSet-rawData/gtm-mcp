package gtm

import "testing"

func TestConvertEnumValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"requestPath", "REQUEST_PATH"},
		{"requestMethod", "REQUEST_METHOD"},
		{"clientName", "CLIENT_NAME"},
		{"queryString", "QUERY_STRING"},
		{"visitorRegion", "VISITOR_REGION"},
		{"pageUrl", "PAGE_URL"},
		{"debugMode", "DEBUG_MODE"},
		{"eventName", "EVENT_NAME"},
		{"clickElement", "CLICK_ELEMENT"},
		{"customEvent", "CUSTOM_EVENT"},
		{"domReady", "DOM_READY"},
		{"windowLoaded", "WINDOW_LOADED"},
		{"youTubeVideo", "YOU_TUBE_VIDEO"},
		{"template", "TEMPLATE"},
		{"boolean", "BOOLEAN"},
		{"pageview", "PAGEVIEW"},
		{"click", "CLICK"},
		{"CONTAINER_VERSION", "CONTAINER_VERSION"},
		{"REQUEST_PATH", "REQUEST_PATH"},
		{"gaawc", "gaawc"},
		{"cvt_198845464_347", "cvt_198845464_347"},
		{"oncePerEvent", "ONCE_PER_EVENT"},
		{"oncePerLoad", "ONCE_PER_LOAD"},
		{"startsWith", "STARTS_WITH"},
		{"matchRegex", "MATCH_REGEX"},
		{"notSet", "NOT_SET"},
		{"notNeeded", "NOT_NEEDED"},
		{"server", "SERVER"},
		{"web", "WEB"},
		{"someNewBuiltInVariable", "SOME_NEW_BUILT_IN_VARIABLE"},
		{"firebaseAppException", "FIREBASE_APP_EXCEPTION"},
		{"scrollDepthThreshold", "SCROLL_DEPTH_THRESHOLD"},
		{"elementVisibilityFirstTime", "ELEMENT_VISIBILITY_FIRST_TIME"},
		{"consentInit", "CONSENT_INIT"},
		{"serverPageview", "SERVER_PAGEVIEW"},
		{"triggerReference", "TRIGGER_REFERENCE"},
		{"tagReference", "TAG_REFERENCE"},
	}

	for _, tt := range tests {
		got := convertEnumValue(tt.input)
		if got != tt.expected {
			t.Errorf("convertEnumValue(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCamelToScreamingCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"requestPath", "REQUEST_PATH"},
		{"customEvent", "CUSTOM_EVENT"},
		{"youTubeVideo", "YOU_TUBE_VIDEO"},
		{"domReady", "DOM_READY"},
		{"serverPageLocationHostname", "SERVER_PAGE_LOCATION_HOSTNAME"},
	}

	for _, tt := range tests {
		got := camelToScreamingCase(tt.input)
		if got != tt.expected {
			t.Errorf("camelToScreamingCase(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestReverseEnumValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"REQUEST_PATH", "requestPath"},
		{"CUSTOM_EVENT", "customEvent"},
		{"DOM_READY", "domReady"},
		{"YOU_TUBE_VIDEO", "youTubeVideo"},
		{"ONCE_PER_EVENT", "oncePerEvent"},
		{"NOT_SET", "notSet"},
		{"TEMPLATE", "template"},
		{"BOOLEAN", "boolean"},
		{"PAGEVIEW", "pageview"},
		{"CLICK", "click"},
		{"SERVER", "server"},
		{"WEB", "web"},
		{"requestPath", "requestPath"},
		{"gaawc", "gaawc"},
		{"cvt_198845464_347", "cvt_198845464_347"},
	}

	for _, tt := range tests {
		got := reverseEnumValue(tt.input)
		if got != tt.expected {
			t.Errorf("reverseEnumValue(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestScreamingToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"REQUEST_PATH", "requestPath"},
		{"CUSTOM_EVENT", "customEvent"},
		{"YOU_TUBE_VIDEO", "youTubeVideo"},
		{"SERVER_PAGE_LOCATION_HOSTNAME", "serverPageLocationHostname"},
		{"FIREBASE_APP_EXCEPTION", "firebaseAppException"},
	}

	for _, tt := range tests {
		got := screamingToCamelCase(tt.input)
		if got != tt.expected {
			t.Errorf("screamingToCamelCase(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestRoundtripConversion(t *testing.T) {
	camelValues := []string{
		"requestPath", "customEvent", "domReady", "youTubeVideo",
		"scrollDepthThreshold", "elementVisibilityFirstTime",
		"firebaseAppException", "serverPageview", "consentInit",
		"oncePerEvent", "notNeeded", "startsWith", "matchRegex",
	}

	for _, camel := range camelValues {
		screaming := convertEnumValue(camel)
		back := reverseEnumValue(screaming)
		if back != camel {
			t.Errorf("roundtrip failed: %q → %q → %q (expected %q)", camel, screaming, back, camel)
		}
	}

	screamingValues := []string{
		"REQUEST_PATH", "CUSTOM_EVENT", "DOM_READY", "YOU_TUBE_VIDEO",
		"SCROLL_DEPTH_THRESHOLD", "ONCE_PER_EVENT", "NOT_SET",
	}

	for _, screaming := range screamingValues {
		camel := reverseEnumValue(screaming)
		back := convertEnumValue(camel)
		if back != screaming {
			t.Errorf("reverse roundtrip failed: %q → %q → %q (expected %q)", screaming, camel, back, screaming)
		}
	}
}
