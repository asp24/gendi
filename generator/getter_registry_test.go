package generator

import (
	"go/types"
	"testing"
)

func TestGetterRegistryAssignRejectsNormalizedMethodCollision(t *testing.T) {
	services := map[string]*serviceDef{
		"foo-bar": {id: "foo-bar"},
		"foo.bar": {id: "foo.bar"},
	}

	for _, orderedIDs := range [][]string{
		{"foo-bar", "foo.bar"},
		{"foo.bar", "foo-bar"},
	} {
		registry := NewGetterRegistry(NewIdentGenerator())
		err := registry.Assign(orderedIDs, services)
		if err == nil {
			t.Fatal("expected normalized method collision")
		}

		want := `service identifiers "foo-bar" and "foo.bar" normalize to the same container method "getFooBar"`
		if err.Error() != want {
			t.Fatalf("unexpected error:\nwant: %s\n got: %s", want, err)
		}
	}
}

func TestGetterRegistryAssignRejectsInitFieldCollision(t *testing.T) {
	services := map[string]*serviceDef{
		"foo": {
			id:       "foo",
			typeName: types.Typ[types.Int],
			shared:   true,
		},
		"fooInit": {
			id:       "fooInit",
			typeName: types.NewPointer(types.Typ[types.Int]),
			shared:   true,
		},
	}

	registry := NewGetterRegistry(NewIdentGenerator())
	err := registry.Assign([]string{"fooInit", "foo"}, services)
	if err == nil {
		t.Fatal("expected container field collision")
	}

	want := `service identifiers "foo" and "fooInit" normalize to the same container field "svc_fooInit"`
	if err.Error() != want {
		t.Fatalf("unexpected error:\nwant: %s\n got: %s", want, err)
	}
}

func TestGetterRegistryAssignIgnoresFieldsThatAreNotGenerated(t *testing.T) {
	tests := []struct {
		name string
		foo  *serviceDef
	}{
		{
			name: "non-shared service",
			foo: &serviceDef{
				id:       "foo",
				typeName: types.Typ[types.Int],
			},
		},
		{
			name: "alias",
			foo: &serviceDef{
				id:          "foo",
				typeName:    types.Typ[types.Int],
				shared:      true,
				aliasTarget: "target",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			services := map[string]*serviceDef{
				"foo": tt.foo,
				"fooInit": {
					id:       "fooInit",
					typeName: types.NewPointer(types.Typ[types.Int]),
					shared:   true,
				},
			}

			registry := NewGetterRegistry(NewIdentGenerator())
			if err := registry.Assign([]string{"foo", "fooInit"}, services); err != nil {
				t.Fatalf("unexpected collision: %v", err)
			}
		})
	}
}

func TestGetterRegistryAssignStoresNormalizedGetterNames(t *testing.T) {
	services := map[string]*serviceDef{
		"private-service": {
			id: "private-service",
		},
		"public-service": {
			id:     "public-service",
			public: true,
		},
	}

	registry := NewGetterRegistry(NewIdentGenerator())
	if err := registry.Assign([]string{"public-service", "private-service"}, services); err != nil {
		t.Fatalf("assign names: %v", err)
	}

	checks := []struct {
		got  string
		want string
	}{
		{got: registry.PrivateService("private-service"), want: "getPrivateService"},
		{got: registry.PrivateService("public-service"), want: "getPublicService"},
		{got: registry.PublicService("public-service"), want: "GetPublicService"},
		{got: registry.MustService("public-service"), want: "MustPublicService"},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("got %q, want %q", check.got, check.want)
		}
	}
}
