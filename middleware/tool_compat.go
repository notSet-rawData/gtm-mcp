package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type gatewayMapping struct {
	Resource string
	Action   string
}

var legacyToolMap = map[string]gatewayMapping{
	"list_accounts": {Resource: "account", Action: "list"},

	"list_containers":  {Resource: "container", Action: "list"},
	"create_container": {Resource: "container", Action: "create"},
	"delete_container": {Resource: "container", Action: "delete"},

	"list_workspaces":      {Resource: "workspace", Action: "list"},
	"create_workspace":     {Resource: "workspace", Action: "create"},
	"workspace_status":     {Resource: "workspace", Action: "status"},
	"get_workspace_status": {Resource: "workspace", Action: "status"},

	"list_tags":  {Resource: "tag", Action: "list"},
	"get_tag":    {Resource: "tag", Action: "get"},
	"create_tag": {Resource: "tag", Action: "create"},
	"update_tag": {Resource: "tag", Action: "update"},
	"delete_tag": {Resource: "tag", Action: "delete"},
	"revert_tag": {Resource: "tag", Action: "revert"},

	"list_triggers":  {Resource: "trigger", Action: "list"},
	"get_trigger":    {Resource: "trigger", Action: "get"},
	"create_trigger": {Resource: "trigger", Action: "create"},
	"update_trigger": {Resource: "trigger", Action: "update"},
	"delete_trigger": {Resource: "trigger", Action: "delete"},
	"revert_trigger": {Resource: "trigger", Action: "revert"},

	"list_variables":  {Resource: "variable", Action: "list"},
	"get_variable":    {Resource: "variable", Action: "get"},
	"create_variable": {Resource: "variable", Action: "create"},
	"update_variable": {Resource: "variable", Action: "update"},
	"delete_variable": {Resource: "variable", Action: "delete"},
	"revert_variable": {Resource: "variable", Action: "revert"},

	"list_folders":   {Resource: "folder", Action: "list"},
	"get_folder":     {Resource: "folder", Action: "get"},
	"create_folder":  {Resource: "folder", Action: "create"},
	"update_folder":  {Resource: "folder", Action: "update"},
	"delete_folder":  {Resource: "folder", Action: "delete"},
	"move_to_folder": {Resource: "folder", Action: "move"},
	"audit_folders":  {Resource: "folder", Action: "audit"},
	"revert_folder":  {Resource: "folder", Action: "revert"},

	"list_templates":  {Resource: "template", Action: "list"},
	"get_template":    {Resource: "template", Action: "get"},
	"create_template": {Resource: "template", Action: "create"},
	"update_template": {Resource: "template", Action: "update"},
	"delete_template": {Resource: "template", Action: "delete"},
	"import_template": {Resource: "template", Action: "import"},
	"revert_template": {Resource: "template", Action: "revert"},

	"list_built_in_variables":   {Resource: "built_in_variable", Action: "list"},
	"enable_built_in_variable":  {Resource: "built_in_variable", Action: "enable"},
	"disable_built_in_variable": {Resource: "built_in_variable", Action: "disable"},
	"revert_built_in_variable":  {Resource: "built_in_variable", Action: "revert"},

	"list_clients":  {Resource: "client", Action: "list"},
	"get_client":    {Resource: "client", Action: "get"},
	"create_client": {Resource: "client", Action: "create"},
	"update_client": {Resource: "client", Action: "update"},
	"delete_client": {Resource: "client", Action: "delete"},
	"revert_client": {Resource: "client", Action: "revert"},

	"list_transformations":  {Resource: "transformation", Action: "list"},
	"get_transformation":    {Resource: "transformation", Action: "get"},
	"create_transformation": {Resource: "transformation", Action: "create"},
	"update_transformation": {Resource: "transformation", Action: "update"},
	"delete_transformation": {Resource: "transformation", Action: "delete"},
	"revert_transformation": {Resource: "transformation", Action: "revert"},

	"list_environments":  {Resource: "environment", Action: "list"},
	"get_environment":    {Resource: "environment", Action: "get"},
	"create_environment": {Resource: "environment", Action: "create"},
	"update_environment": {Resource: "environment", Action: "update"},
	"delete_environment": {Resource: "environment", Action: "delete"},

	"list_user_permissions":  {Resource: "user_permission", Action: "list"},
	"get_user_permission":    {Resource: "user_permission", Action: "get"},
	"create_user_permission": {Resource: "user_permission", Action: "create"},
	"update_user_permission": {Resource: "user_permission", Action: "update"},
	"delete_user_permission": {Resource: "user_permission", Action: "delete"},

	"list_versions":        {Resource: "version", Action: "list"},
	"get_version":          {Resource: "version", Action: "get"},
	"create_version":       {Resource: "version", Action: "create"},
	"publish_version":      {Resource: "version", Action: "publish"},
	"compare_versions":     {Resource: "version", Action: "compare"},
	"find_version_by_date": {Resource: "version", Action: "find_by_date"},
	"set_latest_version":   {Resource: "version", Action: "set_latest"},
	"export_version":       {Resource: "version", Action: "export"},
	"export_container":     {Resource: "version", Action: "export"},

	"list_destinations": {Resource: "destination", Action: "list"},
	"get_destination":   {Resource: "destination", Action: "get"},
	"link_destination":  {Resource: "destination", Action: "link"},

	"list_zones":  {Resource: "zone", Action: "list"},
	"get_zone":    {Resource: "zone", Action: "get"},
	"create_zone": {Resource: "zone", Action: "create"},
	"update_zone": {Resource: "zone", Action: "update"},
	"delete_zone": {Resource: "zone", Action: "delete"},
	"revert_zone": {Resource: "zone", Action: "revert"},

	"list_gtag_configs":  {Resource: "gtag_config", Action: "list"},
	"get_gtag_config":    {Resource: "gtag_config", Action: "get"},
	"create_gtag_config": {Resource: "gtag_config", Action: "create"},
	"update_gtag_config": {Resource: "gtag_config", Action: "update"},
	"delete_gtag_config": {Resource: "gtag_config", Action: "delete"},

	"get_tag_templates":     {Resource: "templates_ref", Action: "tag_templates"},
	"get_trigger_templates": {Resource: "templates_ref", Action: "trigger_templates"},

	"ping":        {Resource: "ping", Action: ""},
	"auth_status": {Resource: "auth_status", Action: ""},
}

