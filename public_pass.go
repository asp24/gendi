package di

// ExposeAllPass is a test-only pass that promotes every service to public,
// causing the generator to emit a public getter for each one. Use it when
// building a test DI container that needs direct access to all services
// regardless of how they are declared in the YAML config.
//
// Enable via: --enable-pass=expose-all
//
// Not intended for production containers — it overrides explicit `public: false`
// declarations and disables unreachable-service pruning (all services become
// reachable roots), so every imported service gets a generated getter.
type ExposeAllPass struct {
}

func (p *ExposeAllPass) Name() string {
	return "expose-all"
}

func (p *ExposeAllPass) RunByDefault() bool {
	return false
}

func (p *ExposeAllPass) Process(cfg *Config) (*Config, error) {
	for id, svc := range cfg.Services {
		svc.Public = true
		cfg.Services[id] = svc
	}

	return cfg, nil
}
