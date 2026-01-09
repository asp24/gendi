package generator

import (
	"testing"
)

func TestGetterRegistry_UniqueNameGeneration(t *testing.T) {
	gr := NewGetterRegistry(nil)

	used := map[string]bool{
		"getFoo": true,
	}

	// Should generate getFoo2 since getFoo is taken
	name := gr.uniqueName("getFoo", used)
	if name != "getFoo2" {
		t.Errorf("Expected 'getFoo2', got '%s'", name)
	}

	// Should return the base name if not used
	name2 := gr.uniqueName("getBar", used)
	if name2 != "getBar" {
		t.Errorf("Expected 'getBar', got '%s'", name2)
	}
}

func TestGetterRegistry_Assign_RealCollision_ServiceNames(t *testing.T) {
	ig := NewIdentGenerator()
	gr := NewGetterRegistry(ig)

	// These service names will generate the same getter: "GetGetService"
	// "getService" -> toCamel("getService") = "GetService" -> "GetGetService"
	// "get_service" -> toCamel("get_service") = "GetService" -> "GetGetService"
	services := map[string]*serviceDef{
		"getService": {
			id:     "getService",
			public: true,
		},
		"get_service": {
			id:     "get_service",
			public: true,
		},
	}

	orderedIDs := []string{"getService", "get_service"}
	err := gr.Assign(orderedIDs, services)

	if err == nil {
		t.Error("Expected error due to getter name collision between 'getService' and 'get_service'")
	} else {
		t.Logf("Got expected collision error: %v", err)
	}
}

func TestGetterRegistry_Assign_RealCollision_MustGetter(t *testing.T) {
	ig := NewIdentGenerator()
	gr := NewGetterRegistry(ig)

	// "service" will generate:
	//   - GetService (public getter)
	//   - MustService (Must* getter)
	// "mustService" will generate:
	//   - GetMustService (public getter)
	//   - MustMustService (Must* getter) - NO COLLISION with MustService
	// So this test actually won't have a collision with current logic

	// To create a real collision, we need a service whose Must* name
	// equals another service's public getter name
	// For example: "getMust" -> "GetGetMust" and "MustGetMust"
	//              "get_must" -> same names
	services := map[string]*serviceDef{
		"getMust": {
			id:     "getMust",
			public: true,
		},
		"get_must": {
			id:     "get_must",
			public: true,
		},
	}

	orderedIDs := []string{"getMust", "get_must"}
	err := gr.Assign(orderedIDs, services)

	if err == nil {
		t.Error("Expected error due to getter name collision")
	} else {
		t.Logf("Got expected collision error: %v", err)
	}
}

func TestGetterRegistry_Assign_NoCollision_PrivateServices(t *testing.T) {
	ig := NewIdentGenerator()
	gr := NewGetterRegistry(ig)

	// Private services with colliding names should be auto-numbered
	services := map[string]*serviceDef{
		"getService": {
			id:     "getService",
			public: false,
		},
		"get_service": {
			id:     "get_service",
			public: false,
		},
	}

	orderedIDs := []string{"getService", "get_service"}
	err := gr.Assign(orderedIDs, services)

	if err != nil {
		t.Fatalf("Private services should not error on collision, got: %v", err)
	}

	// Check that both got unique names via auto-numbering
	getter1 := gr.PrivateService("getService")
	getter2 := gr.PrivateService("get_service")

	if getter1 == getter2 {
		t.Errorf("Private services should have different getters, both got: %s", getter1)
	}

	t.Logf("getService -> %s", getter1)
	t.Logf("get_service -> %s", getter2)
}
