package gtm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// =============================================================================
// PLAN PROMPTS — "¿Qué hago / cómo organizo?"
// Handlers: generate_tracking_plan, suggest_ga4_setup, find_gallery_template,
//           review_before_publish, folder_organization_review,
//           migration_plan_ua_to_ga4, environment_promotion_checklist,
//           plan_sgtm_setup, plan_web_to_sgtm_migration,
//           plan_sgtm_consent_architecture
// =============================================================================

// handleGenerateTrackingPlanPrompt creates markdown tracking plan documentation.
func handleGenerateTrackingPlanPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
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

	triggerMap := make(map[string]string)
	for _, t := range triggers {
		triggerMap[t.TriggerID] = t.Name
	}

	workspaceData := map[string]any{
		"tags":       tags,
		"triggers":   triggers,
		"variables":  variables,
		"triggerMap": triggerMap,
	}

	dataJSON, err := json.MarshalIndent(workspaceData, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Generate tracking plan documentation",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Please generate a comprehensive Markdown tracking plan document from this GTM workspace configuration:

%s

Generate a document with the following structure:

# Tracking Plan

## Overview
- Summary of the tracking implementation
- Total counts (tags, triggers, variables)

## Events

For each tag, create a section:

### [Event Name]
- **Tag Name:** [name]
- **Tag Type:** [type]
- **Trigger(s):** [list of trigger names]
- **Description:** [inferred purpose]
- **Parameters:** [if applicable]

## Triggers

For each trigger:

### [Trigger Name]
- **Type:** [type]
- **Conditions:** [filter conditions if any]
- **Used by:** [list of tags using this trigger]

## Variables

For each variable:

### [Variable Name]
- **Type:** [type]
- **Purpose:** [inferred purpose]

## Data Layer Requirements

List all dataLayer events and variables that need to be pushed from the website.

## Implementation Notes

Any observations about the implementation, dependencies, or recommendations.

Format the output as clean, professional Markdown.`, string(dataJSON)),
				},
			},
		},
	}, nil
}

// handleSuggestGA4SetupPrompt recommends a GA4 tag structure based on goals.
func handleSuggestGA4SetupPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	goals := req.Params.Arguments["goals"]

	if goals == "" {
		return nil, fmt.Errorf("goals description is required")
	}

	tagTemplates := GetTagTemplates()
	triggerTemplates := GetTriggerTemplates()

	templatesData := map[string]any{
		"tagTemplates":     tagTemplates,
		"triggerTemplates": triggerTemplates,
	}

	templatesJSON, err := json.MarshalIndent(templatesData, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "GA4 setup recommendations",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`I need help setting up GA4 tracking in Google Tag Manager for the following goals:

**Tracking Goals:**
%s

Here are the available tag and trigger templates that can be used:

%s

Please provide:

1. **Recommended Tags**
   - List each tag needed with:
     - Tag name (following naming convention: "[Category] - [Action]")
     - Tag type
     - Configuration details
     - Which trigger to use

2. **Recommended Triggers**
   - List each trigger needed with:
     - Trigger name
     - Trigger type
     - Filter conditions (if any)

3. **Required Variables**
   - List any Data Layer variables needed
   - List any built-in variables to enable

4. **Data Layer Requirements**
   - Specify what dataLayer pushes the website needs to implement
   - Provide example code snippets for each event

5. **Implementation Order**
   - Step-by-step order to create the tags, triggers, and variables

6. **Testing Checklist**
   - Key scenarios to test
   - Expected GA4 events and parameters

Please be specific about the GTM configuration - use the exact parameter formats shown in the templates.`, goals, string(templatesJSON)),
				},
			},
		},
	}, nil
}

// handleFindGalleryTemplatePrompt guides LLM to discover Community Template Gallery templates.
func handleFindGalleryTemplatePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	templateName := req.Params.Arguments["templateName"]

	if templateName == "" {
		return nil, fmt.Errorf("templateName is required")
	}

	return &mcp.GetPromptResult{
		Description: "Find and import a Community Template Gallery template",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`I need to find and import the "%s" template from the GTM Community Template Gallery.

**How to find a Community Template:**

1. **Search the web** for: "%s GTM community template github"
   - Community templates are hosted on GitHub
   - Look for results from github.com

2. **Extract the repository info** from the GitHub URL:
   - URL format: github.com/{owner}/{repository}
   - Example: github.com/iubenda/gtm-cookie-solution
     - galleryOwner: "iubenda"
     - galleryRepository: "gtm-cookie-solution"

3. **Browse the Gallery directly** (optional):
   - Visit: https://tagmanager.google.com/gallery/#/?filter=%s
   - Click on the template to see details

**Common templates for reference:**

| Template | galleryOwner | galleryRepository |
|----------|--------------|-------------------|
| iubenda Cookie Solution | iubenda | gtm-cookie-solution |
| Cookiebot | nicktue-gtm-templates | cookiebot-gtm |
| Facebook Pixel | nicktue-gtm-templates | facebook-pixel |

**Once you have the owner and repository:**

Use the import_gallery_template tool:
- galleryOwner: [owner from GitHub]
- galleryRepository: [repository from GitHub]

The tool will return the template type (cvt_{containerId}_{templateId}) to use when creating tags.

Please search for the "%s" template and provide the galleryOwner and galleryRepository values.`, templateName, templateName, templateName, templateName),
				},
			},
		},
	}, nil
}

// handleReviewBeforePublishPrompt generates a pre-publish review checklist with risk assessment.
func handleReviewBeforePublishPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
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

	status, err := client.GetWorkspaceStatus(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace status: %w", err)
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
		"workspaceStatus": status,
		"tags":            tags,
		"triggers":        triggers,
		"variables":       variables,
		"summary": map[string]int{
			"totalTags":      len(tags),
			"totalTriggers":  len(triggers),
			"totalVariables": len(variables),
		},
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Pre-publish review with risk assessment",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Review the pending changes in this GTM workspace before publishing.

## Workspace Status & Configuration
%s

Please provide a pre-publish review covering:

1. **Change Summary**
   - Categorize workspace changes: new, modified, deleted entities
   - Total counts by entity type

2. **Risk Assessment** (rate each: 🟢 Low / 🟡 Medium / 🔴 High)
   - Custom HTML tags with inline scripts?
   - Tags without triggers assigned?
   - Triggers that could fire too broadly (e.g., All Pages without filters)?
   - Variables referencing non-existent data layer keys?

3. **Dependency Check**
   - Do all tags have required triggers?
   - Do triggers reference variables that exist?
   - Are there circular dependencies?

4. **Naming Compliance**
   - Do all new/modified items follow the existing naming convention?

5. **Publish Recommendation**
   - ✅ Safe to publish / ⚠️ Review needed / 🛑 Do not publish
   - Specific concerns if any

6. **Suggested Version Name and Notes**
   - Based on the changes, suggest a descriptive version name`, string(dataJSON)),
				},
			},
		},
	}, nil
}

