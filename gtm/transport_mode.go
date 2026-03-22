package gtm

import "context"

// TransportMode indicates how the MCP server is communicating with clients.
// Used to decide between local filesystem operations (stdio) and inline responses (HTTP).
type TransportMode string

const (
	// TransportStdio means the server runs locally via stdin/stdout.
	// File operations write to the user's local machine.
	TransportStdio TransportMode = "stdio"

	// TransportHTTP means the server runs remotely over HTTP/SSE.
	// File operations must return data inline since the filesystem is not the user's.
	TransportHTTP TransportMode = "http"
)

// transportModeKey is the context key for TransportMode.
type transportModeKey struct{}

// WithTransportMode returns a new context with the given TransportMode.
func WithTransportMode(ctx context.Context, mode TransportMode) context.Context {
	return context.WithValue(ctx, transportModeKey{}, mode)
}

// GetTransportMode returns the TransportMode from the context.
// Defaults to TransportStdio for backward compatibility.
func GetTransportMode(ctx context.Context) TransportMode {
	if mode, ok := ctx.Value(transportModeKey{}).(TransportMode); ok {
		return mode
	}
	return TransportStdio
}

// IsLocalMode returns true if the server can access the user's local filesystem.
func IsLocalMode(ctx context.Context) bool {
	return GetTransportMode(ctx) == TransportStdio
}
