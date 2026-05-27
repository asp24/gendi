package cmd

import (
	"flag"
	"fmt"

	di "github.com/asp24/gendi"
)

// PassConfig holds enabled pass configuration.
type PassConfig struct {
	Enabled map[string]struct{} // Pass names to enable
}

// validate Returns an error if any name in Enabled does not match a selectable pass.
func (pc *PassConfig) validate(selectablePasses []di.Pass) error {
	known := make(map[string]struct{}, len(selectablePasses))
	for _, p := range selectablePasses {
		known[p.Name()] = struct{}{}
	}

	for name := range pc.Enabled {
		if _, ok := known[name]; !ok {
			return fmt.Errorf("--enable-pass: unknown pass %q", name)
		}
	}

	return nil
}

// resolvePasses builds the final list of passes, applying enable filtering to
// selectable passes and deduplicating by pass name.
func (pc *PassConfig) resolvePasses(passes, selectablePasses []di.Pass) ([]di.Pass, error) {
	if err := pc.validate(selectablePasses); err != nil {
		return nil, err
	}

	result := make([]di.Pass, 0, len(passes)+len(selectablePasses))
	included := make(map[string]struct{}, len(passes)+len(selectablePasses))
	for _, p := range passes {
		name := p.Name()
		if _, ok := included[name]; ok {
			continue
		}

		result = append(result, p)
		included[name] = struct{}{}
	}

	for _, p := range selectablePasses {
		name := p.Name()
		if _, ok := included[name]; ok {
			continue
		}

		_, enabled := pc.Enabled[name]
		if !enabled {
			continue
		}

		result = append(result, p)
		included[name] = struct{}{}
	}

	return result, nil
}

// RegisterFlags adds pass enable flags to the flag set.
func (pc *PassConfig) RegisterFlags(flags *flag.FlagSet) {
	flags.Var(&stringSetFlag{values: &pc.Enabled}, "enable-pass", "Enable a specific compiler pass (can be specified multiple times)")
}
