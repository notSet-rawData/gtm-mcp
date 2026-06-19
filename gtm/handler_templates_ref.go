package gtm

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TemplatesRefToolInput struct {
	Action string `json:"action" jsonschema:"enum:tag_templates,trigger_templates,description:Which template reference to retrieve"`
}

func handleGetTagTemplates() (*mcp.CallToolResult, any, error) {
	templates := GetTagTemplates()
	return nil, GetTagTemplatesOutput{
		Templates: templates,
		Usage: `These templates show the correct parameter structure for creating GTM tags.

IMPORTANT - Common mistakes to avoid:
1. For GA4 Event tags (gaawe), use measurementIdOverride with an empty measurementId
2. Event parameters use name/value pairs in maps, NOT direct key names
3. For ecommerce, set sendEcommerceData=true and getEcommerceDataFrom=dataLayer

Copy the parameters JSON and modify values as needed when calling the tag tool with action: create.`,
	}, nil
}

func handleGetTriggerTemplates() (*mcp.CallToolResult, any, error) {
	templates := GetTriggerTemplates()
	return nil, GetTriggerTemplatesOutput{
		Templates: templates,
		Usage: `These templates show the correct structure for creating GTM triggers.

For customEvent triggers, use customEventFilterJson parameter.
For pageview triggers with conditions, use filterJson parameter.
For click/form triggers with conditions, use autoEventFilterJson parameter.`,
	}, nil
}
