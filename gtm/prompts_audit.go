package gtm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

func handleAuditContainerPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	tags, err := client.ListTags(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	triggers, err := client.ListTriggers(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list triggers: %w", err)
	}

	variables, err := client.ListVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	workspaceData := map[string]any{
		"tags":      tags,
		"triggers":  triggers,
		"variables": variables,
		"summary": map[string]int{
			"totalTags":      len(tags),
			"totalTriggers":  len(triggers),
			"totalVariables": len(variables),
		},
	}

	dataJSON, err := json.MarshalIndent(workspaceData, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Container audit analysis request",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Please audit this GTM workspace for potential issues. Here is the current configuration:

%s

Please analyze and report on:

1. **Naming Consistency**
   - Are tag, trigger, and variable names following a consistent pattern?
   - Are there any names that are unclear or non-descriptive?

2. **Duplicate Detection**
   - Are there any tags that appear to be duplicates (same type and similar configuration)?
   - Are there triggers that fire on the same conditions?

3. **Orphaned Items**
   - Are there any triggers that are not used by any tags?
   - Are there any variables that don't appear to be referenced?

4. **Best Practices**
   - Are tags properly organized with appropriate triggers?
   - Are there any paused tags that might be forgotten?
   - Are there missing triggers for common use cases?

5. **GA4 Configuration** (if applicable)
   - Is there a GA4 configuration tag?
   - Are event tags properly linked to the configuration?
   - Are ecommerce events configured correctly?

6. **Security Concerns**
   - Are there any custom HTML tags that might pose security risks?
   - Are there any tags loading external scripts?

Please provide specific recommendations for improvements.`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleAuditConsentPrivacyPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]
	framework := req.Params.Arguments["framework"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	if framework == "" {
		framework = "unknown"
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	tags, err := client.ListTags(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	triggers, err := client.ListTriggers(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list triggers: %w", err)
	}

	variables, err := client.ListVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	builtins, err := client.ListBuiltInVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list built-in variables: %w", err)
	}

	data := map[string]any{
		"tags":             tags,
		"triggers":         triggers,
		"variables":        variables,
		"builtInVariables": builtins,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "GDPR/ePrivacy compliance audit",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Audit this GTM workspace for GDPR/ePrivacy compliance.

**Consent framework in use:** %s

## Workspace Configuration
%s

Please analyze for privacy compliance:

1. **Consent Mode Implementation**
   - Is Google Consent Mode v2 configured?
   - Are there consent initialization tags (gtag consent default/update)?
   - Which tags fire BEFORE consent is granted?
   - Which tags correctly wait for consent?

2. **Tag-by-Tag Consent Audit**
   For each tag, determine:
   | Tag Name | Type | Fires Before Consent? | Consent Check Present? | Risk Level |
   |----------|------|----------------------|----------------------|------------|

3. **PII Exposure Risk**
   - Custom HTML tags sending data to external endpoints?
   - Variables capturing email, phone, or user IDs?
   - URL parameters being captured that might contain PII?
   - Built-in variables that could expose user data?

4. **Cookie Compliance**
   - Which tags set cookies?
   - Are cookie durations aligned with regulations (max 13 months)?
   - Are there third-party cookies being set?

5. **Data Destinations**
   - List all external domains receiving data from tags
   - Are there data transfers outside the EU/EEA?

6. **Remediation Plan**
   - Priority list of issues to fix
   - Specific implementation steps for each fix
   - Recommended consent mode configuration`, framework, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleAuditServerSidePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	clients, err := client.ListClients(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	transformations, err := client.ListTransformations(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list transformations: %w", err)
	}

	tags, err := client.ListTags(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	triggers, err := client.ListTriggers(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list triggers: %w", err)
	}

	variables, err := client.ListVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	data := map[string]any{
		"clients":         clients,
		"transformations": transformations,
		"tags":            tags,
		"triggers":        triggers,
		"variables":       variables,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Server-side GTM container audit",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Audit this server-side GTM (sGTM) container configuration:

## Container Configuration
%s

Please analyze:

1. **Client Configuration**
   - Is there a GA4 client configured to receive events?
   - Are there additional clients (webhook, custom)?
   - Are clients filtering by request path/type correctly?

2. **Transformation Pipeline**
   - What transformations are applied to incoming events?
   - Are transformations correctly ordered?
   - Are there redundant transformations?

3. **Server-Side Tags**
   - Which tags fire server-side (GA4, Facebook CAPI, etc.)?
   - Are tags properly connected to client triggers?
   - Are there tags missing consent checks?

4. **Data Quality**
   - Are PII fields being redacted in transformations?
   - Are event parameters properly mapped between client and server?
   - Are there missing required parameters for destination APIs?

5. **Performance**
   - Tags that could cause latency issues?
   - Unnecessary duplicate API calls?

6. **Recommendations**
   - Missing sGTM best practices
   - Security improvements
   - Optimization opportunities`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleAuditCustomTemplatesPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	templates := make([]TemplateInfo, 0)
	parent := BuildWorkspacePath(accountID, containerID, workspaceID)
	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListTemplatesResponse, error) {
		return client.Service.Accounts.Containers.Workspaces.Templates.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	if resp.Template != nil {
		for _, t := range resp.Template {
			info := TemplateInfo{
				TemplateID:    t.TemplateId,
				Name:          t.Name,
				TagManagerUrl: t.TagManagerUrl,
			}
			if t.GalleryReference != nil && t.GalleryReference.GalleryTemplateId != "" {
				info.Type = fmt.Sprintf("cvt_%s", t.GalleryReference.GalleryTemplateId)
				info.GalleryReference = &GalleryReferenceInfo{
					Owner:             t.GalleryReference.Owner,
					Repository:        t.GalleryReference.Repository,
					Version:           t.GalleryReference.Version,
					GalleryTemplateId: t.GalleryReference.GalleryTemplateId,
				}
			} else {
				info.Type = fmt.Sprintf("cvt_%s_%s", containerID, t.TemplateId)
			}
			templates = append(templates, info)
		}
	}

	tags, err := client.ListTags(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	data := map[string]any{
		"templates": templates,
		"tags":      tags,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Custom template security and quality audit",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Audit the custom templates in this GTM workspace for security and quality.

## Templates and Tags
%s

Please analyze:

1. **Template Inventory**
   | Template Name | Type | Used By (tags) | Has Permissions | Gallery Source |
   |---------------|------|----------------|-----------------|---------------|

2. **Security Review**
   - Templates with overly broad permissions (e.g., access to all cookies)?
   - Templates injecting scripts from external domains?
   - Templates using injectScript without domain restrictions?
   - Templates with access to sensitive APIs (sendPixel, setCookie)?

3. **Code Quality**
   - Templates using sandboxed APIs correctly?
   - Deprecated API calls?
   - Adequate error handling?

4. **Adoption Analysis**
   - Templates not used by any tag (candidates for removal)
   - Custom templates that could be replaced by Gallery alternatives

5. **Permission Matrix**
   | Template | Permissions Declared | Overly Broad? | Recommendation |
   |----------|---------------------|---------------|----------------|

6. **Recommendations**
   - Templates to remove (unused)
   - Templates to replace (Gallery alternatives exist)
   - Security fixes needed (priority ordered)`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleValidateVariableReferencesPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	tags, err := client.ListTags(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	triggers, err := client.ListTriggers(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list triggers: %w", err)
	}

	variables, err := client.ListVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	builtins, err := client.ListBuiltInVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list built-in variables: %w", err)
	}

	data := map[string]any{
		"tags":             tags,
		"triggers":         triggers,
		"variables":        variables,
		"builtInVariables": builtins,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Variable reference validation",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Audit all variable references in this GTM workspace for integrity issues.

## Workspace Configuration
%s

Analyze the following:

1. **Broken References**
   - Variables referenced in tags/triggers (via {{var_name}}) that don't exist in the workspace
   - Built-in variables used but NOT enabled
   - List each broken reference with the entity that uses it

2. **Orphaned Variables**
   - Variables defined but not referenced by any tag, trigger, or other variable
   - Candidates for cleanup — verify before deleting

3. **Circular Dependencies**
   - Variables that reference each other in a cycle
   - Detect chains: A → B → C → A

4. **Variable Dependency Graph**
   For each variable, show:
   - What data source it reads (Data Layer key, DOM, cookie, etc.)
   - Which tags/triggers consume it
   - Chain dependencies (variables referencing other variables)

5. **Impact Assessment**
   | Variable | Referenced By | If Deleted, Breaks |
   |----------|--------------|-------------------|

6. **Recommendations**
   - Variables safe to delete (orphaned, no consumers)
   - Broken references to fix immediately
   - Built-in variables to enable`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleDetectDuplicateTagsPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	tags, err := client.ListTags(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	triggers, err := client.ListTriggers(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list triggers: %w", err)
	}

	triggerMap := make(map[string]string)
	for _, t := range triggers {
		triggerMap[t.TriggerID] = t.Name
	}

	data := map[string]any{
		"tags":       tags,
		"triggerMap": triggerMap,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Duplicate tag detection",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Detect duplicate and redundant tags in this GTM workspace.

## Tags and Trigger Map
%s

Analyze for duplicates:

1. **Exact Duplicates**
   - Tags with identical type + identical parameters (different names)

2. **Near Duplicates** (same type + overlapping triggers)
   | Tag A | Tag B | Type | Shared Triggers | Different Params | Similarity |
   |-------|-------|------|----------------|-----------------|------------|

3. **Paused Duplicates**
   - Paused tags that duplicate the logic of an active tag

4. **Multi-Fire Risk**
   - Tags with the same tracking/measurement ID firing on overlapping triggers
   - Impact: double-counting pageviews, events, or conversions

5. **Consolidation Plan**
   For each duplicate cluster:
   - Which tag to keep (most complete/recent) 
   - Which to delete
   - Any parameters to merge before deletion

6. **Estimated Impact**
   - Number of duplicate events per page load
   - Potential cost impact (e.g., inflated GA4/BigQuery hits, Floodlight charges)`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleOptimizeBuiltInVariablesPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	builtins, err := client.ListBuiltInVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list built-in variables: %w", err)
	}

	tags, err := client.ListTags(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	triggers, err := client.ListTriggers(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list triggers: %w", err)
	}

	variables, err := client.ListVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	data := map[string]any{
		"builtInVariables": builtins,
		"tags":             tags,
		"triggers":         triggers,
		"customVariables":  variables,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Built-in variable optimization",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Optimize built-in variable usage in this GTM workspace.

## Workspace Configuration
%s

Please analyze:

1. **Usage Audit**
   | Built-in Variable | Enabled | Used in Tags | Used in Triggers | Used in Vars | Verdict |
   |-------------------|---------|-------------|-----------------|-------------|---------|

2. **Unnecessary Built-ins**
   - Enabled built-ins not referenced anywhere
   - Recommendation: disable to reduce container size

3. **Missing Built-ins**
   - Based on tags/triggers, which built-ins SHOULD be enabled?
   - Example: scroll triggers without scroll depth built-ins

4. **Redundant Custom Variables**
   - Custom variables that duplicate built-in functionality
   - Replace with built-ins where possible (smaller container, better performance)

5. **Actions**
   For each recommendation, provide the specific enable/disable command:
   - enable_built_in_variables with types: [list]
   - disable_built_in_variables with types: [list]`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleAuditSGTMDataFlowPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	clients, err := client.ListClients(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	transformations, err := client.ListTransformations(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list transformations: %w", err)
	}

	tags, err := client.ListTags(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	triggers, err := client.ListTriggers(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list triggers: %w", err)
	}

	variables, err := client.ListVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	data := map[string]any{
		"clients":         clients,
		"transformations": transformations,
		"tags":            tags,
		"triggers":        triggers,
		"variables":       variables,
		"summary": map[string]int{
			"totalClients":         len(clients),
			"totalTransformations": len(transformations),
			"totalTags":            len(tags),
			"totalTriggers":        len(triggers),
			"totalVariables":       len(variables),
		},
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "sGTM data flow pipeline analysis",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Trace the complete data flow pipeline in this server-side GTM container.

## sGTM Container Configuration
%s

Analyze the full data pipeline from incoming requests to outbound API calls:

1. **Client → Event Data Mapping**
   For each client:
   | Client Name | Type | Priority | Claims Path | Events Generated |
   |-------------|------|----------|-------------|------------------|
   - Are there incoming request types with NO client to claim them?
   - Are clients generating the expected event data objects?

2. **Event Data → Transformation Pipeline**
   For each transformation:
   | Transformation | Type | Fields Modified | Fields Removed | Condition |
   |---------------|------|----------------|----------------|-----------|
   - Are transformations removing fields that downstream tags NEED?
   - Are transformations ordered correctly (PII redaction before enrichment)?

3. **Trigger → Tag Firing**
   For each tag:
   | Tag Name | Type | Trigger | Required Fields | Fields Available Post-Transform |
   |----------|------|---------|----------------|-------------------------------|
   - Are there tags whose trigger conditions NEVER match any client's events?
   - Are there tags missing required parameters after transformations remove them?

4. **Broken Pipeline Detection**
   Identify specific breaks in the chain:
   - Clients generating events that no trigger matches
   - Triggers referencing client names that don't exist
   - Tags expecting event data fields that transformations delete
   - Variables referencing event data keys not set by any client

5. **Flow Diagram**
   Generate a text-based flow diagram:
   [Client A] → [Event: page_view] → [Transform: redact PII] → [Trigger: All Events] → [GA4 Tag]
   [Client A] → [Event: purchase] → [Transform: enrich] → [Trigger: Purchase] → [CAPI Tag]

6. **Recommendations**
   - Gaps where data flows into nothing (lost events)
   - Redundant paths (same event processed twice)
   - Missing transformations in critical paths`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleAuditSGTMPIIExposurePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	transformations, err := client.ListTransformations(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list transformations: %w", err)
	}

	tags, err := client.ListTags(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	variables, err := client.ListVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	clients, err := client.ListClients(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	data := map[string]any{
		"transformations": transformations,
		"tags":            tags,
		"variables":       variables,
		"clients":         clients,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "sGTM PII exposure audit",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Audit this server-side GTM container for PII (Personally Identifiable Information) exposure.

The key advantage of sGTM is that YOU control what data leaves your server. This audit verifies that control is properly implemented.

## sGTM Container Configuration
%s

Analyze for PII exposure risks:

1. **PII Fields in Event Data**
   Identify event data fields likely to contain PII:
   | Field | Type | Source | PII Risk | Redacted by Transform? |
   |-------|------|--------|----------|----------------------|
   Common PII fields: email, phone, ip_address, user_id, first_name, last_name, address, user_data.*

2. **Transformation Coverage**
   - Which transformations remove or hash PII fields?
   - Are PII transformations applied BEFORE all outbound tags?
   - Are there outbound tags that fire WITHOUT going through PII transformations?

3. **Tag-by-Tag PII Exposure**
   For each outbound tag (GA4, CAPI, HTTP Request, etc.):
   | Tag Name | Destination | PII Fields Sent | Hashed? | Transform Applied? | Risk |
   |----------|-------------|----------------|---------|-------------------|------|

4. **GA4 Tag Specific**
   - Is redactVisitorIp enabled?
   - Is removeAdsDataRedaction configured?
   - Are user_data fields being forwarded unnecessarily?

5. **HTTP Request Tags**
   - What PII appears in request URLs, headers, or body?
   - Are API keys/tokens hardcoded in parameters?

6. **Cookie Handling**
   - Are clients writing cookies with PII?
   - Cookie scope and duration for cookies containing user identifiers?

7. **GDPR Compliance Matrix**
   | Data Category | Collected | Purpose | Legal Basis | Retention | Risk Level |
   |---------------|-----------|---------|-------------|-----------|------------|

8. **Remediation Plan**
   Priority-ordered list of:
   - Transformations to add (hash or remove PII)
   - Tag parameters to modify
   - Variables to redact
   Provide specific implementation steps for each fix.`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleAuditSGTMClientPriorityPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	clients, err := client.ListClients(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}

	tags, err := client.ListTags(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	triggers, err := client.ListTriggers(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list triggers: %w", err)
	}

	data := map[string]any{
		"clients":  clients,
		"tags":     tags,
		"triggers": triggers,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "sGTM client priority and claim analysis",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Analyze the client priority configuration in this server-side GTM container.

In sGTM, clients process incoming requests in PRIORITY ORDER (highest priority first). The first client to "claim" a request prevents all subsequent clients from processing it. Misconfigured priorities cause silent data loss.

## sGTM Container Configuration
%s

Analyze:

1. **Client Priority Order**
   | Priority | Client Name | Type | Claims Pattern | Status |
   |----------|-------------|------|---------------|--------|
   (Sorted by priority, highest first)

2. **Priority Conflicts**
   - Clients with IDENTICAL priority values (undefined claim order)
   - Clients of the same type competing for the same requests
   - GA4 client priority relative to custom/webhook clients

3. **Claim Coverage Analysis**
   - Which incoming request paths/types are claimed by which client?
   - Are there request paths that NO client claims (lost data)?
   - Are there request paths claimed by multiple clients (only first wins)?

4. **Client-to-Tag Alignment**
   For each client, which tags consume its events?
   | Client | Events Generated | Tags That Fire | Tags Missing |
   |--------|-----------------|----------------|-------------|
   - Are there clients whose events don't trigger ANY tags?
   - Are there tags expecting events from a client type that doesn't exist?

5. **Inactive Clients**
   - Clients with no tags downstream (processing requests for nothing)
   - Clients that could be disabled to reduce processing overhead

6. **Best Practice Recommendations**
   - GA4 client should have highest priority (lowest number)
   - HTTP request/webhook clients should have lower priority
   - Custom clients should have explicit priority (never default)
   - Unused clients should be removed to avoid accidental claims`, string(dataJSON)),
				},
			},
		},
	}, nil
}