// handleFolderOrganizationReviewPrompt analyzes folder structure and suggests improvements.
func handleFolderOrganizationReviewPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
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

	folders, err := client.ListFolders(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	folderEntities := make(map[string]any)
	for _, f := range folders {
		entities, err := client.GetFolderEntities(ctx, accountID, containerID, workspaceID, f.FolderID)
		if err == nil {
			folderEntities[f.Name] = entities
		}
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
		"folders":        folders,
		"folderEntities": folderEntities,
		"tags":           tags,
		"triggers":       triggers,
		"variables":      variables,
		"summary": map[string]int{
			"totalFolders":   len(folders),
			"totalTags":      len(tags),
			"totalTriggers":  len(triggers),
			"totalVariables": len(variables),
		},
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Folder organization review and suggestions",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Analyze the folder organization of this GTM workspace and suggest improvements.

## Workspace Structure
%s

Please analyze:

1. **Current Structure Assessment**
   - Items unorganized (no folder): count and percentage
   - Are items logically grouped?
   - Is the naming convention consistent?

2. **Suggested Folder Structure**
   | Folder Name | Purpose | Items to Include | Priority |
   |-------------|---------|-----------------|----------|

   Common patterns to consider:
   - By vendor: GA4, Facebook, LinkedIn, etc.
   - By function: Analytics, Marketing, Consent, Utility
   - By business area: Ecommerce, Lead Gen, Content

3. **Naming Convention**
   - Tags: [Vendor] - [Action] - [Detail]
   - Triggers: [Type] - [Event/Page]
   - Variables: [Type] - [Key]
   - List items that should be renamed

4. **Cleanup Recommendations**
   - Paused tags that could be deleted
   - Duplicate or redundant items
   - Items with unclear names

5. **Implementation Plan**
   Step-by-step reorganization with exact tool calls`, string(dataJSON)),
				},
			},
		},
	}, nil
}

// handleMigrationPlanUA2GA4Prompt generates a UA to GA4 migration plan.
func handleMigrationPlanUA2GA4Prompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]
	ga4MeasurementID := req.Params.Arguments["ga4MeasurementId"]

	if accountID == "" || containerID == "" || workspaceID == "" || ga4MeasurementID == "" {
		return nil, fmt.Errorf("accountId, containerId, workspaceId, and ga4MeasurementId are required")
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

	data := map[string]any{
		"tags":      tags,
		"triggers":  triggers,
		"variables": variables,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "UA to GA4 migration plan",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Analyze this GTM workspace and generate a migration plan from Universal Analytics to GA4.

**Target GA4 Measurement ID:** %s

## Current Workspace Configuration
%s

Please generate:

1. **UA Tag Inventory**
   | UA Tag Name | Track Type | Category/Action/Label | GA4 Equivalent Event | Effort |
   |-------------|-----------|----------------------|---------------------|--------|

2. **Migration Map**
   For each UA tag:
   - GA4 equivalent (pageview→page_view, event→custom_event, transaction→purchase)
   - Parameters to migrate (custom dimensions → event parameters)
   - New GA4 tag configuration needed

3. **Tags to Eliminate**
   - UA tags redundant with GA4 auto-collected events
   - UA tags with no GA4 equivalent (deprecated features)

4. **Variable Refactoring**
   - Variables referencing UA-specific fields needing updates
   - DataLayer schema changes required

5. **Implementation Order**
   Step-by-step with dependencies:
   1. Create GA4 Config tag
   2. Migrate pageview tags
   3. Migrate event tags
   4. Migrate ecommerce tags
   5. Validate and remove UA tags

6. **Validation Checklist**
   Per migrated tag:
   | Original UA Tag | New GA4 Tag | Test Scenario | Expected GA4 Event |
   |----------------|-------------|--------------|-------------------|

7. **Effort Estimation**
   - Low: direct mapping, auto-collected equivalent
   - Medium: parameter remapping needed
   - High: dataLayer schema change or custom implementation`, ga4MeasurementID, string(dataJSON)),
				},
			},
		},
	}, nil
}

