package yaml

import (
	"strings"
	"testing"

	di "github.com/asp24/gendi"
)

func TestParseArgumentString(t *testing.T) {
	tests := []struct {
		in       string
		wantKind di.ArgumentKind
		wantVal  string
		wantErr  string
	}{
		{in: "@.inner", wantKind: di.ArgInner, wantVal: "@.inner"},
		{in: "@svc", wantKind: di.ArgServiceRef, wantVal: "svc"},
		{in: "%param%", wantKind: di.ArgParam, wantVal: "param"},
		{in: "!spread:@svc", wantKind: di.ArgSpread, wantVal: "@svc"},
		{in: "!tagged:tag", wantKind: di.ArgTagged, wantVal: "tag"},
		{in: "!go:os.Stdout", wantKind: di.ArgGoRef, wantVal: "os.Stdout"},
		{in: "!field:@cfg.Host", wantKind: di.ArgFieldAccessService, wantVal: "cfg.Host"},
		{in: "!field:!go:pkg.V.F", wantKind: di.ArgFieldAccessGo, wantVal: "pkg.V.F"},
		// Plain literals, including ones starting with '!'
		{in: "hello", wantKind: di.ArgLiteral},
		{in: "!important", wantKind: di.ArgLiteral},
		// Malformed reserved prefixes must be rejected, not treated as literals
		{in: "!field:config.Host", wantErr: "!field:"},
		{in: "!field:%p%.X", wantErr: "!field:"},
		{in: "!field:@", wantErr: "!field:"},
		{in: "!field:!go:", wantErr: "!field:"},
		{in: "!spread:", wantErr: "!spread:"},
		{in: "!tagged:", wantErr: "!tagged:"},
		{in: "!go:", wantErr: "!go:"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			kind, val, err := ParseArgumentString(tt.in)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got kind=%d err=%v", tt.wantErr, kind, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if kind != tt.wantKind {
				t.Errorf("kind = %d, want %d", kind, tt.wantKind)
			}
			if val != tt.wantVal {
				t.Errorf("value = %q, want %q", val, tt.wantVal)
			}
		})
	}
}
