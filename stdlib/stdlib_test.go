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
