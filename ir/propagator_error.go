package ir

// errorPropagator propagates error flags through the dependency graph
type errorPropagator struct{}

// propagate computes error propagation for all services
func (p *errorPropagator) propagate(ctx *buildContext) {
	// Initialize from constructors
	for _, svc := range ctx.services {
		if svc.Constructor != nil {
			svc.BuildCanError = svc.Constructor.ReturnsError
		}
	}

	// Propagate until stable
	changed := true
	for changed {
		changed = false

		// Build errors propagate from dependencies
		for _, svc := range ctx.services {
			if svc.BuildCanError {
				continue
			}
			for _, dep := range svc.Dependencies {
				if dep.CanError {
					svc.BuildCanError = true
					changed = true
					break
				}
			}
		}

		// Getter errors include decorator errors
		for _, svc := range ctx.services {
			newVal := svc.BuildCanError
			if len(svc.Decorators) > 0 {
				for _, dec := range svc.Decorators {
					if dec.BuildCanError {
						newVal = true
						break
					}
				}
			}
			if svc.CanError != newVal {
				svc.CanError = newVal
				changed = true
			}
		}
	}
}
