package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GatewayInput is the single input type for the unified "gtm" tool.
// It captures the resource and action, plus typed arguments that will be
// deserialized into the appropriate resource-specific input type.
type GatewayInput struct {
	Resource string                 `json:"resource" jsonschema:"enum:account,container,workspace,tag,trigger,variable,folder,template,built_in_variable,client,transformation,environment,user_permission,version,destination,zone,gtag_config,templates_ref,ping,auth_status,description:The GTM resource type to operate on"`
	Action   string                 `json:"action" jsonschema:"description:The action to perform on the resource (e.g. list, get, create, update, delete, revert). Available actions vary by resource."`
	Args     map[string]interface{} `json:"args,omitempty" jsonschema:"description:Resource-specific parameters as a JSON object. Contents depend on the resource and action."`
}

// registerGateway registers the single unified "gtm" tool.
func registerGateway(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "gtm",
		Description: `Unified Google Tag Manager gateway. ALL GTM operations go through this single tool.

USAGE: {"resource": "<resource>", "action": "<action>", "args": {<params>}}

RESOURCES & ACTIONS:
  account → list
  container → list, create, delete
  workspace → list, create, status
  tag → list, get, create, update, delete, revert
  trigger → list, get, create, update, delete, revert
  variable → list, get, create, update, delete, revert, add_lookup_entry, remove_lookup_entry, list_lookup_entries
  folder → list, get, create, update, delete, move, audit, revert
  template → list, get, create, update, delete, import, revert
  built_in_variable → list, enable, disable, revert
  client → list, get, create, update, delete, revert
  transformation → list, get, create, update, delete, revert
  environment → list, get, create, update, delete
  user_permission → list, get, create, update, delete
  version → list, get, create, publish, compare, find_by_date, set_latest, export, import
  destination → list, get, link
  zone → list, get, create, update, delete, revert
  gtag_config → list, get, create, update, delete
  templates_ref → tag_templates, trigger_templates
  ping, auth_status → (no action needed)

CRITICAL: UPDATE BEHAVIOR (PARTIAL UPDATES)
  Update operations are PARTIAL — fields you omit are PRESERVED from the existing entity.
  You only need to send the fields you want to CHANGE, plus required identifiers (accountId, containerId, workspaceId, and the entity ID).
  Example: To rename a variable without changing its parameters:
    {"resource": "variable", "action": "update", "args": {"accountId": "...", "containerId": "...", "workspaceId": "...", "variableId": "...", "name": "New Name", "type": "smm"}}
  The existing parameter array, notes, and parentFolderId will be preserved automatically.
  To ADD an entry to an existing parameter list (e.g. a RegEx table), you must GET the current entity first, then send the FULL updated parameter array with the new entry added.

PARAMETER STRUCTURE for tags, triggers, variables:
  The "parameter" field is an array of parameter objects. Each parameter has:
    - type: "template" (string value), "boolean", "integer", "list" (array), "map" (key-value pairs)
    - key: the parameter name (e.g. "html", "pixelId", "trackingId")
    - value: the parameter value (for template/boolean/integer types)
    - list: array of sub-parameters (for type "list")
    - map: array of sub-parameters (for type "map")
  Example parameter array for a Custom HTML tag:
    [{"type": "template", "key": "html", "value": "<script>...</script>"}, {"type": "boolean", "key": "supportDocumentWrite", "value": "false"}]

COMMUNITY TEMPLATES (gallery tags):
  Community/gallery templates have type IDs like "cvt_CONTAINERID_NNN" (e.g. "cvt_36936833_663").
  DO NOT use generic names like "facebook_pixel" or "meta_pixel" — these are NOT valid GTM types.
  To find the correct template type ID:
    1. Use {"resource": "templates_ref", "action": "tag_templates"} to list available templates
    2. Or GET an existing tag of the same template type to see its type ID
    3. The type ID is container-specific and must match exactly

COMMON ARGS (required for most create/update/delete/get operations):
  accountId, containerId, workspaceId — always required
  For update: also include the entity ID (tagId, triggerId, variableId, etc.)
  For create: include name, type, and type-specific parameters

EXAMPLES:
  List accounts: {"resource": "account", "action": "list", "args": {}}
  Get a tag: {"resource": "tag", "action": "get", "args": {"accountId": "123", "containerId": "456", "workspaceId": "7", "tagId": "89"}}
  Create Custom HTML tag: {"resource": "tag", "action": "create", "args": {"accountId": "123", "containerId": "456", "workspaceId": "7", "name": "My Tag", "type": "html", "parameter": [{"type": "template", "key": "html", "value": "<script>console.log('hi')</script>"}], "firingTriggerIds": ["2147479553"]}}
  Update tag name only: {"resource": "tag", "action": "update", "args": {"accountId": "123", "containerId": "456", "workspaceId": "7", "tagId": "89", "name": "Renamed Tag", "type": "html"}}
  Export version: {"resource": "version", "action": "export", "args": {"accountId": "123", "containerId": "456", "versionId": "1"}}
  Check auth: {"resource": "auth_status", "action": "", "args": {}}

LOOKUP TABLE OPERATIONS (for RegEx Table variables):
  Instead of manually GET+modify+PUT entire parameter arrays, use these high-level actions:
  Add entries:    {"resource": "variable", "action": "add_lookup_entry", "args": {"accountId": "...", "containerId": "...", "workspaceId": "...", "variableId": "676", "entries": [{"pattern": "^myPattern$", "output": "myValue"}]}}
  Remove entries: {"resource": "variable", "action": "remove_lookup_entry", "args": {"accountId": "...", "containerId": "...", "workspaceId": "...", "variableId": "676", "patterns": ["^myPattern$"]}}
  List entries:   {"resource": "variable", "action": "list_lookup_entries", "args": {"accountId": "...", "containerId": "...", "workspaceId": "...", "variableId": "676"}}
  These handle the full GET→merge→UPDATE cycle internally. Duplicates are detected automatically.

IMPORTANT: When using manual UPDATE (not the lookup helpers above):
  - The "name" and "type" fields are REQUIRED, even if unchanged
  - You must send the FULL parameter array, not just the changed entry`,
	}, func(ctx context.Context, req *mcp.CallToolRequest, input GatewayInput) (*mcp.CallToolResult, any, error) {
		return routeGateway(ctx, input)
	})
}

