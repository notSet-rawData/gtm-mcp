package gtm

// TagTemplate provides example parameter structures for creating tags.
type TagTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Parameters  string `json:"parameters"`
	Notes       string `json:"notes"`
}

// GetTagTemplates returns example parameter structures for common tag types.
// These templates help LLMs create tags with the correct GTM API parameter format.
func GetTagTemplates() []TagTemplate {
	return []TagTemplate{
		{
			Name:        "GA4 Configuration",
			Description: "Google Analytics 4 configuration tag (fires on all pages)",
			Type:        "gaawc",
			Parameters: `[
  {"type": "template", "key": "measurementId", "value": "G-XXXXXXXXXX"}
]`,
			Notes: "Use gaawc type for GA4 Config tags. The measurementId should be your GA4 Measurement ID.",
		},
		{
			Name:        "GA4 Event (Simple)",
			Description: "Google Analytics 4 event tag with custom event name",
			Type:        "gaawe",
			Parameters: `[
  {"type": "tagReference", "key": "measurementId", "value": ""},
  {"type": "template", "key": "measurementIdOverride", "value": "{{GA4 Measurement ID}}"},
  {"type": "template", "key": "eventName", "value": "custom_event_name"}
]`,
			Notes: "Use gaawe type for GA4 Event tags. measurementId must be empty tagReference, use measurementIdOverride for the actual value (variable reference or literal).",
		},
		{
			Name:        "GA4 Event with Parameters",
			Description: "Google Analytics 4 event tag with custom parameters",
			Type:        "gaawe",
			Parameters: `[
  {"type": "tagReference", "key": "measurementId", "value": ""},
  {"type": "template", "key": "measurementIdOverride", "value": "{{GA4 Measurement ID}}"},
  {"type": "template", "key": "eventName", "value": "button_click"},
  {"type": "list", "key": "eventParameters", "list": [
    {"type": "map", "map": [
      {"type": "template", "key": "name", "value": "button_id"},
      {"type": "template", "key": "value", "value": "{{Click ID}}"}
    ]},
    {"type": "map", "map": [
      {"type": "template", "key": "name", "value": "button_text"},
      {"type": "template", "key": "value", "value": "{{Click Text}}"}
    ]}
  ]}
]`,
			Notes: "Event parameters use name/value pairs inside map structures. Do NOT use the parameter name as the key directly.",
		},
		{
			Name:        "GA4 Ecommerce Purchase",
			Description: "Google Analytics 4 ecommerce purchase event (reads items from dataLayer)",
			Type:        "gaawe",
			Parameters: `[
  {"type": "tagReference", "key": "measurementId", "value": ""},
  {"type": "template", "key": "measurementIdOverride", "value": "{{GA4 Measurement ID}}"},
  {"type": "template", "key": "eventName", "value": "purchase"},
  {"type": "boolean", "key": "sendEcommerceData", "value": "true"},
  {"type": "template", "key": "getEcommerceDataFrom", "value": "dataLayer"},
  {"type": "list", "key": "eventParameters", "list": [
    {"type": "map", "map": [
      {"type": "template", "key": "name", "value": "transaction_id"},
      {"type": "template", "key": "value", "value": "{{DL - Transaction ID}}"}
    ]}
  ]}
]`,
			Notes: "For ecommerce events, set sendEcommerceData=true and getEcommerceDataFrom=dataLayer. The items array will be read automatically from the dataLayer ecommerce object.",
		},
		{
			Name:        "GA4 Ecommerce Add to Cart",
			Description: "Google Analytics 4 ecommerce add_to_cart event",
			Type:        "gaawe",
			Parameters: `[
  {"type": "tagReference", "key": "measurementId", "value": ""},
  {"type": "template", "key": "measurementIdOverride", "value": "{{GA4 Measurement ID}}"},
  {"type": "template", "key": "eventName", "value": "add_to_cart"},
  {"type": "boolean", "key": "sendEcommerceData", "value": "true"},
  {"type": "template", "key": "getEcommerceDataFrom", "value": "dataLayer"}
]`,
			Notes: "Similar to purchase, but for add_to_cart event. Items are read from dataLayer.",
		},
		{
			Name:        "GA4 Ecommerce View Item",
			Description: "Google Analytics 4 ecommerce view_item event",
			Type:        "gaawe",
			Parameters: `[
  {"type": "tagReference", "key": "measurementId", "value": ""},
  {"type": "template", "key": "measurementIdOverride", "value": "{{GA4 Measurement ID}}"},
  {"type": "template", "key": "eventName", "value": "view_item"},
  {"type": "boolean", "key": "sendEcommerceData", "value": "true"},
  {"type": "template", "key": "getEcommerceDataFrom", "value": "dataLayer"}
]`,
			Notes: "For product detail page views. Items are read from dataLayer.",
		},
		{
			Name:        "Custom HTML",
			Description: "Custom HTML tag for arbitrary JavaScript",
			Type:        "html",
			Parameters: `[
  {"type": "template", "key": "html", "value": "<script>\n  console.log('Hello from GTM!');\n</script>"}
]`,
			Notes: "Use html type for custom JavaScript. The html parameter contains the script.",
		},
		{
			Name:        "Custom Image (Pixel)",
			Description: "Custom image tag for tracking pixels",
			Type:        "img",
			Parameters: `[
  {"type": "template", "key": "url", "value": "https://example.com/pixel.gif?event=pageview"},
  {"type": "boolean", "key": "useCacheBuster", "value": "true"},
  {"type": "template", "key": "cacheBusterQueryParam", "value": "gtmcb"}
]`,
			Notes: "Use img type for tracking pixels. Enable cacheBuster to prevent caching.",
		},
	}
}

