package parameters

import (
	"errors"
	"fmt"
)

// ProviderComposite queries providers in reverse priority order (later
// providers override earlier ones): it continues past ErrParameterNotFound
// and stops on the first other result, so each parameter is looked up once.
type ProviderComposite struct {
	providers []Provider
}

// NewProviderComposite returns a composite provider.
func NewProviderComposite(providers ...Provider) *ProviderComposite {
	return &ProviderComposite{providers: providers}
}

func (p *ProviderComposite) Lookup(name string) (any, error) {
	for i := len(p.providers) - 1; i >= 0; i-- {
		val, err := p.providers[i].Lookup(name)
		if errors.Is(err, ErrParameterNotFound) {
			continue
		}
		return val, err
	}
	return nil, fmt.Errorf("parameter %q: %w", name, ErrParameterNotFound)
}
