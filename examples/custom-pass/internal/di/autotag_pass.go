package di

import (
	"strings"

	di "github.com/asp24/gendi"
)

// AutoTagPass automatically tags services based on naming conventions.
// Services ending with "Handler" get tagged as "http.handler"
// Services ending with "Repository" get tagged as "repository"
type AutoTagPass struct{}

func (p *AutoTagPass) Name() string {
	return "auto-tag"
}

func (p *AutoTagPass) Process(cfg *di.Config) (*di.Config, error) {
	for id, svc := range cfg.Services {
		// Auto-tag HTTP handlers
		if strings.HasSuffix(id, "Handler") || strings.HasSuffix(id, ".handler") {
			svc.Tags = append(svc.Tags, di.ServiceTag{
				Name: "http.handler",
				Attributes: map[string]interface{}{
					"auto_tagged": true,
				},
			})
			cfg.Services[id] = svc
		}

		// Auto-tag repositories
		if strings.HasSuffix(id, "Repository") || strings.HasSuffix(id, ".repo") {
			svc.Tags = append(svc.Tags, di.ServiceTag{
				Name: "repository",
				Attributes: map[string]interface{}{
					"auto_tagged": true,
				},
			})
			cfg.Services[id] = svc
		}
	}

	return cfg, nil
}
