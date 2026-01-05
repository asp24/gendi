package xmaps

import (
	"cmp"
	"slices"
)

func OrderedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	kSlice := make([]K, 0, len(m))
	for k := range m {
		kSlice = append(kSlice, k)
	}

	slices.Sort(kSlice)

	return kSlice
}