// handleEnvironmentPromotionChecklistPrompt generates a promotion checklist for environment transitions.
func handleEnvironmentPromotionChecklistPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
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

	environments, err := client.ListEnvironments(ctx, accountID, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	status, err := client.GetWorkspaceStatus(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace status: %w", err)
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

	headers, err := client.ListVersionHeaders(ctx, accountID, containerID)
	if err != nil {
		headers = nil
	}

	data := map[string]any{
		"environments":    environments,
		"workspaceStatus": status,
		"tags":            tags,
		"triggers":        triggers,
		"variables":       variables,
		"versionHeaders":  headers,
		"summary": map[string]int{
			"totalEnvironments": len(environments),
			"totalTags":         len(tags),
			"totalTriggers":     len(triggers),
			"totalVariables":    len(variables),
		},
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Environment promotion checklist",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Generate a promotion checklist for moving GTM changes through environments (dev → staging → production).

## Environment and Workspace Data
%s

Please analyze and generate:

1. **Environment Inventory**
   | Environment | Type | Container Version | URL | Status |
   |-------------|------|------------------|-----|--------|
   Identify: which is dev/staging/production based on name and type

2. **Version Alignment Check**
   - Which version is live (published)?
   - Which version is in each environment?
   - Are environments out of sync?

3. **Pre-Promotion Checklist**
   For each promotion step (dev→staging, staging→production):
   
   #### ✅ Code Quality
   - [ ] All tags have firing triggers
   - [ ] No broken variable references
   - [ ] Naming conventions followed
   
   #### ✅ Compliance
   - [ ] Consent mode configured
   - [ ] No PII in tag parameters
   - [ ] Data destinations approved
   
   #### ✅ Testing
   - [ ] Changes verified in GTM Preview mode
   - [ ] Key conversion events confirmed
   - [ ] No conflicting tags
   
   #### ✅ Approval
   - [ ] Version notes documented
   - [ ] Stakeholders informed

4. **Risk Assessment by Environment**
   | Risk | Dev Impact | Staging Impact | Prod Impact |
   |------|-----------|---------------|-------------|

5. **Rollback Plan**
   For each environment, what version to roll back to if issues are found

6. **Promotion Commands**
   Step-by-step tool commands to execute the promotion`, string(dataJSON)),
				},
			},
		},
	}, nil
}

// handlePlanSGTMSetupPrompt generates a complete sGTM setup plan from scratch.
func handlePlanSGTMSetupPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	goals := req.Params.Arguments["goals"]
	containerType := req.Params.Arguments["containerType"]

	if goals == "" {
		return nil, fmt.Errorf("goals description is required (e.g., 'GA4 + Meta CAPI + consent mode')")
	}

	if containerType == "" {
		containerType = "web+server"
	}

	clientTemplates := GetClientTemplates()
	sgtmTagTemplates := GetServerSideTagTemplates()
	transformationTemplates := GetTransformationTemplates()

	templatesData := map[string]any{
		"clientTemplates":         clientTemplates,
		"serverSideTagTemplates":  sgtmTagTemplates,
		"transformationTemplates": transformationTemplates,
	}

	templatesJSON, err := json.MarshalIndent(templatesData, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Complete sGTM container setup plan",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Generate a complete server-side GTM (sGTM) container setup plan from scratch.

**Tracking Goals:** %s
**Container Type:** %s

## Available sGTM Templates (blueprints)
%s

Create a step-by-step setup plan covering:

1. **Architecture Overview**
   - Web container → sGTM container data flow diagram
   - Which events go through the server vs stay client-side
   - First-party domain setup recommendation (e.g., sgtm.yourdomain.com)

2. **Client Configuration**
   Based on goals, which clients to create:
   | # | Client Name | Type | Priority | Purpose |
   |---|-------------|------|----------|---------|
   Use the clientTemplates above as reference for parameters.

3. **Tag Configuration**
   | # | Tag Name | Type | Trigger | Key Parameters |
   |---|----------|------|---------|----------------|
   Use serverSideTagTemplates above as reference.
   - Include GA4 server tag (almost always needed)
   - Include Conversion Linker (for Google Ads)
   - Map goals to specific tags (CAPI, HTTP Request, etc.)

4. **Transformation Pipeline**
   | # | Transformation Name | Purpose | Fields Modified | Order |
   |---|--------------------|---------|--------------------|-------|
   - PII redaction FIRST (before any outbound tags)
   - Parameter enrichment SECOND
   - Event filtering LAST

5. **Variables to Create**
   | Variable Name | Type | Event Data Key | Used By |
   |---------------|------|----------------|---------|

6. **Built-in Variables to Enable**
   For sGTM: clientName, requestPath, eventName, etc.

7. **Deployment Recommendation**
   - Cloud Run setup (instances, region, scaling)
   - Preview server setup
   - Custom domain DNS configuration
   - Estimated monthly cost

8. **Web Container Changes**
   What to change in the web container to route to sGTM:
   - GA4 Config tag: set server_container_url
   - Which web tags to keep vs migrate to server

9. **Testing Checklist**
   - [ ] Preview mode works for sGTM container
   - [ ] GA4 events arrive in sGTM
   - [ ] Server tags fire and receive 200 responses
   - [ ] PII transformations verified
   - [ ] First-party cookies set correctly

10. **Implementation Order**
    Numbered steps with dependencies marked`, goals, containerType, string(templatesJSON)),
				},
			},
		},
	}, nil
}