// routeGateway dispatches to the correct resource handler.
func routeGateway(ctx context.Context, input GatewayInput) (*mcp.CallToolResult, any, error) {
	switch input.Resource {
	case "account":
		return routeAccount(ctx, input)
	case "container":
		return routeContainer(ctx, input)
	case "workspace":
		return routeWorkspace(ctx, input)
	case "tag":
		return routeTag(ctx, input)
	case "trigger":
		return routeTrigger(ctx, input)
	case "variable":
		return routeVariable(ctx, input)
	case "folder":
		return routeFolder(ctx, input)
	case "template":
		return routeTemplate(ctx, input)
	case "built_in_variable":
		return routeBuiltInVariable(ctx, input)
	case "client":
		return routeClient(ctx, input)
	case "transformation":
		return routeTransformation(ctx, input)
	case "environment":
		return routeEnvironment(ctx, input)
	case "user_permission":
		return routeUserPermission(ctx, input)
	case "version":
		return routeVersion(ctx, input)
	case "destination":
		return routeDestination(ctx, input)
	case "zone":
		return routeZone(ctx, input)
	case "gtag_config":
		return routeGtagConfig(ctx, input)
	case "templates_ref":
		return routeTemplatesRef(ctx, input)
	case "ping":
		return routePing(ctx, input)
	case "auth_status":
		return routeAuthStatus(ctx, input)
	default:
		return nil, nil, fmt.Errorf(
			"unknown resource %q — valid resources: account, container, workspace, tag, trigger, variable, folder, template, built_in_variable, client, transformation, environment, user_permission, version, destination, zone, gtag_config, templates_ref, ping, auth_status",
			input.Resource,
		)
	}
}

// unmarshalArgs deserializes the args map into the given typed input struct.
// If args is nil or empty, the target struct will have zero values.
func unmarshalArgs(args map[string]interface{}, target interface{}) error {
	if len(args) == 0 {
		return nil
	}
	// Marshal back to JSON, then unmarshal into the typed struct
	raw, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal args: %w", err)
	}
	return json.Unmarshal(raw, target)
}

// --- Resource routers ---

func routeAccount(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input AccountToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for account: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleListAccounts(ctx)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource account — valid actions: list", input.Action)
	}
}

func routeContainer(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input ContainerToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for container: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleListContainers(ctx, input)
	case "create":
		return handleCreateContainer(ctx, input)
	case "delete":
		return handleDeleteContainer(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource container — valid actions: list, create, delete", input.Action)
	}
}

func routeWorkspace(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input WorkspaceToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for workspace: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleListWorkspaces(ctx, input)
	case "create":
		return handleCreateWorkspace(ctx, input)
	case "status":
		return handleGetWorkspaceStatus(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource workspace — valid actions: list, create, status", input.Action)
	}
}

