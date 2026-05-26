package cmd

import (
	"flag"
	"fmt"

	di "github.com/asp24/gendi"
)

// PassConfig holds enabled/disabled pass configuration.
type PassConfig struct {
	Enabled  map[string]struct{} // Pass names to enable
	Disabled map[string]struct{} // Pass names to disable
}

// validate Returns an error if a name appears in both Enabled and Disabled, or if any
// name in Enabled or Disabled does not match a registered pass.
func (pc *PassConfig) validate(passes []di.SelectablePass) error {
	known := make(map[string]struct{}, len(passes))
	for _, p := range passes {
		known[p.Name()] = struct{}{}
	}

	for name := range pc.Enabled {
		if _, ok := known[name]; !ok {
			return fmt.Errorf("--enable-pass: unknown pass %q", name)
		}
	}

	for name := range pc.Disabled {
		if _, ok := known[name]; !ok {
			return fmt.Errorf("--disable-pass: unknown pass %q", name)
		}
	}

	for name := range pc.Enabled {
		if _, ok := pc.Disabled[name]; ok {
			return fmt.Errorf("pass %q is both enabled and disabled", name)
		}
	}

	return nil
}

// resolvePasses builds the final list of passes from the passed-in list,
// applying enable/disable filtering and always including enabled-by-default passes.
func (pc *PassConfig) resolvePasses(passes []di.SelectablePass) ([]di.Pass, error) {
	if err := pc.validate(passes); err != nil {
		return nil, err
	}

	result := make([]di.Pass, 0, len(passes))
	included := make(map[string]struct{}, len(passes))
	for _, p := range passes {
		name := p.Name()
		if _, ok := included[name]; ok {
			continue
		}

		_, enabled := pc.Enabled[name]
		if !p.RunByDefault() && !enabled {
			continue
		}

		if _, disabled := pc.Disabled[name]; disabled {
			continue
		}

		result = append(result, p)
		included[name] = struct{}{}
	}

	return result, nil
}

// RegisterFlags adds pass enable/disable flags to the flag set.
func (pc *PassConfig) RegisterFlags(flags *flag.FlagSet) {
	flags.Var(&stringSetFlag{values: &pc.Enabled}, "enable-pass", "Enable a specific compiler pass (can be specified multiple times)")
	flags.Var(&stringSetFlag{values: &pc.Disabled}, "disable-pass", "Disable a specific compiler pass (can be specified multiple times)")
}
