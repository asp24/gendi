package generics

// NewChan creates a new buffered channel of type T.
func NewChan[T any](bufferSize int) chan T {
	return make(chan T, bufferSize)
}

// NewPool creates a new pool of type T.
func NewPool[T any](capacity int) *Pool[T] {
	return &Pool[T]{
		items:    make([]T, 0, capacity),
		capacity: capacity,
	}
}

// NewSlice creates a new slice of type T with given capacity.
func NewSlice[T any](capacity int) []T {
	return make([]T, 0, capacity)
}

// NewMap creates a new map with key type K and value type V.
func NewMap[K comparable, V any]() map[K]V {
	return make(map[K]V)
}

// Pool is a simple pool of items.
type Pool[T any] struct {
	items    []T
	capacity int
}

// Event is a test event type.
type Event struct {
	ID      int
	Payload string
}

// Message is a test message type.
type Message struct {
	From    string
	Content string
}
