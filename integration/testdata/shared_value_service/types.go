package main

// buildCount tracks how many times NewCounter runs, so the test can prove the
// shared value-type getter builds its service exactly once and caches it.
var buildCount int

// Counter is a value type (non-nilable), so its shared getter is rendered by
// sharedValueGetterRenderer, which caches via a separate Init flag.
type Counter struct {
	ID int
}

func NewCounter() Counter {
	buildCount++
	return Counter{ID: buildCount}
}

func BuildCount() int {
	return buildCount
}
