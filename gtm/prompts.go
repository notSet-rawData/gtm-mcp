package gtm

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterPrompts adds all GTM prompts to the MCP server.
// Handlers are organized by "Job To Be Done" (JTBD):
//   - prompts_audit.go  → "Is my container healthy?"  (10 handlers)
//   - prompts_plan.go   → "What should I do / organize?" (10 handlers)
//   - prompts_debug.go  → "Why doesn't it work / it broke" (8 handlers)
func RegisterPrompts(server *mcp.Server) {

	// =========================================================================
	// AUDIT — "¿Mi container está bien?"
	// =========================================================================

	server.AddPrompt(&mcp.Prompt{
		Name:        "audit_container",
		Description: "Analyze a GTM workspace for potential issues, duplicates, naming inconsistencies, and best practice violations",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleAuditContainerPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "audit_consent_privacy",
		Description: "Audit GTM workspace for GDPR/ePrivacy compliance — check consent mode, tag firing conditions, and potential PII exposure in tracking",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
			{Name: "framework", Description: "Consent framework in use (e.g., 'OneTrust', 'Cookiebot', 'Didomi', 'none')", Required: false},
		},
	}, handleAuditConsentPrivacyPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "audit_server_side",
		Description: "Analyze server-side GTM container configuration — clients, transformations, and tag interactions for sGTM best practices",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The server-side GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleAuditServerSidePrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "audit_custom_templates",
		Description: "Audit custom tag and variable templates for security issues, outdated code, and compliance with Community Template Gallery standards",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleAuditCustomTemplatesPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "validate_variable_references",
		Description: "Audit all workspace variables for broken references, orphaned variables, and circular dependencies — prevents silent production errors",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleValidateVariableReferencesPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "detect_duplicate_tags",
		Description: "Detect redundant or duplicate tags analyzing type, triggers, and key parameters — reduces payload and analytics costs",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleDetectDuplicateTagsPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "optimize_built_in_variables",
		Description: "Review enabled built-in variables and recommend which to enable or disable based on actual usage patterns in the workspace",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleOptimizeBuiltInVariablesPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "audit_sgtm_data_flow",
		Description: "Trace the complete sGTM data pipeline: client → transformation → trigger → tag. Detects broken chains, orphaned paths, and silent event loss",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The server-side GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleAuditSGTMDataFlowPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "audit_sgtm_pii_exposure",
		Description: "Audit sGTM container for PII leaking to third parties — checks transformations, tag parameters, and cookie handling for GDPR compliance",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The server-side GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleAuditSGTMPIIExposurePrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "audit_sgtm_client_priority",
		Description: "Analyze sGTM client priority order, detect claim conflicts, and verify client-to-tag alignment — prevents silent data loss from misconfigured priorities",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The server-side GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleAuditSGTMClientPriorityPrompt)

	// =========================================================================
	// PLAN — "¿Qué hago / cómo organizo?"
	// =========================================================================

	server.AddPrompt(&mcp.Prompt{
		Name:        "generate_tracking_plan",
		Description: "Generate a Markdown tracking plan document from existing tags, triggers, and variables in a workspace",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleGenerateTrackingPlanPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "suggest_ga4_setup",
		Description: "Recommend a GA4 tag structure based on tracking goals and requirements",
		Arguments: []*mcp.PromptArgument{
			{Name: "goals", Description: "Description of tracking goals (e.g., 'ecommerce purchase tracking, form submissions, button clicks')", Required: true},
		},
	}, handleSuggestGA4SetupPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "find_gallery_template",
		Description: "Guide to find and import a Community Template Gallery template by name",
		Arguments: []*mcp.PromptArgument{
			{Name: "templateName", Description: "The name of the template to find (e.g., 'iubenda', 'cookiebot', 'facebook pixel')", Required: true},
		},
	}, handleFindGalleryTemplatePrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "review_before_publish",
		Description: "Review pending workspace changes and provide a pre-publish checklist with risk assessment before creating a version",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleReviewBeforePublishPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "folder_organization_review",
		Description: "Analyze workspace folder structure, detect unorganized entities, and propose reorganization based on naming patterns and implementation type",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleFolderOrganizationReviewPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "migration_plan_ua_to_ga4",
		Description: "Analyze Universal Analytics tags and generate a detailed GA4 migration plan with equivalences, gaps, and implementation order",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
			{Name: "ga4MeasurementId", Description: "The target GA4 Measurement ID (e.g., 'G-XXXXXXXXXX')", Required: true},
		},
	}, handleMigrationPlanUA2GA4Prompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "environment_promotion_checklist",
		Description: "Generate a promotion checklist for safely moving GTM changes through environments (dev → staging → production) with version alignment checks",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The workspace ID with changes to promote", Required: true},
		},
	}, handleEnvironmentPromotionChecklistPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "plan_sgtm_setup",
		Description: "Generate a complete server-side GTM setup plan from scratch — clients, tags, transformations, deployment, and testing checklist based on tracking goals",
		Arguments: []*mcp.PromptArgument{
			{Name: "goals", Description: "Tracking goals (e.g., 'GA4 + Meta CAPI + consent mode')", Required: true},
			{Name: "containerType", Description: "Setup type: 'web+server' (default) or 'server-only'", Required: false},
		},
	}, handlePlanSGTMSetupPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "plan_web_to_sgtm_migration",
		Description: "Analyze a web GTM container and generate a migration plan to server-side — classifies web tags as migratable or DOM-dependent, maps to sGTM equivalents",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The web GTM container ID to analyze for migration", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handlePlanWebToSGTMMigrationPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "plan_sgtm_consent_architecture",
		Description: "Design consent mode architecture for sGTM — maps browser consent signals to server-side enforcement via variables, triggers, and transformations",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The server-side GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
			{Name: "cmpPlatform", Description: "CMP platform in use (e.g., 'OneTrust', 'Cookiebot', 'Didomi')", Required: false},
		},
	}, handlePlanSGTMConsentArchitecturePrompt)

	// =========================================================================
	// DEBUG — "¿Por qué no funciona? / Se rompió algo"
	// =========================================================================

	server.AddPrompt(&mcp.Prompt{
		Name:        "debug_trigger_coverage",
		Description: "Analyze which pages/events may not be covered by any trigger, and which triggers have conditions so restrictive they probably never fire",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
			{Name: "urlPatterns", Description: "Optional list of URL patterns from your site to verify trigger coverage against", Required: false},
		},
	}, handleDebugTriggerCoveragePrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "tag_firing_dependency_graph",
		Description: "Build the complete firing dependency graph between tags, triggers, variables, and tag sequences — essential for debugging execution order",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
		},
	}, handleTagFiringDependencyGraphPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "compare_workspaces",
		Description: "Compare two workspaces in the same container to identify divergences in tags, triggers, and variables — ideal before merging parallel work",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceIdA", Description: "First workspace ID to compare", Required: true},
			{Name: "workspaceIdB", Description: "Second workspace ID to compare", Required: true},
		},
	}, handleCompareWorkspacesPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "diff_versions",
		Description: "Compare two published container versions and generate a structured changelog with field-level diffs and data collection impact analysis — essential for post-mortems",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "versionIdA", Description: "Base version ID (older/before)", Required: true},
			{Name: "versionIdB", Description: "Target version ID (newer/after)", Required: true},
		},
	}, handleDiffVersionsPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "sync_workspace_conflicts",
		Description: "Sync workspace with latest container version, detect merge conflicts, and provide resolution guidance with dependency-aware ordering",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The workspace ID to sync", Required: true},
		},
	}, handleSyncWorkspaceConflictsPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "rollback_strategy",
		Description: "Analyze container version history around an incident date and recommend the optimal rollback target with trade-off analysis — for incident response",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The GTM container ID", Required: true},
			{Name: "incidentDate", Description: "Date when the data anomaly was first detected (e.g., '2025-03-10')", Required: true},
			{Name: "symptomDescription", Description: "Description of the observed problem (e.g., 'conversion tracking dropped 50%')", Required: false},
		},
	}, handleRollbackStrategyPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "debug_sgtm_event_loss",
		Description: "Diagnose silent event loss in sGTM — walks through all 6 pipeline stages (client claim → event data → transformation → trigger → tag → outbound) to find where events disappear",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The server-side GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
			{Name: "symptom", Description: "Description of the event loss (e.g., 'GA4 events missing', 'CAPI conversions not showing')", Required: false},
		},
	}, handleDebugSGTMEventLossPrompt)

	server.AddPrompt(&mcp.Prompt{
		Name:        "debug_sgtm_tag_response",
		Description: "Debug HTTP errors (400/401/403/500) from sGTM outbound tags — analyzes variable resolution, transformation impact, and simulates payloads to find the issue",
		Arguments: []*mcp.PromptArgument{
			{Name: "accountId", Description: "The GTM account ID", Required: true},
			{Name: "containerId", Description: "The server-side GTM container ID", Required: true},
			{Name: "workspaceId", Description: "The GTM workspace ID", Required: true},
			{Name: "tagId", Description: "Optional specific tag ID to focus debugging on", Required: false},
		},
	}, handleDebugSGTMTagResponsePrompt)
}