func routeTag(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input TagToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for tag: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleTagList(ctx, input)
	case "get":
		return handleTagGet(ctx, input)
	case "create":
		return handleTagCreate(ctx, input)
	case "update":
		return handleTagUpdate(ctx, input)
	case "delete":
		return handleTagDelete(ctx, input)
	case "revert":
		return handleTagRevert(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource tag — valid actions: list, get, create, update, delete, revert", input.Action)
	}
}

func routeTrigger(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input TriggerToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for trigger: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleTriggerList(ctx, input)
	case "get":
		return handleTriggerGet(ctx, input)
	case "create":
		return handleTriggerCreate(ctx, input)
	case "update":
		return handleTriggerUpdate(ctx, input)
	case "delete":
		return handleTriggerDelete(ctx, input)
	case "revert":
		return handleTriggerRevert(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource trigger — valid actions: list, get, create, update, delete, revert", input.Action)
	}
}

func routeVariable(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input VariableToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for variable: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleVariableList(ctx, input)
	case "get":
		return handleVariableGet(ctx, input)
	case "create":
		return handleVariableCreate(ctx, input)
	case "update":
		return handleVariableUpdate(ctx, input)
	case "delete":
		return handleVariableDelete(ctx, input)
	case "revert":
		return handleVariableRevert(ctx, input)
	case "add_lookup_entry":
		return handleVariableAddLookupEntry(ctx, input)
	case "remove_lookup_entry":
		return handleVariableRemoveLookupEntry(ctx, input)
	case "list_lookup_entries":
		return handleVariableListLookupEntries(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource variable — valid actions: list, get, create, update, delete, revert, add_lookup_entry, remove_lookup_entry, list_lookup_entries", input.Action)
	}
}

func routeFolder(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input FolderToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for folder: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleFolderList(ctx, input)
	case "get":
		return handleFolderGet(ctx, input)
	case "create":
		return handleFolderCreate(ctx, input)
	case "update":
		return handleFolderUpdate(ctx, input)
	case "delete":
		return handleFolderDelete(ctx, input)
	case "move":
		return handleFolderMove(ctx, input)
	case "audit":
		return handleFolderAudit(ctx, input)
	case "revert":
		return handleFolderRevert(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource folder — valid actions: list, get, create, update, delete, move, audit, revert", input.Action)
	}
}

func routeTemplate(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input TemplateToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for template: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleTemplateList(ctx, input)
	case "get":
		return handleTemplateGet(ctx, input)
	case "create":
		return handleTemplateCreate(ctx, input)
	case "update":
		return handleTemplateUpdate(ctx, input)
	case "delete":
		return handleTemplateDelete(ctx, input)
	case "import":
		return handleTemplateImport(ctx, input)
	case "revert":
		return handleTemplateRevert(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource template — valid actions: list, get, create, update, delete, import, revert", input.Action)
	}
}

func routeBuiltInVariable(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input BuiltInVariableToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for built_in_variable: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleBuiltInVariableList(ctx, input)
	case "enable":
		return handleBuiltInVariableEnable(ctx, input)
	case "disable":
		return handleBuiltInVariableDisable(ctx, input)
	case "revert":
		return handleBuiltInVariableRevert(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource built_in_variable — valid actions: list, enable, disable, revert", input.Action)
	}
}

func routeClient(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input ClientToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for client: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleClientList(ctx, input)
	case "get":
		return handleClientGet(ctx, input)
	case "create":
		return handleClientCreate(ctx, input)
	case "update":
		return handleClientUpdate(ctx, input)
	case "delete":
		return handleClientDelete(ctx, input)
	case "revert":
		return handleClientRevert(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource client — valid actions: list, get, create, update, delete, revert", input.Action)
	}
}

func routeTransformation(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input TransformationToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for transformation: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleTransformationList(ctx, input)
	case "get":
		return handleTransformationGet(ctx, input)
	case "create":
		return handleTransformationCreate(ctx, input)
	case "update":
		return handleTransformationUpdate(ctx, input)
	case "delete":
		return handleTransformationDelete(ctx, input)
	case "revert":
		return handleTransformationRevert(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource transformation — valid actions: list, get, create, update, delete, revert", input.Action)
	}
}

func routeEnvironment(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input EnvironmentToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for environment: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleEnvironmentList(ctx, input)
	case "get":
		return handleEnvironmentGet(ctx, input)
	case "create":
		return handleEnvironmentCreate(ctx, input)
	case "update":
		return handleEnvironmentUpdate(ctx, input)
	case "delete":
		return handleEnvironmentDelete(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource environment — valid actions: list, get, create, update, delete", input.Action)
	}
}