// TriggerTemplate provides example structures for creating triggers.
type TriggerTemplate struct {
	Name                  string `json:"name"`
	Description           string `json:"description"`
	Type                  string `json:"type"`
	FilterJSON            string `json:"filterJson,omitempty"`
	AutoEventFilterJSON   string `json:"autoEventFilterJson,omitempty"`
	CustomEventFilterJSON string `json:"customEventFilterJson,omitempty"`
	Notes                 string `json:"notes"`
}

// GetTriggerTemplates returns example structures for common trigger types.
func GetTriggerTemplates() []TriggerTemplate {
	return []TriggerTemplate{
		{
			Name:        "All Pages",
			Description: "Fires on every page view",
			Type:        "pageview",
			Notes:       "Simple pageview trigger with no filters.",
		},
		{
			Name:        "Specific Page",
			Description: "Fires on a specific page URL",
			Type:        "pageview",
			FilterJSON: `[
  {"type": "contains", "parameter": [
    {"type": "template", "key": "arg0", "value": "{{Page URL}}"},
    {"type": "template", "key": "arg1", "value": "/checkout"}
  ]}
]`,
			Notes: "Use filterJson to match specific pages. arg0 is the variable, arg1 is the value to match.",
		},
		{
			Name:        "Custom Event",
			Description: "Fires on a dataLayer custom event",
			Type:        "customEvent",
			CustomEventFilterJSON: `[
  {"type": "equals", "parameter": [
    {"type": "template", "key": "arg0", "value": "{{_event}}"},
    {"type": "template", "key": "arg1", "value": "purchase"}
  ]}
]`,
			Notes: "For customEvent triggers, use customEventFilterJson (not filterJson). The {{_event}} variable matches the dataLayer event name.",
		},
		{
			Name:        "Click - All Elements",
			Description: "Fires on all element clicks",
			Type:        "linkClick",
			AutoEventFilterJSON: `[
  {"type": "contains", "parameter": [
    {"type": "template", "key": "arg0", "value": "{{Click Classes}}"},
    {"type": "template", "key": "arg1", "value": "cta-button"}
  ]}
]`,
			Notes: "Use linkClick for click triggers. Use autoEventFilterJson to filter by click element properties (Click Classes, Click ID, Click URL, etc.).",
		},
		{
			Name:        "Form Submission",
			Description: "Fires on form submissions",
			Type:        "formSubmission",
			AutoEventFilterJSON: `[
  {"type": "equals", "parameter": [
    {"type": "template", "key": "arg0", "value": "{{Form ID}}"},
    {"type": "template", "key": "arg1", "value": "contact-form"}
  ]}
]`,
			Notes: "Use formSubmission type. Use autoEventFilterJson to filter by form properties (Form ID, Form Classes, Form URL, etc.).",
		},
	}
}