// handlePlanWebToSGTMMigrationPrompt analyzes web container for server-side migration.
func handlePlanWebToSGTMMigrationPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
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

	// Include sGTM templates as migration targets
	sgtmTagTemplates := GetServerSideTagTemplates()
	clientTemplates := GetClientTemplates()

	data := map[string]any{
		"webTags":                tags,
		"webTriggers":            triggers,
		"webVariables":           variables,
		"serverSideTagTemplates": sgtmTagTemplates,
		"clientTemplates":        clientTemplates,
		"summary": map[string]int{
			"totalTags":      len(tags),
			"totalTriggers":  len(triggers),
			"totalVariables": len(variables),
		},
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Web-to-sGTM migration plan",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Analyze this web GTM container and generate a migration plan to server-side GTM (sGTM).

## Web Container Configuration + sGTM Templates
%s

Analyze each web tag and classify for migration:

1. **Migration Classification**
   | Web Tag | Type | Migratable? | sGTM Equivalent | Effort | Notes |
   |---------|------|------------|-----------------|--------|-------|
   
   Classification rules:
   - **GA4 tags** → sGTM GA4 tag (inherit config from GA4 client) ✅ Easy
   - **Facebook Pixel** → Meta CAPI tag (Gallery template) ✅ Medium
   - **Google Ads Conversion** → sGTM Google Ads tag ✅ Easy
   - **Custom HTML (API calls)** → HTTP Request tag ✅ Medium
   - **Custom HTML (DOM manipulation)** → ❌ Cannot migrate (DOM-dependent)
   - **Conversion Linker** → sGTM Conversion Linker ✅ Easy
   - **Floodlight** → sGTM Floodlight ✅ Easy

2. **Non-Migratable Tags**
   Tags that MUST stay in the web container:
   | Tag | Reason | Alternative |
   |-----|--------|-------------|
   (DOM-dependent tags, consent banners, client-side UI modifications)

3. **Web Container Post-Migration**
   What the web container should look like AFTER migration:
   - GA4 Config tag with server_container_url set
   - Consent initialization tags
   - Tags that require DOM access
   - Everything else removed/paused

4. **sGTM Container Blueprint**
   Clients, tags, transformations, and triggers to create:
   | Entity Type | Name | Configuration Summary |
   |-------------|------|-----------------------|

5. **Variable Mapping**
   | Web Variable | Type | sGTM Equivalent | How to Access in sGTM |
   |-------------|------|-----------------|----------------------|
   (Data Layer vars → Event Data, DOM vars → not available, cookies → client cookies)

6. **Migration Timeline**
   | Phase | What | Duration | Dependencies |
   |-------|------|----------|-------------|
   Phase 1: Setup sGTM + GA4 (parallel tracking)
   Phase 2: Migrate marketing tags (CAPI, Ads)
   Phase 3: Validate parity, disable web tags
   Phase 4: Clean up web container

7. **Risk Assessment**
   - Data loss risks during migration
   - Conversion tracking gaps
   - Consent mode compatibility
   - Testing strategy for each phase`, string(dataJSON)),
				},
			},
		},
	}, nil
}

// handlePlanSGTMConsentArchitecturePrompt designs consent mode for server-side GTM.
func handlePlanSGTMConsentArchitecturePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]
	cmpPlatform := req.Params.Arguments["cmpPlatform"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	if cmpPlatform == "" {
		cmpPlatform = "unknown"
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

	transformations, err := client.ListTransformations(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list transformations: %w", err)
	}

	variables, err := client.ListVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	data := map[string]any{
		"clients":         clients,
		"tags":            tags,
		"triggers":        triggers,
		"transformations": transformations,
		"variables":       variables,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "sGTM consent mode architecture plan",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Design a consent mode architecture for this server-side GTM container.

Consent in sGTM is complex: the consent state originates in the BROWSER but must be respected on the SERVER. Without proper implementation, the server will send data regardless of user consent.

**CMP Platform:** %s

## sGTM Container Configuration
%s

Design the consent architecture:

1. **Current Consent State**
   - Is consent information being forwarded from the web container?
   - Which event data fields carry consent state (gcs, gcd, consent_mode_*)?
   - Are there any tags firing without consent checks?

2. **Consent Signal Flow**
   ` + "```" + `
   [Browser] → Consent granted/denied
     ↓
   [Web Container] → gtag consent update → sets gcs/gcd parameters
     ↓
   [GA4 Client] → Reads consent from incoming request
     ↓
   [Event Data] → Consent state as event data variable
     ↓
   [Trigger Conditions] → Fire tag only if consent granted
     ↓
   [Transformation] → Strip PII if consent denied
     ↓
   [Tag] → Send only consented data
   ` + "```" + `

3. **Consent Variables to Create**
   | Variable Name | Type | Event Data Key | Purpose |
   |---------------|------|----------------|---------|
   | Consent - Analytics | Event Data | x-ga-gcs | Reads analytics consent |
   | Consent - Ads | Event Data | x-ga-gcd | Reads ad personalization consent |

4. **Trigger Modifications**
   For each existing tag, add consent-aware triggers:
   | Tag | Current Trigger | New Trigger Condition | Consent Type |
   |-----|----------------|----------------------|-------------|

5. **Transformation Gates**
   Transformations to add for consent-conditional data processing:
   | Transformation | When | Action |
   |---------------|------|--------|
   | Strip User Data | analytics_storage denied | Remove user_data.* fields |
   | Strip Ad IDs | ad_storage denied | Remove gclid, fbclid, dclid |
   | Anonymize IP | analytics_storage denied | Enable redactVisitorIp |

6. **CMP Integration (%s)**
   - How %s signals propagate to the server
   - Required web container configuration
   - Server-side consent verification approach

7. **Tag-by-Tag Consent Matrix**
   | Tag | analytics_storage | ad_storage | ad_personalization | Behavior When Denied |
   |-----|-------------------|------------|-------------------|---------------------|

8. **Testing Plan**
   - How to verify consent is respected in Preview Mode
   - Test scenarios: all granted, all denied, partial consent
   - Verification that denied tags don't fire

9. **Implementation Steps**
   Numbered, ordered steps with tool calls`, cmpPlatform, string(dataJSON), cmpPlatform, cmpPlatform),
				},
			},
		},
	}, nil
}

