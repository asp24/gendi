package stdlib

import "testing"

func TestNewChan(t *testing.T) {
	ch := NewChan[int](10)
	if cap(ch) != 10 {
		t.Errorf("NewChan capacity = %d, want 10", cap(ch))
	}

	// Test send/receive
	ch <- 42
	if v := <-ch; v != 42 {
		t.Errorf("received %d, want 42", v)
	}
}

func TestNewChanUnbuffered(t *testing.T) {
	ch := NewChan[string](0)
	if cap(ch) != 0 {
		t.Errorf("NewChan(0) capacity = %d, want 0", cap(ch))
	}
}
