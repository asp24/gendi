package yaml

import (
	"testing"

	di "github.com/asp24/gendi"
	"gopkg.in/yaml.v3"
)

func TestParseServiceAlias(t *testing.T) {
	raw := &RawService{
		Alias: "@foo",
	}
	p := NewParser()
	svc, err := p.convertService(raw, nil)
	if err != nil {
		t.Fatalf("convertService failed: %v", err)
	}
	if svc.Alias != "foo" {
		t.Errorf("expected alias 'foo', got '%s'", svc.Alias)
	}
}

func TestParseServiceAliasDirect(t *testing.T) {
	raw := &RawService{
		Alias: "foo", // direct ID, no @
	}
	p := NewParser()
	svc, err := p.convertService(raw, nil)
	if err != nil {
		t.Fatalf("convertService failed: %v", err)
	}
	if svc.Alias != "foo" {
		t.Errorf("expected alias 'foo', got '%s'", svc.Alias)
	}
}

func TestParseArgumentReference(t *testing.T) {
	val := "@myService"
	raw := &RawArgument{
		Value: &val,
	}
	p := NewParser()
	arg, err := p.convertArgument(raw)
	if err != nil {
		t.Fatalf("convertArgument failed: %v", err)
	}
	if arg.Kind != di.ArgServiceRef {
		t.Errorf("expected kind ArgServiceRef, got %v", arg.Kind)
	}
	if arg.Value != "myService" {
		t.Errorf("expected value 'myService', got '%s'", arg.Value)
	}
}

func TestParseArgumentLiteralString(t *testing.T) {
	val := "just a string"
	raw := &RawArgument{
		Value: &val,
	}
	p := NewParser()
	arg, err := p.convertArgument(raw)
	if err != nil {
		t.Fatalf("convertArgument failed: %v", err)
	}
	if arg.Kind != di.ArgLiteral {
		t.Errorf("expected kind ArgLiteral, got %v", arg.Kind)
	}
	if arg.Literal.String() != val {
		t.Errorf("expected literal value '%s', got '%s'", val, arg.Literal.String())
	}
}

func TestParseArgumentLiteralNode(t *testing.T) {
	node := yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!int",
		Value: "42",
	}
	raw := &RawArgument{
		Node: &node,
	}
	p := NewParser()
	arg, err := p.convertArgument(raw)
	if err != nil {
		t.Fatalf("convertArgument failed: %v", err)
	}
	if arg.Kind != di.ArgLiteral {
		t.Errorf("expected kind ArgLiteral, got %v", arg.Kind)
	}
	// Verify integer parsing logic if possible
}

func TestServiceDefaults(t *testing.T) {
	tests := []struct {
		name           string
		defaults       *ServiceDefaults
		service        *RawService
		expectedShared *bool
		expectedPublic bool
	}{
		{
			name: "no defaults",
			defaults: nil,
			service: &RawService{
				Type: "string",
			},
			expectedShared: nil,
			expectedPublic: false,
		},
		{
			name: "inherit shared from defaults",
			defaults: &ServiceDefaults{
				Shared: boolPtr(true),
			},
			service: &RawService{
				Type: "string",
			},
			expectedShared: boolPtr(true),
			expectedPublic: false,
		},
		{
			name: "inherit public from defaults",
			defaults: &ServiceDefaults{
				Public: boolPtr(true),
			},
			service: &RawService{
				Type: "string",
			},
			expectedShared: nil,
			expectedPublic: true,
		},
		{
			name: "override shared",
			defaults: &ServiceDefaults{
				Shared: boolPtr(true),
			},
			service: &RawService{
				Type:   "string",
				Shared: boolPtr(false),
			},
			expectedShared: boolPtr(false),
			expectedPublic: false,
		},
		{
			name: "override public",
			defaults: &ServiceDefaults{
				Public: boolPtr(true),
			},
			service: &RawService{
				Type:   "string",
				Public: boolPtr(false),
			},
			expectedShared: nil,
			expectedPublic: false,
		},
		{
			name: "inherit both from defaults",
			defaults: &ServiceDefaults{
				Shared: boolPtr(true),
				Public: boolPtr(true),
			},
			service: &RawService{
				Type: "string",
			},
			expectedShared: boolPtr(true),
			expectedPublic: true,
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := p.convertService(tt.service, tt.defaults)
			if err != nil {
				t.Fatalf("convertService failed: %v", err)
			}

			if !boolPtrEqual(svc.Shared, tt.expectedShared) {
				t.Errorf("expected shared=%v, got %v", boolPtrStr(tt.expectedShared), boolPtrStr(svc.Shared))
			}

			if svc.Public != tt.expectedPublic {
				t.Errorf("expected public=%v, got %v", tt.expectedPublic, svc.Public)
			}
		})
	}
}

func TestValidateDefaultsRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name        string
		service     *RawService
		expectError string
	}{
		{
			name: "type not allowed",
			service: &RawService{
				Type: "string",
			},
			expectError: "type",
		},
		{
			name: "constructor not allowed",
			service: &RawService{
				Constructor: RawConstructor{
					Func: "NewFoo",
				},
			},
			expectError: "constructor",
		},
		{
			name: "alias not allowed",
			service: &RawService{
				Alias: "@foo",
			},
			expectError: "alias",
		},
		{
			name: "decorates not allowed",
			service: &RawService{
				Decorates: "base",
			},
			expectError: "decorates",
		},
		{
			name: "decoration_priority not allowed",
			service: &RawService{
				DecorationPriority: 10,
			},
			expectError: "decoration_priority",
		},
		{
			name: "tags not allowed",
			service: &RawService{
				Tags: []rawServiceTag{{Name: "foo"}},
			},
			expectError: "tags",
		},
		{
			name: "only shared allowed",
			service: &RawService{
				Shared: boolPtr(true),
			},
			expectError: "",
		},
		{
			name: "only public allowed",
			service: &RawService{
				Public: boolPtr(true),
			},
			expectError: "",
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.validateDefaults(tt.service)
			if tt.expectError == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectError)
				} else if !contains(err.Error(), tt.expectError) {
					t.Errorf("expected error containing %q, got: %v", tt.expectError, err)
				}
			}
		})
	}
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func boolPtrStr(b *bool) string {
	if b == nil {
		return "nil"
	}
	if *b {
		return "true"
	}
	return "false"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestThisSubstitutionInFunc(t *testing.T) {
	raw := &RawService{
		Type: "string",
		Constructor: RawConstructor{
			Func: "$this.NewService",
		},
	}
	p := NewParser()
	svc, err := p.convertServiceWithPackage(raw, nil, "github.com/example/app")
	if err != nil {
		t.Fatalf("convertServiceWithPackage failed: %v", err)
	}

	expected := "github.com/example/app.NewService"
	if svc.Constructor.Func != expected {
		t.Errorf("expected constructor func '%s', got '%s'", expected, svc.Constructor.Func)
	}
}

func TestThisSubstitutionInMethod(t *testing.T) {
	raw := &RawService{
		Type: "string",
		Constructor: RawConstructor{
			Method: "$this.@service.Method",
		},
	}
	p := NewParser()
	svc, err := p.convertServiceWithPackage(raw, nil, "github.com/example/app")
	if err != nil {
		t.Fatalf("convertServiceWithPackage failed: %v", err)
	}

	expected := "github.com/example/app.@service.Method"
	if svc.Constructor.Method != expected {
		t.Errorf("expected constructor method '%s', got '%s'", expected, svc.Constructor.Method)
	}
}

func TestThisSubstitutionNoPackage(t *testing.T) {
	// When no package is provided, $this should remain unchanged
	raw := &RawService{
		Type: "string",
		Constructor: RawConstructor{
			Func: "$this.NewService",
		},
	}
	p := NewParser()
	svc, err := p.convertServiceWithPackage(raw, nil, "")
	if err != nil {
		t.Fatalf("convertServiceWithPackage failed: %v", err)
	}

	// Should remain unchanged when thisPackage is empty
	if svc.Constructor.Func != "$this.NewService" {
		t.Errorf("expected constructor func '$this.NewService', got '%s'", svc.Constructor.Func)
	}
}

func TestThisSubstitutionNotAtStart(t *testing.T) {
	// $this should only be substituted when at the start
	raw := &RawService{
		Type: "string",
		Constructor: RawConstructor{
			Func: "github.com/other/$this.NewService",
		},
	}
	p := NewParser()
	svc, err := p.convertServiceWithPackage(raw, nil, "github.com/example/app")
	if err != nil {
		t.Fatalf("convertServiceWithPackage failed: %v", err)
	}

	// Should remain unchanged since $this is not at the start
	expected := "github.com/other/$this.NewService"
	if svc.Constructor.Func != expected {
		t.Errorf("expected constructor func '%s', got '%s'", expected, svc.Constructor.Func)
	}
}

func TestThisSubstitutionNoConstructor(t *testing.T) {
	// Test with alias (no constructor)
	raw := &RawService{
		Alias: "@other",
	}
	p := NewParser()
	svc, err := p.convertServiceWithPackage(raw, nil, "github.com/example/app")
	if err != nil {
		t.Fatalf("convertServiceWithPackage failed: %v", err)
	}

	// Should not crash, constructor fields should be empty
	if svc.Constructor.Func != "" {
		t.Errorf("expected empty constructor func, got '%s'", svc.Constructor.Func)
	}
	if svc.Constructor.Method != "" {
		t.Errorf("expected empty constructor method, got '%s'", svc.Constructor.Method)
	}
}
