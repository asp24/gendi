package cmd

import (
	"flag"

	di "github.com/asp24/gendi"
)

// PassConfig holds enabled/disabled pass configuration.
type PassConfig struct {
	Enabled  map[string]struct{} // Pass names to enable
	Disabled map[string]struct{} // Pass names to disable
}

// resolvePasses builds the final list of passes from the passed-in list,
// applying enable/disable filtering and always including enabled-by-default passes.
func (pc *PassConfig) resolvePasses(passes []di.OptionalPass) []di.Pass {
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

	return result
}

// RegisterFlags adds pass enable/disable flags to the flag set.
func (pc *PassConfig) RegisterFlags(flags *flag.FlagSet) {
	flags.Var(&stringSetFlag{values: &pc.Enabled}, "enable-pass", "Enable a specific compiler pass (can be specified multiple times)")
	flags.Var(&stringSetFlag{values: &pc.Disabled}, "disable-pass", "Disable a specific compiler pass (can be specified multiple times)")
}