func routeUserPermission(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input UserPermissionToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for user_permission: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleUserPermissionList(ctx, input)
	case "get":
		return handleUserPermissionGet(ctx, input)
	case "create":
		return handleUserPermissionCreate(ctx, input)
	case "update":
		return handleUserPermissionUpdate(ctx, input)
	case "delete":
		return handleUserPermissionDelete(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource user_permission — valid actions: list, get, create, update, delete", input.Action)
	}
}

func routeVersion(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input VersionToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for version: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleVersionList(ctx, input)
	case "get":
		return handleVersionGet(ctx, input)
	case "create":
		return handleVersionCreate(ctx, input)
	case "publish":
		return handleVersionPublish(ctx, input)
	case "compare":
		return handleVersionCompare(ctx, input)
	case "find_by_date":
		return handleVersionFindByDate(ctx, input)
	case "set_latest":
		return handleVersionSetLatest(ctx, input)
	case "export":
		return handleVersionExport(ctx, input)
	case "import":
		return handleVersionImport(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource version — valid actions: list, get, create, publish, compare, find_by_date, set_latest, export, import", input.Action)
	}
}

func routeDestination(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input DestinationToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for destination: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleDestinationList(ctx, input)
	case "get":
		return handleDestinationGet(ctx, input)
	case "link":
		return handleDestinationLink(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource destination — valid actions: list, get, link", input.Action)
	}
}

func routeZone(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input ZoneToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for zone: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleZoneList(ctx, input)
	case "get":
		return handleZoneGet(ctx, input)
	case "create":
		return handleZoneCreate(ctx, input)
	case "update":
		return handleZoneUpdate(ctx, input)
	case "delete":
		return handleZoneDelete(ctx, input)
	case "revert":
		return handleZoneRevert(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource zone — valid actions: list, get, create, update, delete, revert", input.Action)
	}
}

func routeGtagConfig(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input GtagConfigToolInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for gtag_config: %w", err)
	}
	input.Action = gw.Action
	switch input.Action {
	case "list":
		return handleGtagConfigList(ctx, input)
	case "get":
		return handleGtagConfigGet(ctx, input)
	case "create":
		return handleGtagConfigCreate(ctx, input)
	case "update":
		return handleGtagConfigUpdate(ctx, input)
	case "delete":
		return handleGtagConfigDelete(ctx, input)
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource gtag_config — valid actions: list, get, create, update, delete", input.Action)
	}
}

func routeTemplatesRef(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	switch gw.Action {
	case "tag_templates":
		return handleGetTagTemplates()
	case "trigger_templates":
		return handleGetTriggerTemplates()
	default:
		return nil, nil, fmt.Errorf("unknown action %q for resource templates_ref — valid actions: tag_templates, trigger_templates", gw.Action)
	}
}

// --- Ping and Auth Status (utility resources) ---

// PingInput for the ping utility embedded in the gateway.
type PingInput struct {
	Message string `json:"message,omitempty"`
}

// PingOutput for the ping utility.
type PingOutput struct {
	Reply     string `json:"reply"`
	Timestamp string `json:"timestamp"`
}

func routePing(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	var input PingInput
	if err := unmarshalArgs(gw.Args, &input); err != nil {
		return nil, nil, fmt.Errorf("invalid args for ping: %w", err)
	}

	reply := "pong"
	if input.Message != "" {
		reply = fmt.Sprintf("pong: %s", input.Message)
	}
	return nil, PingOutput{Reply: reply, Timestamp: time.Now().UTC().Format(time.RFC3339)}, nil
}

// AuthStatusOutput for the auth_status utility.
type AuthStatusOutput struct {
	Authenticated bool   `json:"authenticated"`
	Message       string `json:"message"`
}

func routeAuthStatus(ctx context.Context, gw GatewayInput) (*mcp.CallToolResult, any, error) {
	tokenInfo := getTokenInfoFromContext(ctx)
	output := AuthStatusOutput{Authenticated: tokenInfo != nil}
	if tokenInfo != nil {
		output.Message = "You are authenticated and can access GTM data"
	} else {
		output.Message = "Not authenticated. GTM tools will require authentication."
	}
	return nil, output, nil
}

// getTokenInfoFromContext wraps the auth package to get token info without importing auth in gateway.
// This defers to the getClient mechanism which already handles auth internally.
func getTokenInfoFromContext(ctx context.Context) interface{} {
	// We use the auth package's GetTokenInfo via a thin wrapper to avoid import cycles.
	// The actual import is in tools.go which already imports auth.
	return getTokenInfo(ctx)
}
