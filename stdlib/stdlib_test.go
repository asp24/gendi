package stdlib

import (
	"bytes"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"
)

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

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient(30 * time.Second)
	if client == nil {
		t.Fatal("NewHTTPClient returned nil")
	}
	if client.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", client.Timeout)
	}
}

func TestNewHTTPClientWithTransport(t *testing.T) {
	transport := &http.Transport{
		MaxIdleConns: 50,
	}
	client := NewHTTPClientWithTransport(15*time.Second, transport)
	if client == nil {
		t.Fatal("NewHTTPClientWithTransport returned nil")
	}
	if client.Timeout != 15*time.Second {
		t.Errorf("Timeout = %v, want 15s", client.Timeout)
	}
	if client.Transport != transport {
		t.Error("Transport not set correctly")
	}
}

func TestNewHTTPTransport(t *testing.T) {
	transport := NewHTTPTransport(100, 10, 90*time.Second)
	if transport == nil {
		t.Fatal("NewHTTPTransport returned nil")
	}
	if transport.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 10", transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", transport.IdleConnTimeout)
	}
}

func TestNewSlogTextHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := NewSlogTextHandler(&buf, slog.LevelInfo)
	if handler == nil {
		t.Fatal("NewSlogTextHandler returned nil")
	}

	logger := slog.New(handler)
	logger.Info("test message")

	if !bytes.Contains(buf.Bytes(), []byte("test message")) {
		t.Errorf("Log output should contain 'test message', got: %s", buf.String())
	}
}

func TestNewSlogJSONHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := NewSlogJSONHandler(&buf, slog.LevelInfo)
	if handler == nil {
		t.Fatal("NewSlogJSONHandler returned nil")
	}

	logger := slog.New(handler)
	logger.Info("test message")

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte(`"msg":"test message"`)) {
		t.Errorf("Log output should contain JSON msg field, got: %s", output)
	}
}

func TestNewSlogLogger(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	logger := NewSlogLogger(handler)
	if logger == nil {
		t.Fatal("NewSlogLogger returned nil")
	}

	logger.Info("hello")
	if !bytes.Contains(buf.Bytes(), []byte("hello")) {
		t.Errorf("Log output should contain 'hello', got: %s", buf.String())
	}
}

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

func TestMakeSlice(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := MakeSlice[int]()
		if len(result) != 0 {
			t.Errorf("MakeSlice() length = %d, want 0", len(result))
		}
		if result == nil {
			t.Error("MakeSlice() should return empty slice, not nil")
		}
	})

	t.Run("single_item", func(t *testing.T) {
		result := MakeSlice[string]("hello")
		if len(result) != 1 {
			t.Errorf("len(result) = %d, want 1", len(result))
		}
		if result[0] != "hello" {
			t.Errorf("result[0] = %q, want %q", result[0], "hello")
		}
	})

	t.Run("multiple_items", func(t *testing.T) {
		result := MakeSlice[int](1, 2, 3, 4, 5)
		if len(result) != 5 {
			t.Errorf("len(result) = %d, want 5", len(result))
		}
		for i, expected := range []int{1, 2, 3, 4, 5} {
			if result[i] != expected {
				t.Errorf("result[%d] = %d, want %d", i, result[i], expected)
			}
		}
	})

	t.Run("interface_types", func(t *testing.T) {
		// Using built-in interface to test generic behavior
		var items []interface{} = MakeSlice[interface{}]("hello", 42, true)
		if len(items) != 3 {
			t.Errorf("len(result) = %d, want 3", len(items))
		}
	})

	t.Run("pointer_types", func(t *testing.T) {
		type Service struct {
			ID int
		}

		s1 := &Service{ID: 1}
		s2 := &Service{ID: 2}
		result := MakeSlice[*Service](s1, s2)

		if len(result) != 2 {
			t.Errorf("len(result) = %d, want 2", len(result))
		}
		if result[0].ID != 1 {
			t.Errorf("result[0].ID = %d, want 1", result[0].ID)
		}
		if result[1].ID != 2 {
			t.Errorf("result[1].ID = %d, want 2", result[1].ID)
		}
	})
}