// ClientTemplate provides example parameter structures for creating sGTM clients.
type ClientTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Priority    int64  `json:"priority"`
	Parameters  string `json:"parameters"`
	Notes       string `json:"notes"`
}

// GetClientTemplates returns example parameter structures for common server-side GTM client types.
// These templates help LLMs create sGTM clients with the correct parameter format.
func GetClientTemplates() []ClientTemplate {
	return []ClientTemplate{
		{
			Name:        "GA4 Client",
			Description: "Google Analytics 4 client — claims and processes GA4 /collect and /g/collect requests",
			Type:        "gaaw_client",
			Priority:    10,
			Parameters: `[
  {"type": "boolean", "key": "activateGtagOnPage", "value": "false"},
  {"type": "boolean", "key": "enableCookieOverrides", "value": "false"}
]`,
			Notes: "The GA4 client (gaaw_client) automatically claims GA4 requests. Priority determines which client gets first claim when multiple clients match. activateGtagOnPage serves the GA4 JS library from your server domain (first-party). enableCookieOverrides enables server-managed cookies.",
		},
		{
			Name:        "GA4 Client with Server-Managed Cookies",
			Description: "GA4 client with first-party cookie management for improved tracking accuracy",
			Type:        "gaaw_client",
			Priority:    10,
			Parameters: `[
  {"type": "boolean", "key": "activateGtagOnPage", "value": "true"},
  {"type": "boolean", "key": "enableCookieOverrides", "value": "true"},
  {"type": "template", "key": "cookiePrefix", "value": "_ga_sst"},
  {"type": "boolean", "key": "enableRegionSpecificSettings", "value": "false"},
  {"type": "boolean", "key": "enableJsLibrary", "value": "true"}
]`,
			Notes: "Server-managed cookies (enableCookieOverrides) set first-party cookies from your server domain, improving tracking in browsers with ITP restrictions (Safari, Firefox). cookiePrefix avoids collision with client-side cookies. enableJsLibrary serves gtag.js from your domain.",
		},
		{
			Name:        "HTTP Request Client (Webhook Receiver)",
			Description: "Generic HTTP client that claims requests matching a specific path — webhooks, Measurement Protocol, custom APIs",
			Type:        "http_request",
			Priority:    20,
			Parameters: `[
  {"type": "template", "key": "requestMethod", "value": "POST"},
  {"type": "template", "key": "requestPath", "value": "/webhook"},
  {"type": "boolean", "key": "defaultResponse", "value": "true"}
]`,
			Notes: "http_request clients claim incoming HTTP requests matching specific paths. Use for webhooks, Measurement Protocol forwarding, or custom API endpoints. Higher priority number = lower priority (claimed after other clients). defaultResponse sends a 200 OK automatically.",
		},
		{
			Name:        "Measurement Protocol Client",
			Description: "Client for receiving GA4 Measurement Protocol hits (server-to-server events)",
			Type:        "http_request",
			Priority:    15,
			Parameters: `[
  {"type": "template", "key": "requestMethod", "value": "POST"},
  {"type": "template", "key": "requestPath", "value": "/mp/collect"},
  {"type": "boolean", "key": "defaultResponse", "value": "true"}
]`,
			Notes: "Receives Measurement Protocol v2 requests at /mp/collect. Typically used for offline conversions, CRM events, or server-to-server data. Pair with a GA4 tag to forward events to your property.",
		},
		// NOTE: Facebook CAPI, TikTok Events API, etc. are sGTM TAGS (they send data out),
		// not clients (which receive/claim incoming requests). Those belong in GetServerSideTagTemplates().
	}
}

