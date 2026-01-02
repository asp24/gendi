package ir

import "testing"

func TestPruneUnreachableRemovesServices(t *testing.T) {
	publicSvc := &Service{ID: "public", Public: true}
	privateSvc := &Service{ID: "private"}
	sharedDep := &Service{ID: "dep"}

	publicSvc.Dependencies = []*Service{sharedDep}

	ctx := &buildContext{
		services: map[string]*Service{
			publicSvc.ID:  publicSvc,
			privateSvc.ID: privateSvc,
			sharedDep.ID:  sharedDep,
		},
		tags: map[string]*Tag{},
		order: []string{
			privateSvc.ID,
			publicSvc.ID,
			sharedDep.ID,
		},
	}

	pruneUnreachable(ctx)

	if _, ok := ctx.services[publicSvc.ID]; !ok {
		t.Fatalf("expected public service to remain")
	}
	if _, ok := ctx.services[sharedDep.ID]; !ok {
		t.Fatalf("expected dependency to remain")
	}
	if _, ok := ctx.services[privateSvc.ID]; ok {
		t.Fatalf("expected private service to be removed")
	}
	if len(ctx.order) != 2 {
		t.Fatalf("expected 2 services after prune, got %d", len(ctx.order))
	}
}

func TestPruneUnreachableFiltersTagServices(t *testing.T) {
	publicSvc := &Service{ID: "public", Public: true}
	privateSvc := &Service{ID: "private"}
	tag := &Tag{
		Name:     "t",
		Public:   true,
		Services: []*Service{privateSvc},
	}

	ctx := &buildContext{
		services: map[string]*Service{
			publicSvc.ID:  publicSvc,
			privateSvc.ID: privateSvc,
		},
		tags: map[string]*Tag{
			tag.Name: tag,
		},
		order: []string{publicSvc.ID, privateSvc.ID},
	}

	pruneUnreachable(ctx)

	if len(tag.Services) != 1 || tag.Services[0].ID != privateSvc.ID {
		t.Fatalf("expected tag services to be filtered to reachable ones")
	}
}
