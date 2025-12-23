package di

// ConfigMerger handles merging DI configurations.
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
