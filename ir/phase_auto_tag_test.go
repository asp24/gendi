package ir

import (
	"go/types"
	"testing"

	di "github.com/asp24/gendi"
)

func TestAutoTagPhaseAddsTags(t *testing.T) {
	iface := types.NewInterfaceType(nil, nil)
	iface.Complete()

	tag := &Tag{
		Name:        "auto.tag",
		ElementType: iface,
	}

	container := NewContainer()
	container.tags["auto.tag"] = tag

	aliasTarget := &Service{ID: "svc.target", Type: types.Typ[types.Int], Autoconfigure: true}

	svcOne := &Service{ID: "svc.one", Type: types.Typ[types.String], Autoconfigure: true}
	svcTwo := &Service{ID: "svc.two", Type: types.Typ[types.Int], Autoconfigure: true, Tags: []*ServiceTag{{Tag: tag}}}
	svcInner := &Service{ID: "svc.inner", Type: types.Typ[types.Int], Autoconfigure: false}
	svcAlias := &Service{ID: "svc.alias", Type: types.Typ[types.Int], Alias: aliasTarget, Autoconfigure: true}
	svcOptOut := &Service{ID: "svc.optout", Type: types.Typ[types.String], Autoconfigure: false}

	container.Services = map[string]*Service{
		svcOne.ID:    svcOne,
		svcTwo.ID:    svcTwo,
		svcInner.ID:  svcInner,
		svcAlias.ID:  svcAlias,
		svcOptOut.ID: svcOptOut,
	}

	cfg := &di.Config{
		Tags: map[string]di.Tag{
			"auto.tag": {Autoconfigure: true},
		},
	}

	if err := (&autoTagPhase{}).Apply(cfg, container); err != nil {
		t.Fatalf("autoTagPhase failed: %v", err)
	}

	if !serviceHasTag(svcOne, tag) {
		t.Fatalf("expected service %q to be auto-tagged", svcOne.ID)
	}

	if countServiceTags(svcTwo, tag) != 1 {
		t.Fatalf("expected service %q to keep a single tag", svcTwo.ID)
	}

	if serviceHasTag(svcInner, tag) {
		t.Fatalf("did not expect inner service %q to be auto-tagged", svcInner.ID)
	}

	if serviceHasTag(svcAlias, tag) {
		t.Fatalf("did not expect alias service %q to be auto-tagged", svcAlias.ID)
	}

	if serviceHasTag(svcOptOut, tag) {
		t.Fatalf("did not expect opt-out service %q to be auto-tagged", svcOptOut.ID)
	}
}

func serviceHasTag(svc *Service, tag *Tag) bool {
	for _, st := range svc.Tags {
		if st.Tag == tag {
			return true
		}
	}
	return false
}

func countServiceTags(svc *Service, tag *Tag) int {
	count := 0
	for _, st := range svc.Tags {
		if st.Tag == tag {
			count++
		}
	}
	return count
}
