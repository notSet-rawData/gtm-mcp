package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewAuditMiddleware(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}

			start := time.Now()

			toolName := ""
			if ctr, ok := req.(*mcp.CallToolRequest); ok {
				toolName = ctr.Params.Name
			}

			result, err := next(ctx, method, req)

			duration := time.Since(start)

			fields := []any{
				"audit", "tool_call",
				"tool", toolName,
				"duration_ms", duration.Milliseconds(),
			}

			if err != nil {
				fields = append(fields, "status", "error", "error", err.Error())
				logger.Warn("audit_tool_call_failed", fields...)
			} else {
				fields = append(fields, "status", "success")
				logger.Info("audit_tool_call", fields...)
			}

			return result, err
		}
	}
}
