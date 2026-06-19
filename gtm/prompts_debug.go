package gtm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func handleDebugTriggerCoveragePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]
	urlPatterns := req.Params.Arguments["urlPatterns"]

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

	tagTriggerUsage := make(map[string][]string)
	for _, tag := range tags {
		for _, tid := range tag.FiringTriggerID {
			tagTriggerUsage[tid] = append(tagTriggerUsage[tid], tag.Name)
		}
	}

	data := map[string]any{
		"tags":            tags,
		"triggers":        triggers,
		"triggerMap":      triggerMap,
		"tagTriggerUsage": tagTriggerUsage,
	}

	if urlPatterns != "" {
		data["urlPatterns"] = urlPatterns
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Trigger coverage analysis",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Analyze trigger coverage in this GTM workspace — find gaps and overly restrictive triggers.

## Workspace Data
%s

Please analyze:

1. **Orphaned Triggers** (defined but not used by any tag)
   | Trigger Name | Type | Conditions | Used By Tags |
   |-------------|------|-----------|-------------|
   → Candidates for deletion or wiring to tags

2. **Tags Without Triggers**
   - Tags that have no firing trigger assigned (will never fire)

3. **Overly Restrictive Triggers**
   - Triggers with conditions so specific they probably never match
   - Examples: exact URL match on development URLs, impossible filter combinations

4. **Coverage Gaps**
   Expected triggers that are MISSING:
   - No All Pages pageview trigger → GA4 Config won't fire
   - No form submission triggers → forms not tracked
   - No scroll tracking → engagement unmeasured
   - No click triggers → CTA clicks not tracked

5. **Trigger Overlap**
   - Triggers that fire in the same conditions (potential double-fires)
   - Trigger Groups that may conflict

6. **Recommendations**
   - Triggers to add for standard coverage
   - Triggers to simplify (overly complex filters)
   - Triggers to remove (orphaned)`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleTagFiringDependencyGraphPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
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
		Description: "Tag firing dependency graph",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Build the complete execution dependency graph for this GTM workspace.

## Workspace Configuration
%s

Please generate:

1. **Tag Execution Chain**
   For each tag, trace the full execution path:
   - Variables resolved → Trigger evaluates → Tag fires → Setup/teardown sequence

2. **Setup/Teardown Sequences**
   - Tags with setupTag or teardownTag references
   - Execution order: setup → main → teardown
   - stopOnFailure implications

3. **Trigger Groups**
   - triggerGroup type triggers that aggregate multiple triggers
   - Effective firing condition of each group

4. **Dependency Graph** (Mermaid diagram)
   Generate a Mermaid graph showing:
   - Trigger → Tag relationships
   - Tag → Tag relationships (setup/teardown)
   - Variable → Tag/Trigger dependencies

5. **Execution Timeline**
   For a typical pageview:
   | Order | Event | Triggers | Tags Fired |
   |-------|-------|----------|-----------|

6. **Risk Areas**
   - Circular dependencies
   - Tags with stopOnFailure that could break the chain
   - Tags depending on variables resolved by other tags`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleCompareWorkspacesPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceIDA := req.Params.Arguments["workspaceIdA"]
	workspaceIDB := req.Params.Arguments["workspaceIdB"]

	if accountID == "" || containerID == "" || workspaceIDA == "" || workspaceIDB == "" {
		return nil, fmt.Errorf("accountId, containerId, workspaceIdA, and workspaceIdB are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	tagsA, _ := client.ListTags(ctx, accountID, containerID, workspaceIDA)
	triggersA, _ := client.ListTriggers(ctx, accountID, containerID, workspaceIDA)
	variablesA, _ := client.ListVariables(ctx, accountID, containerID, workspaceIDA)

	tagsB, _ := client.ListTags(ctx, accountID, containerID, workspaceIDB)
	triggersB, _ := client.ListTriggers(ctx, accountID, containerID, workspaceIDB)
	variablesB, _ := client.ListVariables(ctx, accountID, containerID, workspaceIDB)

	data := map[string]any{
		"workspaceA": map[string]any{
			"id":        workspaceIDA,
			"tags":      tagsA,
			"triggers":  triggersA,
			"variables": variablesA,
			"counts": map[string]int{
				"tags":      len(tagsA),
				"triggers":  len(triggersA),
				"variables": len(variablesA),
			},
		},
		"workspaceB": map[string]any{
			"id":        workspaceIDB,
			"tags":      tagsB,
			"triggers":  triggersB,
			"variables": variablesB,
			"counts": map[string]int{
				"tags":      len(tagsB),
				"triggers":  len(triggersB),
				"variables": len(variablesB),
			},
		},
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Workspace comparison",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Compare these two GTM workspaces and identify divergences.

## Workspace Data
%s

Please provide:

1. **Summary**
   | Metric | Workspace A | Workspace B |
   |--------|------------|------------|
   | Tags   | ... | ... |
   | Triggers | ... | ... |
   | Variables | ... | ... |

2. **Tags Diff**
   - Present only in A
   - Present only in B
   - Present in both but modified (different config)

3. **Triggers Diff**
   - Same analysis as tags

4. **Variables Diff**
   - Same analysis as tags

5. **Conflict Risk Assessment**
   - Can these workspaces be merged safely?
   - Which entities would conflict?

6. **Merge Recommendation**
   - Which workspace is more complete?
   - Suggested merge strategy`, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleDiffVersionsPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	versionIDA := req.Params.Arguments["versionIdA"]
	versionIDB := req.Params.Arguments["versionIdB"]

	if accountID == "" || containerID == "" || versionIDA == "" || versionIDB == "" {
		return nil, fmt.Errorf("accountId, containerId, versionIdA, and versionIdB are required")
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	versionA, err := client.GetContainerVersion(ctx, accountID, containerID, versionIDA)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %s: %w", versionIDA, err)
	}

	versionB, err := client.GetContainerVersion(ctx, accountID, containerID, versionIDB)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %s: %w", versionIDB, err)
	}

	headers, err := client.ListVersionHeaders(ctx, accountID, containerID)
	if err != nil {
		headers = nil
	}

	data := map[string]any{
		"versionA":       versionA,
		"versionB":       versionB,
		"versionHeaders": headers,
		"summary": map[string]any{
			"versionA": map[string]any{
				"id":       versionIDA,
				"name":     versionA.Name,
				"numTags":  len(versionA.Tags),
				"numTrigs": len(versionA.Triggers),
				"numVars":  len(versionA.Variables),
			},
			"versionB": map[string]any{
				"id":       versionIDB,
				"name":     versionB.Name,
				"numTags":  len(versionB.Tags),
				"numTrigs": len(versionB.Triggers),
				"numVars":  len(versionB.Variables),
			},
		},
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Container version diff and changelog",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Compare these two GTM container versions and generate a structured changelog.

## Version Data
%s

Please generate a comprehensive diff analysis:

1. **Version Overview**
   | | Version A (base) | Version B (target) |
   |---|---|---|
   | ID | %s | %s |
   | Name | %s | %s |
   | Tags | %d | %d |
   | Triggers | %d | %d |
   | Variables | %d | %d |

2. **Structured Changelog**
   
   ### Tags
   | Change | Tag Name | Type | Details |
   |--------|----------|------|---------|
   For each: ADDED / MODIFIED / DELETED with specific field changes

   ### Triggers
   | Change | Trigger Name | Type | Details |
   |--------|-------------|------|---------|

   ### Variables
   | Change | Variable Name | Type | Details |
   |--------|-------------|------|---------|

3. **Field-Level Diff for Modifications**
   For each modified entity, show:
   - Field name → old value → new value
   - Highlight measurement-impacting changes (eventName, trackingId, sendTo)

4. **Impact Analysis**
   - 🔴 **High-Risk Changes**: deleted active tags, changed trackingId, modified All Pages triggers
   - 🟡 **Medium-Risk**: new tags without blocking triggers, changed event parameters
   - 🟢 **Low-Risk**: naming changes, paused/unpaused, folder moves

5. **Data Collection Impact**
   - Which events are newly tracked?
   - Which events stopped being tracked?
   - Which events changed their parameters?

6. **Timeline Context**
   If version headers are available, show where these versions sit in the publish timeline.

7. **Verdict**
   Could version B be responsible for a data anomaly? Summarize the most likely impact.`,
						string(dataJSON),
						versionIDA, versionIDB,
						versionA.Name, versionB.Name,
						len(versionA.Tags), len(versionB.Tags),
						len(versionA.Triggers), len(versionB.Triggers),
						len(versionA.Variables), len(versionB.Variables)),
				},
			},
		},
	}, nil
}

func handleSyncWorkspaceConflictsPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
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

	syncStatus, err := client.SyncWorkspace(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to sync workspace: %w", err)
	}

	wsStatus, err := client.GetWorkspaceStatus(ctx, accountID, containerID, workspaceID)
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
		"syncResult":      syncStatus,
		"workspaceStatus": wsStatus,
		"tags":            tags,
		"triggers":        triggers,
		"variables":       variables,
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Workspace sync conflict resolution",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`The workspace has been synced with the latest container version. Analyze the results and guide conflict resolution.

## Sync Results and Workspace Data
%s

Please analyze:

1. **Sync Summary**
   - Conflicts detected: %d
   - Conflicting entities: %v
   - Workspace has pending changes: %v

2. **Conflict Details**
   For each conflicting entity:
   | Entity | Type | Workspace Version | Base Version | Conflict Type |
   |--------|------|-------------------|-------------|--------------|
   
   Conflict types:
   - **Edit/Edit**: Both workspace and base modified the same entity
   - **Edit/Delete**: One side edited, other deleted
   - **Delete/Edit**: One side deleted, other edited

3. **Resolution Strategy**
   For each conflict, recommend:
   | Entity | Recommendation | Rationale |
   |--------|---------------|-----------|
   Options: Keep workspace version / Accept base version / Manual merge needed

