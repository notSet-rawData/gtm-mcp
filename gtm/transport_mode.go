package gtm

import "context"

type TransportMode string

const (
	TransportStdio TransportMode = "stdio"

	TransportHTTP TransportMode = "http"
)

type transportModeKey struct{}

func WithTransportMode(ctx context.Context, mode TransportMode) context.Context {
	return context.WithValue(ctx, transportModeKey{}, mode)
}

func GetTransportMode(ctx context.Context) TransportMode {
	if mode, ok := ctx.Value(transportModeKey{}).(TransportMode); ok {
		return mode
	}
	return TransportStdio
}

func IsLocalMode(ctx context.Context) bool {
	return GetTransportMode(ctx) == TransportStdio
}
