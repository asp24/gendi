package stdlib

// MakeSlice creates a slice from variadic arguments.
// This is a utility function used by the generated DI container code
// to create tagged service collections.
//
// Example usage:
//
//	items := MakeSlice[Notifier](notifier1, notifier2, notifier3)
//	// Returns: []Notifier{notifier1, notifier2, notifier3}
func MakeSlice[T any](items ...T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
