package yaml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	di "github.com/asp24/gendi"
)

func TestParseServiceAlias(t *testing.T) {
	raw := &RawService{
		Alias: "@foo",
	}
	p := NewParser()
	svc, err := p.convertServiceWithPackageAndFile(raw, nil, "", "")
	if err != nil {
		t.Fatalf("convertServiceWithPackageAndFile failed: %v", err)
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
	svc, err := p.convertServiceWithPackageAndFile(raw, nil, "", "")
	if err != nil {
		t.Fatalf("convertServiceWithPackageAndFile failed: %v", err)
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
	arg, err := p.convertArgumentWithFile(raw, "")
	if err != nil {
		t.Fatalf("convertArgumentWithFile failed: %v", err)
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
	arg, err := p.convertArgumentWithFile(raw, "")
	if err != nil {
		t.Fatalf("convertArgumentWithFile failed: %v", err)
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
	arg, err := p.convertArgumentWithFile(raw, "")
	if err != nil {
		t.Fatalf("convertArgumentWithFile failed: %v", err)
	}
	if arg.Kind != di.ArgLiteral {
		t.Errorf("expected kind ArgLiteral, got %v", arg.Kind)
	}
	// Verify integer parsing logic if possible
}

func TestServiceDefaults(t *testing.T) {
	tests := []struct {
		name                  string
		defaults              *ServiceDefaults
		service               *RawService
		expectedShared        *bool
		expectedPublic        bool
		expectedAutoconfigure bool
	}{
		{
			name:     "no defaults",
			defaults: nil,
			service: &RawService{
				Type: "string",
			},
			expectedShared:        nil,
			expectedPublic:        false,
			expectedAutoconfigure: true,
		},
		{
			name: "inherit shared from defaults",
			defaults: &ServiceDefaults{
				Shared: boolPtr(true),
			},
			service: &RawService{
				Type: "string",
			},
			expectedShared:        boolPtr(true),
			expectedPublic:        false,
			expectedAutoconfigure: true,
		},
		{
			name: "inherit public from defaults",
			defaults: &ServiceDefaults{
				Public: boolPtr(true),
			},
			service: &RawService{
				Type: "string",
			},
			expectedShared:        nil,
			expectedPublic:        true,
			expectedAutoconfigure: true,
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
			expectedShared:        boolPtr(false),
			expectedPublic:        false,
			expectedAutoconfigure: true,
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
			expectedShared:        nil,
			expectedPublic:        false,
			expectedAutoconfigure: true,
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
			expectedShared:        boolPtr(true),
			expectedPublic:        true,
			expectedAutoconfigure: true,
		},
		{
			name: "inherit autoconfigure from defaults",
			defaults: &ServiceDefaults{
				Autoconfigure: boolPtr(false),
			},
			service: &RawService{
				Type: "string",
			},
			expectedShared:        nil,
			expectedPublic:        false,
			expectedAutoconfigure: false,
		},
		{
			name: "override autoconfigure",
			defaults: &ServiceDefaults{
				Autoconfigure: boolPtr(false),
			},
			service: &RawService{
				Type:          "string",
				Autoconfigure: boolPtr(true),
			},
			expectedShared:        nil,
			expectedPublic:        false,
			expectedAutoconfigure: true,
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := p.convertServiceWithPackageAndFile(tt.service, tt.defaults, "", "")
			if err != nil {
				t.Fatalf("convertServiceWithPackageAndFile failed: %v", err)
			}

			if svc.Shared != resolveBoolPtr(tt.expectedShared) {
				t.Errorf("expected shared=%v, got %v", resolveBoolPtr(tt.expectedShared), svc.Shared)
			}

			if svc.Public != tt.expectedPublic {
				t.Errorf("expected public=%v, got %v", tt.expectedPublic, svc.Public)
			}

			if svc.Autoconfigure != tt.expectedAutoconfigure {
				t.Errorf("expected autoconfigure=%v, got %v", tt.expectedAutoconfigure, svc.Autoconfigure)
			}
		})
	}
}

func TestServiceTagFlattened(t *testing.T) {
	// Test new syntax where attributes are at the same level as name
	raw := &RawService{
		Type: "string",
		Tags: []RawServiceTag{
			{
				Name: "test.tag",
				Attributes: map[string]interface{}{
					"priority": 10,
					"enabled":  true,
				},
			},
		},
	}

	p := NewParser()
	svc, err := p.convertServiceWithPackageAndFile(raw, nil, "", "")
	if err != nil {
		t.Fatalf("convertServiceWithPackageAndFile failed: %v", err)
	}

	if len(svc.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(svc.Tags))
	}

	tag := svc.Tags[0]
	if tag.Name != "test.tag" {
		t.Errorf("expected tag name 'test.tag', got '%s'", tag.Name)
	}

	if priority, ok := tag.Attributes["priority"].(int); !ok || priority != 10 {
		t.Errorf("expected priority=10, got %v", tag.Attributes["priority"])
	}

	if enabled, ok := tag.Attributes["enabled"].(bool); !ok || !enabled {
		t.Errorf("expected enabled=true, got %v", tag.Attributes["enabled"])
	}
}

func TestServiceTagOnlyName(t *testing.T) {
	// Test tag with only name (no attributes)
	raw := &RawService{
		Type: "string",
		Tags: []RawServiceTag{
			{
				Name:       "marker.tag",
				Attributes: map[string]interface{}{},
			},
		},
	}

	p := NewParser()
	svc, err := p.convertServiceWithPackageAndFile(raw, nil, "", "")
	if err != nil {
		t.Fatalf("convertServiceWithPackageAndFile failed: %v", err)
	}

	if len(svc.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(svc.Tags))
	}

	tag := svc.Tags[0]
	if tag.Name != "marker.tag" {
		t.Errorf("expected tag name 'marker.tag', got '%s'", tag.Name)
	}

	if len(tag.Attributes) != 0 {
		t.Errorf("expected no attributes, got %v", tag.Attributes)
	}
}

func TestServiceTagYAMLParsing(t *testing.T) {
	// Test that YAML parsing works with the new flattened syntax
	yamlContent := `
services:
  test.service:
    type: string
    tags:
      - name: handler.http
        priority: 10
        path: /api/test
      - name: marker.tag
`

	var raw RawConfig
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	svc, ok := raw.Services["test.service"]
	if !ok {
		t.Fatal("service 'test.service' not found")
	}

	if len(svc.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(svc.Tags))
	}

	// First tag: handler.http with attributes
	tag1 := svc.Tags[0]
	if tag1.Name != "handler.http" {
		t.Errorf("expected tag name 'handler.http', got '%s'", tag1.Name)
	}
	if priority, ok := tag1.Attributes["priority"].(int); !ok || priority != 10 {
		t.Errorf("expected priority=10, got %v", tag1.Attributes["priority"])
	}
	if path, ok := tag1.Attributes["path"].(string); !ok || path != "/api/test" {
		t.Errorf("expected path='/api/test', got %v", tag1.Attributes["path"])
	}

	// Second tag: marker.tag with no attributes
	tag2 := svc.Tags[1]
	if tag2.Name != "marker.tag" {
		t.Errorf("expected tag name 'marker.tag', got '%s'", tag2.Name)
	}
	if len(tag2.Attributes) != 0 {
		t.Errorf("expected no attributes for marker tag, got %v", tag2.Attributes)
	}
}

