package parameters

// ProviderComposite tries providers in order.
type ProviderComposite struct {
	providers []Provider
}

// NewProviderComposite returns a composite provider.
func NewProviderComposite(providers ...Provider) *ProviderComposite {
	return &ProviderComposite{providers: providers}
}

func (p *ProviderComposite) Has(name string) bool {
	for _, provider := range p.providers {
		if provider.Has(name) {
			return true
		}
	}

	return false
}

func (p *ProviderComposite) getLastProviderWhoHas(name string) Provider {
	for _, provider := range p.providers {
		if !provider.Has(name) {
			continue
		}

		return provider
	}

	return ProviderNullInstance
}

func (p *ProviderComposite) GetString(name string) (string, error) {
	return p.getLastProviderWhoHas(name).GetString(name)
}

func (p *ProviderComposite) GetInt(name string) (int, error) {
	return p.getLastProviderWhoHas(name).GetInt(name)
}

func (p *ProviderComposite) GetBool(name string) (bool, error) {
	return p.getLastProviderWhoHas(name).GetBool(name)
}

func (p *ProviderComposite) GetFloat(name string) (float64, error) {
	return p.getLastProviderWhoHas(name).GetFloat(name)
}
