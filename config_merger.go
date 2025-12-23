package di

// ConfigMerger handles merging DI configurations and applying prefixes.
type ConfigMerger struct{}

// NewConfigMerger creates a new config merger.
func NewConfigMerger() *ConfigMerger {
	return &ConfigMerger{}
}

// Merge merges src config into dst config.
// Returns the merged dst config.
func (m *ConfigMerger) Merge(dst, src *Config) *Config {
	if dst.Parameters == nil {
		dst.Parameters = map[string]Parameter{}
	}
	if dst.Tags == nil {
		dst.Tags = map[string]Tag{}
	}
	if dst.Services == nil {
		dst.Services = map[string]*Service{}
	}

	for k, v := range src.Parameters {
		dst.Parameters[k] = v
	}
	for k, v := range src.Tags {
		dst.Tags[k] = v
	}
	for k, v := range src.Services {
		copySvc := *v
		dst.Services[k] = &copySvc
	}
	return dst
}

// ApplyPrefix applies a prefix to all service names and internal references in cfg.
// This is useful when importing configs with a namespace.
func (m *ConfigMerger) ApplyPrefix(cfg *Config, prefix string) {
	if prefix == "" || len(cfg.Services) == 0 {
		return
	}

	// Track original service names to know which refs to prefix
	original := map[string]bool{}
	for name := range cfg.Services {
		original[name] = true
	}

	// Update internal references
	for _, svc := range cfg.Services {
		if svc.Decorates != "" && original[svc.Decorates] {
			svc.Decorates = prefix + svc.Decorates
		}
		if svc.Alias != "" && original[svc.Alias] {
			svc.Alias = prefix + svc.Alias
		}
		for i := range svc.Constructor.Args {
			arg := &svc.Constructor.Args[i]
			if arg.Kind == ArgServiceRef && original[arg.Value] {
				arg.Value = prefix + arg.Value
			}
		}
	}

	// Apply prefix to service names
	prefixed := map[string]*Service{}
	for name, svc := range cfg.Services {
		prefixed[prefix+name] = svc
	}
	cfg.Services = prefixed
}