func TestServiceTagStringShorthand(t *testing.T) {
	yamlContent := `
services:
  test.service:
    type: string
    tags:
      - marker.tag
      - name: handler.http
        priority: 10
`

	var raw RawConfig
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		t.Fatalf("failed to unmarshal YAML: %v", err)
	}

	svc, ok := raw.Services["test.service"]
	if !ok {
		t.Fatal("service 'test.service' not found")
	}

	if len(svc.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(svc.Tags))
	}

	// First tag: string shorthand
	tag1 := svc.Tags[0]
	if tag1.Name != "marker.tag" {
		t.Errorf("expected tag name 'marker.tag', got '%s'", tag1.Name)
	}
	if len(tag1.Attributes) != 0 {
		t.Errorf("expected no attributes, got %v", tag1.Attributes)
	}

	// Second tag: map syntax
	tag2 := svc.Tags[1]
	if tag2.Name != "handler.http" {
		t.Errorf("expected tag name 'handler.http', got '%s'", tag2.Name)
	}
	if priority, ok := tag2.Attributes["priority"].(int); !ok || priority != 10 {
		t.Errorf("expected priority=10, got %v", tag2.Attributes["priority"])
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
				Tags: []RawServiceTag{{Name: "foo"}},
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
				} else if !strings.Contains(err.Error(), tt.expectError) {
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

func resolveBoolPtr(b *bool) bool {
	if b == nil {
		return true // Default shared is true
	}
	return *b
}

func TestThisSubstitutionInConstructor(t *testing.T) {
	tests := []struct {
		name       string
		raw        *RawService
		thisPkg    string
		wantFunc   string
		wantMethod string
	}{
		{
			name:     "func substituted",
			raw:      &RawService{Type: "string", Constructor: RawConstructor{Func: "$this.NewService"}},
			thisPkg:  "github.com/example/app",
			wantFunc: "github.com/example/app.NewService",
		},
		{
			name:       "method substituted",
			raw:        &RawService{Type: "string", Constructor: RawConstructor{Method: "$this.@service.Method"}},
			thisPkg:    "github.com/example/app",
			wantMethod: "github.com/example/app.@service.Method",
		},
		{
			name:     "no package leaves $this unchanged",
			raw:      &RawService{Type: "string", Constructor: RawConstructor{Func: "$this.NewService"}},
			thisPkg:  "",
			wantFunc: "$this.NewService",
		},
		{
			name:     "$this not at start stays unchanged",
			raw:      &RawService{Type: "string", Constructor: RawConstructor{Func: "github.com/other/$this.NewService"}},
			thisPkg:  "github.com/example/app",
			wantFunc: "github.com/other/$this.NewService",
		},
		{
			name:    "alias has no constructor",
			raw:     &RawService{Alias: "@other"},
			thisPkg: "github.com/example/app",
		},
		{
			name:     "both type and func substituted",
			raw:      &RawService{Type: "$this.Logger", Constructor: RawConstructor{Func: "$this.NewLogger"}},
			thisPkg:  "github.com/example/app",
			wantFunc: "github.com/example/app.NewLogger",
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := p.convertServiceWithPackageAndFile(tt.raw, nil, tt.thisPkg, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantFunc != "" && svc.Constructor.Func != tt.wantFunc {
				t.Errorf("Func = %q, want %q", svc.Constructor.Func, tt.wantFunc)
			}
			if tt.wantMethod != "" && svc.Constructor.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", svc.Constructor.Method, tt.wantMethod)
			}
		})
	}
}