var consolidatedTools = map[string]bool{
	"account": true, "container": true, "workspace": true,
	"tag": true, "trigger": true, "variable": true,
	"folder": true, "template": true, "built_in_variable": true,
	"client": true, "transformation": true, "environment": true,
	"user_permission": true, "version": true, "destination": true,
	"zone": true, "gtag_config": true, "templates_ref": true,
}

var paramRenames = map[string]string{
	"version:versionIdA": "baseVersionId",
	"version:versionIdB": "targetVersionId",
}

func NewToolCompatMiddleware(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}

			ctr, ok := req.(*mcp.CallToolRequest)
			if !ok {
				return next(ctx, method, req)
			}

			if ctr.Params.Name == "gtm" {
				return next(ctx, method, req)
			}

			toolName := ctr.Params.Name
			mapping, found := legacyToolMap[toolName]
			if !found {
				normalized := strings.TrimPrefix(toolName, "gtm_")
				mapping, found = legacyToolMap[normalized]
			}

			if found {
				return remapToGatewayLegacy(ctx, method, req, ctr, mapping, logger, next)
			}

			if consolidatedTools[toolName] {
				return remapToGatewayConsolidated(ctx, method, req, ctr, toolName, logger, next)
			}

			return next(ctx, method, req)
		}
	}
}

func remapToGatewayLegacy(
	ctx context.Context, method string, req mcp.Request,
	ctr *mcp.CallToolRequest, mapping gatewayMapping,
	logger *slog.Logger, next mcp.MethodHandler,
) (mcp.Result, error) {
	logger.Info("tool_compat_remap_legacy",
		"from", ctr.Params.Name,
		"to_resource", mapping.Resource,
		"to_action", mapping.Action,
	)

	existingArgs := make(map[string]interface{})
	if len(ctr.Params.Arguments) > 0 {
		_ = json.Unmarshal(ctr.Params.Arguments, &existingArgs)
	}

	applyParamRenames(logger, mapping.Resource, existingArgs)

	delete(existingArgs, "action")

	argsJSON, _ := json.Marshal(existingArgs)
	gwArgs := map[string]interface{}{
		"resource": mapping.Resource,
		"action":   mapping.Action,
		"args":     json.RawMessage(argsJSON),
	}

	newArgs, _ := json.Marshal(gwArgs)
	ctr.Params.Name = "gtm"
	ctr.Params.Arguments = json.RawMessage(newArgs)

	return next(ctx, method, req)
}

func remapToGatewayConsolidated(
	ctx context.Context, method string, req mcp.Request,
	ctr *mcp.CallToolRequest, resource string,
	logger *slog.Logger, next mcp.MethodHandler,
) (mcp.Result, error) {
	existingArgs := make(map[string]interface{})
	if len(ctr.Params.Arguments) > 0 {
		_ = json.Unmarshal(ctr.Params.Arguments, &existingArgs)
	}

	action, _ := existingArgs["action"].(string)
	delete(existingArgs, "action")

	logger.Info("tool_compat_remap_consolidated",
		"from_tool", resource,
		"action", action,
	)

	applyParamRenames(logger, resource, existingArgs)

	argsJSON, _ := json.Marshal(existingArgs)
	gwArgs := map[string]interface{}{
		"resource": resource,
		"action":   action,
		"args":     json.RawMessage(argsJSON),
	}

	newArgs, _ := json.Marshal(gwArgs)
	ctr.Params.Name = "gtm"
	ctr.Params.Arguments = json.RawMessage(newArgs)

	return next(ctx, method, req)
}

func applyParamRenames(logger *slog.Logger, resource string, args map[string]interface{}) {
	for oldKey, val := range args {
		lookupKey := resource + ":" + oldKey
		if newKey, ok := paramRenames[lookupKey]; ok {
			logger.Info("param_compat_rename",
				"resource", resource,
				"from", oldKey,
				"to", newKey,
			)
			args[newKey] = val
			delete(args, oldKey)
		}
	}
}