// ServerSideTagTemplate provides example parameter structures for sGTM tags.
type ServerSideTagTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Parameters  string `json:"parameters"`
	Notes       string `json:"notes"`
}

// GetServerSideTagTemplates returns example parameter structures for server-side GTM tag types.
// These are separate from web tag templates because sGTM tags use different type identifiers.
// Source: datalayer-server-side-gtm notebook (built-in tags) + marketing-meta notebook (CAPI params).
func GetServerSideTagTemplates() []ServerSideTagTemplate {
	return []ServerSideTagTemplate{
		{
			Name:        "GA4 Server-Side Tag",
			Description: "Forwards event data from GA4 client to Google Analytics 4 property — the most common sGTM tag",
			Type:        "sgtmgaaw",
			Parameters: `[
  {"type": "boolean", "key": "inheritMeasurementId", "value": "true"},
  {"type": "template", "key": "measurementIdOverride", "value": ""},
  {"type": "boolean", "key": "inheritEventName", "value": "true"},
  {"type": "template", "key": "eventNameOverride", "value": ""},
  {"type": "boolean", "key": "redactVisitorIp", "value": "false"},
  {"type": "boolean", "key": "removeAdsDataRedaction", "value": "false"}
]`,
			Notes: "Uses type 'sgtmgaaw'. When inheritMeasurementId is true, reads the Measurement ID from the incoming GA4 client event. Set measurementIdOverride to route data to a different GA4 property. inheritEventName allows forwarding the original event name or overriding it. redactVisitorIp removes the user's IP before sending to GA4.",
		},
		{
			Name:        "HTTP Request Tag",
			Description: "Sends HTTP requests to any endpoint — use for webhooks, custom APIs, or forwarding data to non-Google destinations",
			Type:        "sgtmhttp",
			Parameters: `[
  {"type": "template", "key": "requestUrl", "value": "https://api.example.com/events"},
  {"type": "template", "key": "requestMethod", "value": "POST"},
  {"type": "template", "key": "requestBody", "value": "{\"event\": \"{{Event Name}}\", \"timestamp\": \"{{Event Timestamp}}\"}"},
  {"type": "list", "key": "requestHeaders", "list": [
    {"type": "map", "map": [
      {"type": "template", "key": "headerName", "value": "Content-Type"},
      {"type": "template", "key": "headerValue", "value": "application/json"}
    ]},
    {"type": "map", "map": [
      {"type": "template", "key": "headerName", "value": "Authorization"},
      {"type": "template", "key": "headerValue", "value": "Bearer {{API Key}}"}
    ]}
  ]}
]`,
			Notes: "Uses type 'sgtmhttp'. Sends arbitrary HTTP requests from the server. Reference sGTM variables (Event Name, Event Timestamp, etc.) in the request body. Supports custom headers for authentication. Use this for integrations without a dedicated Gallery template.",
		},
		{
			Name:        "Conversion Linker",
			Description: "Reads Google click IDs (GCLID, DCLID) from incoming requests and stores them in first-party cookies for conversion attribution",
			Type:        "gclidw",
			Parameters: `[
  {"type": "boolean", "key": "enableCrossDomain", "value": "false"},
  {"type": "boolean", "key": "enableUrlPassthrough", "value": "false"}
]`,
			Notes: "Uses type 'gclidw'. Essential for Google Ads and Floodlight conversion tracking in sGTM. Reads GCLID/DCLID from the incoming request, stores in cookies (_gcl_aw, _gcl_dc). Should fire on All Pages trigger. enableCrossDomain is needed if tracking spans multiple domains.",
		},
		{
			Name:        "Meta Conversions API (CAPI) — Gallery Template",
			Description: "Sends server-side events to Meta/Facebook Conversions API for ad attribution and optimization",
			Type:        "cvt_CONTAINER_TEMPLATE_ID",
			Parameters: `[
  {"type": "template", "key": "pixelId", "value": "YOUR_PIXEL_ID"},
  {"type": "template", "key": "accessToken", "value": "YOUR_ACCESS_TOKEN"},
  {"type": "template", "key": "actionSource", "value": "website"},
  {"type": "template", "key": "eventId", "value": "{{Event ID}}"},
  {"type": "boolean", "key": "inheritEventName", "value": "true"}
]`,
			Notes: "Gallery template — import first via import_gallery_template. Type will be 'cvt_{containerId}_{templateId}'. Required by Meta: pixelId, accessToken, actionSource, event_source_url (auto-read from GA4 client), client_user_agent (auto-read). For deduplication with browser pixel, eventId must match the browser's eventID. Send user data (em, ph, fbp, fbc) for better Event Match Quality. Parameters em/ph/fn/ln require SHA256 hashing.",
		},
	}
}