func TestThisSubstitutionInType(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		wantType string
	}{
		{"plain", "$this.Logger", "github.com/example/app.Logger"},
		{"pointer", "*$this.Logger", "*github.com/example/app.Logger"},
		{"slice", "[]$this.Logger", "[]github.com/example/app.Logger"},
		{"map value", "map[string]$this.Logger", "map[string]github.com/example/app.Logger"},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := &RawService{
				Type:        tt.typ,
				Constructor: RawConstructor{Func: "pkg.New"},
			}
			svc, err := p.convertServiceWithPackageAndFile(raw, nil, "github.com/example/app", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if svc.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", svc.Type, tt.wantType)
			}
		})
	}
}

func TestThisSubstitutionInTagElementType(t *testing.T) {
	tests := []struct {
		name        string
		elementType string
		configDir   string
		want        string
	}{
		{"plain", "$this.Notifier", "WITH_MOD", "github.com/example/app.Notifier"},
		{"pointer", "*$this.Handler", "WITH_MOD", "*github.com/example/app.Handler"},
		{"slice", "[]$this.Middleware", "WITH_MOD", "[]github.com/example/app.Middleware"},
		{"no package", "$this.Notifier", "", "$this.Notifier"},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir := tt.configDir
			if configDir == "WITH_MOD" {
				tempDir := t.TempDir()
				modFile := filepath.Join(tempDir, "go.mod")
				if err := os.WriteFile(modFile, []byte("module github.com/example/app\n\ngo 1.21\n"), 0o644); err != nil {
					t.Fatalf("failed to write go.mod: %v", err)
				}
				configDir = tempDir
			}

			raw := &RawConfig{
				Tags: map[string]RawTag{
					"test.tag": {ElementType: tt.elementType},
				},
			}
			cfg, err := p.ConvertConfigWithDirAndFile(raw, configDir, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tag, ok := cfg.Tags["test.tag"]
			if !ok {
				t.Fatal("tag not found")
			}
			if tag.ElementType != tt.want {
				t.Errorf("ElementType = %q, want %q", tag.ElementType, tt.want)
			}
		})
	}
}

