package stdlib

import (
	"os"
	"testing"
)

func TestNewStdout(t *testing.T) {
	w := NewStdout()
	if w != os.Stdout {
		t.Error("NewStdout should return os.Stdout")
	}
}

func TestNewStderr(t *testing.T) {
	w := NewStderr()
	if w != os.Stderr {
		t.Error("NewStderr should return os.Stderr")
	}
}
