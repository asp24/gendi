package cmd

import (
	"sort"
	"strings"
)

// stringSetFlag is a flag.Value that collects multiple values into a set.
type stringSetFlag struct {
	values *map[string]struct{}
}

func (f *stringSetFlag) String() string {
	if f.values == nil || *f.values == nil {
		return ""
	}
	values := make([]string, 0, len(*f.values))
	for value := range *f.values {
		values = append(values, value)
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func (f *stringSetFlag) Set(s string) error {
	if *f.values == nil {
		*f.values = make(map[string]struct{})
	}
	(*f.values)[s] = struct{}{}
	return nil
}
