package stdlib

import (
	"bytes"
	"log/slog"
	"testing"
)

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