4. **Safe Resolution Order**
   Order conflicts by dependency — resolve variables first, then triggers, then tags:
   1. Variables (other entities depend on them)
   2. Triggers (tags depend on them)
   3. Tags (leaf nodes)

5. **Post-Resolution Verification**
   After resolving conflicts:
   - Re-run workspace sync to confirm no remaining conflicts
   - Verify no broken references
   - Preview changes in GTM debug mode

6. **Prevention Tips**
   Based on the conflict patterns, suggest collaboration improvements:
   - Should team members use separate workspaces?
   - Are there naming conventions that could prevent conflicts?
   - Would a branching strategy help?`, string(dataJSON), syncStatus.ConflictCount, syncStatus.ConflictingEntities, wsStatus.HasChanges),
				},
			},
		},
	}, nil
}

func handleRollbackStrategyPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	incidentDate := req.Params.Arguments["incidentDate"]
	symptomDescription := req.Params.Arguments["symptomDescription"]

	if accountID == "" || containerID == "" || incidentDate == "" {
		return nil, fmt.Errorf("accountId, containerId, and incidentDate are required")
	}

	if symptomDescription == "" {
		symptomDescription = "Data anomaly detected"
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	headers, err := client.ListVersionHeaders(ctx, accountID, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list version headers: %w", err)
	}

	maxVersions := 5
	if len(headers) < maxVersions {
		maxVersions = len(headers)
	}

	versions := make([]*ContainerVersionDetail, 0, maxVersions)
	for i := 0; i < maxVersions; i++ {
		v, err := client.GetContainerVersion(ctx, accountID, containerID, headers[i].VersionID)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}

	data := map[string]any{
		"incidentDate":       incidentDate,
		"symptomDescription": symptomDescription,
		"versionHeaders":     headers,
		"recentVersions":     versions,
		"totalVersions":      len(headers),
		"detailedVersions":   len(versions),
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "Rollback strategy analysis",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Analyze the version history of this GTM container and generate an optimal rollback strategy.

**Incident Date:** %s
**Symptom:** %s

## Version History and Recent Version Details
%s

Please analyze and recommend:

1. **Timeline Analysis**
   Map versions against the incident date:
   | Version ID | Name | # Tags | # Triggers | # Variables | Relationship to Incident |
   |-----------|------|--------|-----------|------------|------------------------|
   Mark versions as: BEFORE incident / SUSPECTED / AFTER incident

2. **Suspect Versions**
   Identify versions published around incidentDate that could have caused the symptom:
   - What changed in each suspect version?
   - Which changes could explain the observed symptom?

3. **Rollback Target**
   Recommend the best version to roll back to:
   - **Target Version:** [ID and name]
   - **Rationale:** Why this version specifically
   - **Last known good state:** Confirm this version was stable

4. **Trade-off Analysis**
   If we roll back to the target version, what do we lose?
   | Entity | Present in Target? | Lost if Rolled Back |
   |--------|-------------------|-------------------|
   List tags/triggers/variables that exist in current version but NOT in the target

5. **Selective vs. Full Rollback**
   - Can the issue be fixed by reverting only specific entities?
   - Or is a full version rollback necessary?
   - If selective: list exact entities to revert

6. **Rollback Execution Plan**
   Step-by-step:
   1. Verify the rollback target in GTM Preview mode
   2. Use set_latest_version to revert (tool command)
   3. Verify in production
   
   ⚠️ The set_latest_version action is IRREVERSIBLE — confirm with the user before proceeding.

7. **Post-Rollback Verification**
   - Key events to verify are still firing
   - Expected data points to check in GA4/BigQuery
   - Estimated time for data to normalize

8. **Impact Window**
   - Estimated start of data impact
   - Estimated duration of impact
   - Data recovery recommendations`, incidentDate, symptomDescription, string(dataJSON)),
				},
			},
		},
	}, nil
}

func handleDebugSGTMEventLossPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]
	symptom := req.Params.Arguments["symptom"]

	if accountID == "" || containerID == "" || workspaceID == "" {
		return nil, fmt.Errorf("accountId, containerId, and workspaceId are required")
	}

	if symptom == "" {
		symptom = "Events appear to be missing or not reaching their destination"
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
		Description: "sGTM event loss diagnosis",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Diagnose event loss in this server-side GTM container.

**Reported Symptom:** %s

In sGTM, event loss is SILENT — there are no browser console errors, no visible warnings. Events simply disappear at one of several pipeline stages. This makes systematic debugging essential.

## sGTM Container Configuration
%s

Walk through each pipeline stage and identify where events could be lost:

1. **Stage 1: Request Arrival → Client Claim**
   Check:
   - Is there a client configured to claim the incoming request type?
   - Does the client's priority allow it to claim before other clients?
   - Is the client's request path filter matching the actual request path?
   🔍 Common failure: Requests arriving at /collect but client filters on a different path.

2. **Stage 2: Client → Event Data Object**
   Check:
   - Is the client generating the expected event data structure?
   - Are required fields present in the event data (event_name, client_id)?
   🔍 Common failure: Client claims the request but fails to parse it into valid event data.

3. **Stage 3: Transformation → Event Modification**
   Check:
   - Are transformations removing fields needed by downstream tags?
   - Are transformations filtering out events entirely?
   - Transformation execution order vs tag requirements?
   🔍 Common failure: PII redaction transformation removes event_name or other required fields.

4. **Stage 4: Trigger Evaluation**
   Check:
   - Do trigger conditions match the event data produced by clients?
   - Are triggers checking for Client Name that matches the actual client name?
   - Are there filter conditions that are too restrictive?
   🔍 Common failure: Trigger checks for Client Name = "GA4" but actual client name is "Google Analytics: GA4".

5. **Stage 5: Tag Firing**
   Check:
   - Does the tag have the correct configuration (measurement ID, API endpoint)?
   - Are required parameters present (not resolved from missing variables)?
   - Is the tag paused?
   🔍 Common failure: Tag references a variable that returns undefined because event data key doesn't exist.

6. **Stage 6: Outbound Request → Destination**
   Check:
   - Is the outbound HTTP request reaching the destination?
   - Is the destination returning 200 OK or an error?
   - Are authentication tokens valid and not expired?
   🔍 Common failure: CAPI tag gets 400 because access token expired.

7. **Debugging Checklist**
   For the specific symptom "%s":
   - [ ] Open sGTM Preview Mode → check if requests appear
   - [ ] Click on request → verify client claimed it
   - [ ] Check Event Data tab → verify fields present
   - [ ] Check Tags tab → verify tag fired (not blocked)
   - [ ] Check Outgoing Requests → verify HTTP 200 response
   - [ ] Check Console tab → look for error messages

8. **Most Likely Root Cause**
   Based on the container configuration and symptom, what is the most probable failure point?`, symptom, string(dataJSON), symptom),
				},
			},
		},
	}, nil
}

func handleDebugSGTMTagResponsePrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	accountID := req.Params.Arguments["accountId"]
	containerID := req.Params.Arguments["containerId"]
	workspaceID := req.Params.Arguments["workspaceId"]
	tagId := req.Params.Arguments["tagId"]

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

	variables, err := client.ListVariables(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list variables: %w", err)
	}

	transformations, err := client.ListTransformations(ctx, accountID, containerID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list transformations: %w", err)
	}

	data := map[string]any{
		"tags":            tags,
		"variables":       variables,
		"transformations": transformations,
	}

	if tagId != "" {
		data["focusTagId"] = tagId
	}

	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	return &mcp.GetPromptResult{
		Description: "sGTM tag HTTP response debugging",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: fmt.Sprintf(`Debug HTTP errors from outbound tags in this server-side GTM container.

sGTM tags send HTTP requests to external APIs (Google Analytics, Meta CAPI, custom endpoints). When these requests fail, the errors are only visible in sGTM Preview Mode or Cloud Logging — they are invisible to the end user.

## sGTM Container Configuration
%s

For each outbound tag, analyze potential failure points:

1. **Tag Inventory — Outbound Requests**
   | Tag Name | Type | Destination URL | Auth Method | Risk |
   |----------|------|----------------|-------------|------|

2. **Common HTTP Error Analysis**
   For each tag type, check for common error patterns:

   **400 Bad Request:**
   - Missing required parameters in the request body?
   - Malformed JSON in HTTP Request tag body?
   - Variables that resolve to empty/undefined?
   - Parameters not matching the API's expected format?

   **401 Unauthorized:**
   - Expired access tokens (especially CAPI, custom APIs)?
   - Missing Authorization headers in HTTP Request tags?
   - API keys hardcoded but rotated on the server?

   **403 Forbidden:**
   - IP restrictions on the destination API?
   - Insufficient API permissions/scopes?
   - Rate limiting by the destination?

   **404 Not Found:**
   - Incorrect endpoint URL?
   - API version deprecated?

   **500 Server Error:**
   - Destination API is down?
   - Payload size too large?

3. **Variable Resolution Check**
   For each tag's parameters, verify the variable chain:
   | Tag | Parameter | Variable | Resolves From | Could Be Empty? |
   |-----|-----------|----------|---------------|----------------|
   Highlight variables that depend on event data keys that may not exist for all events.

4. **Transformation Impact**
   Are transformations modifying or removing fields that tags need?
   | Transformation | Removes/Modifies | Tags Affected | Risk |
   |---------------|-----------------|--------------|------|

5. **Payload Simulation**
   For each critical tag, reconstruct the likely outbound payload:
   `+"```json"+`
   {
     "data": [{
       "event_name": "{{Event Name}}",
       "event_time": "{{Event Timestamp}}",
       "action_source": "website",
       "user_data": { ... }
     }]
   }
   `+"```"+`
   Identify fields that would be missing or malformed.

6. **Monitoring Recommendations**
   - Enable logToConsole in tag templates for debugging
   - Set up Cloud Logging alerts for tag errors
   - Monitor /healthy endpoint for server health
   - Recommended GCP Logs Explorer queries for each tag type

7. **Fix Priority**
   | Priority | Tag | Issue | Fix |
   |----------|-----|-------|-----|`, string(dataJSON)),
				},
			},
		},
	}, nil
}
