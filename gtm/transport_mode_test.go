package gtm

import (
	"context"
	"testing"
)

func TestWithTransportMode_Stdio(t *testing.T) {
	ctx := WithTransportMode(context.Background(), TransportStdio)
	got := GetTransportMode(ctx)
	if got != TransportStdio {
		t.Errorf("GetTransportMode() = %q, want %q", got, TransportStdio)
	}
}

func TestWithTransportMode_HTTP(t *testing.T) {
	ctx := WithTransportMode(context.Background(), TransportHTTP)
	got := GetTransportMode(ctx)
	if got != TransportHTTP {
		t.Errorf("GetTransportMode() = %q, want %q", got, TransportHTTP)
	}
}

func TestGetTransportMode_DefaultIsStdio(t *testing.T) {
	ctx := context.Background()
	got := GetTransportMode(ctx)
	if got != TransportStdio {
		t.Errorf("GetTransportMode(empty ctx) = %q, want %q (default)", got, TransportStdio)
	}
}

func TestIsLocalMode(t *testing.T) {
	tests := []struct {
		name string
		mode TransportMode
		want bool
	}{
		{"stdio is local", TransportStdio, true},
		{"http is not local", TransportHTTP, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := WithTransportMode(context.Background(), tt.mode)
			if got := IsLocalMode(ctx); got != tt.want {
				t.Errorf("IsLocalMode() = %v, want %v", got, tt.want)
			}
		})
	}
}