// TransformationTemplate provides example parameter structures for sGTM transformations.
type TransformationTemplate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Parameters  string `json:"parameters"`
	Notes       string `json:"notes"`
}

// GetTransformationTemplates returns example structures for common sGTM transformation types.
func GetTransformationTemplates() []TransformationTemplate {
	return []TransformationTemplate{
		{
			Name:        "Redact PII from Event Data",
			Description: "Remove or hash personally identifiable information before forwarding",
			Type:        "gtes",
			Parameters: `[
  {"type": "list", "key": "rules", "list": [
    {"type": "map", "map": [
      {"type": "template", "key": "eventDataKey", "value": "user_data.email_address"},
      {"type": "template", "key": "action", "value": "remove"}
    ]},
    {"type": "map", "map": [
      {"type": "template", "key": "eventDataKey", "value": "user_data.phone_number"},
      {"type": "template", "key": "action", "value": "remove"}
    ]}
  ]}
]`,
			Notes: "Use transformation type 'gtes' (Google Tag Event Settings) to modify event data keys. Actions: 'remove' deletes the key, 'set' overrides it, 'hash' applies SHA256 hashing. Apply to sensitive fields before data leaves your server.",
		},
		{
			Name:        "Add Server-Side Parameters",
			Description: "Enrich events with server-side data (timestamps, internal IDs)",
			Type:        "gtes",
			Parameters: `[
  {"type": "list", "key": "rules", "list": [
    {"type": "map", "map": [
      {"type": "template", "key": "eventDataKey", "value": "server_timestamp"},
      {"type": "template", "key": "action", "value": "set"},
      {"type": "template", "key": "value", "value": "{{Server Timestamp}}"}
    ]},
    {"type": "map", "map": [
      {"type": "template", "key": "eventDataKey", "value": "server_processed"},
      {"type": "template", "key": "action", "value": "set"},
      {"type": "template", "key": "value", "value": "true"}
    ]}
  ]}
]`,
			Notes: "Use 'set' action to add or override event data keys. Reference server-side variables with {{Variable Name}} syntax. Useful for adding server timestamps, internal IDs, or enrichment data from APIs.",
		},
		{
			Name:        "Filter by Event Name",
			Description: "Apply transformation only to specific events",
			Type:        "gtes",
			Parameters: `[
  {"type": "list", "key": "rules", "list": [
    {"type": "map", "map": [
      {"type": "template", "key": "eventDataKey", "value": "ip_override"},
      {"type": "template", "key": "action", "value": "remove"}
    ]}
  ]},
  {"type": "template", "key": "triggerCondition", "value": "{{Event Name}} equals purchase"}
]`,
			Notes: "Combine rules with triggerCondition to selectively apply transformations. This is useful for redacting IP addresses from purchase events or adding parameters only to specific event types.",
		},
	}
}