func TestConvertLiteralTypes(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name    string
		node    yaml.Node
		wantErr string
	}{
		{
			name: "float",
			node: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: "3.14"},
		},
		{
			name: "bool",
			node: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"},
		},
		{
			name: "null",
			node: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"},
		},
		{
			name:    "unsupported",
			node:    yaml.Node{Kind: yaml.ScalarNode, Tag: "!!binary", Value: "data"},
			wantErr: "unsupported literal type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.convertLiteral(&tt.node)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestConvertConfigWithDirAndFile(t *testing.T) {
	p := NewParser()

	t.Run("parameter_missing_type", func(t *testing.T) {
		raw := &RawConfig{
			Parameters: map[string]RawParameter{
				"bad": {Value: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x"}},
			},
		}
		_, err := p.ConvertConfigWithDirAndFile(raw, "", "")
		if err == nil || !strings.Contains(err.Error(), "type is required") {
			t.Fatalf("expected 'type is required' error, got: %v", err)
		}
	})

	t.Run("parameter_bad_literal", func(t *testing.T) {
		raw := &RawConfig{
			Parameters: map[string]RawParameter{
				"bad": {Type: "int", Value: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!binary", Value: "x"}},
			},
		}
		_, err := p.ConvertConfigWithDirAndFile(raw, "", "")
		if err == nil {
			t.Fatal("expected error for bad literal type")
		}
	})

	t.Run("parameter_ok", func(t *testing.T) {
		raw := &RawConfig{
			Parameters: map[string]RawParameter{
				"host": {Type: "string", Value: yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "localhost"}},
			},
		}
		cfg, err := p.ConvertConfigWithDirAndFile(raw, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Parameters["host"].Type != "string" {
			t.Fatal("expected parameter 'host' with type string")
		}
	})

	t.Run("default_applied_to_services", func(t *testing.T) {
		raw := &RawConfig{
			Services: map[string]*RawService{
				"_default": {Shared: boolPtr(false)},
				"svc": {
					Constructor: RawConstructor{Func: "pkg.New"},
				},
			},
		}
		cfg, err := p.ConvertConfigWithDirAndFile(raw, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Services["svc"].Shared != false {
			t.Fatal("expected shared=false from _default")
		}
		if _, ok := cfg.Services["_default"]; ok {
			t.Fatal("_default should not appear in output services")
		}
	})

	t.Run("default_invalid", func(t *testing.T) {
		raw := &RawConfig{
			Services: map[string]*RawService{
				"_default": {Type: "bad"},
			},
		}
		_, err := p.ConvertConfigWithDirAndFile(raw, "", "")
		if err == nil || !strings.Contains(err.Error(), "_default") {
			t.Fatalf("expected _default error, got: %v", err)
		}
	})

	t.Run("service_convert_error", func(t *testing.T) {
		badNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!binary", Value: "x"}
		raw := &RawConfig{
			Services: map[string]*RawService{
				"bad": {
					Constructor: RawConstructor{
						Args: []RawArgument{{Node: badNode}},
					},
				},
			},
		}
		_, err := p.ConvertConfigWithDirAndFile(raw, "", "")
		if err == nil {
			t.Fatal("expected service conversion error")
		}
	})
}

func TestConvertArgumentEmpty(t *testing.T) {
	p := NewParser()
	_, err := p.convertArgumentWithFile(&RawArgument{}, "")
	if err == nil || !strings.Contains(err.Error(), "must have a value") {
		t.Fatalf("expected 'must have a value' error, got: %v", err)
	}
}

func TestThisSubstitutionInGoAndFieldArgs(t *testing.T) {
	p := NewParser()

	t.Run("go_ref_this", func(t *testing.T) {
		goVal := "!go:$this.DefaultLevel"
		raw := &RawService{
			Constructor: RawConstructor{
				Func: "pkg.New",
				Args: []RawArgument{{Value: &goVal}},
			},
		}
		svc, err := p.convertServiceWithPackageAndFile(raw, nil, "github.com/app", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svc.Constructor.Args[0].Value != "github.com/app.DefaultLevel" {
			t.Fatalf("expected substituted go ref, got: %s", svc.Constructor.Args[0].Value)
		}
	})

	t.Run("field_go_ref_this", func(t *testing.T) {
		fieldVal := "!field:!go:$this.DefaultCfg.Host"
		raw := &RawService{
			Constructor: RawConstructor{
				Func: "pkg.New",
				Args: []RawArgument{{Value: &fieldVal}},
			},
		}
		svc, err := p.convertServiceWithPackageAndFile(raw, nil, "github.com/app", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svc.Constructor.Args[0].Kind != di.ArgFieldAccessGo {
			t.Fatalf("expected ArgFieldAccessGo, got: %d", svc.Constructor.Args[0].Kind)
		}
		expected := "github.com/app.DefaultCfg.Host"
		if svc.Constructor.Args[0].Value != expected {
			t.Fatalf("expected %q, got: %q", expected, svc.Constructor.Args[0].Value)
		}
	})
}

func TestTagAutoconfigureParsed(t *testing.T) {
	raw := &RawConfig{
		Tags: map[string]RawTag{
			"auto.tag": {
				ElementType:   "string",
				Autoconfigure: true,
			},
		},
	}
	p := NewParser()
	cfg, err := p.ConvertConfigWithDirAndFile(raw, "", "")
	if err != nil {
		t.Fatalf("convertConfigWithDir failed: %v", err)
	}

	tag, ok := cfg.Tags["auto.tag"]
	if !ok {
		t.Fatal("tag 'auto.tag' not found")
	}

	if !tag.Autoconfigure {
		t.Fatal("expected tag 'auto.tag' to have autoconfigure enabled")
	}
}
