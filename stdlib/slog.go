package stdlib

import (
	"io"
	"log/slog"
)

// NewSlogTextHandler creates a text handler that writes to the given writer.
//
// Example:
//
//	services:
//	  log_handler:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewSlogTextHandler"
//	      args:
//	        - "@log_output"
//	        - "%log_level%"
func NewSlogTextHandler(w io.Writer, level slog.Level) slog.Handler {
	return slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})
}

// NewSlogJSONHandler creates a JSON handler that writes to the given writer.
//
// Example:
//
//	services:
//	  log_handler:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewSlogJSONHandler"
//	      args:
//	        - "@log_output"
//	        - "%log_level%"
func NewSlogJSONHandler(w io.Writer, level slog.Level) slog.Handler {
	return slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
}

// NewSlogLogger creates a new slog.Logger with the given handler.
//
// Example:
//
//	services:
//	  log_handler:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewSlogJSONHandler"
//	      args:
//	        - "@log_output"
//	        - "%log_level%"
//
//	  logger:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewSlogLogger"
//	      args:
//	        - "@log_handler"
//	    public: true
func NewSlogLogger(handler slog.Handler) *slog.Logger {
	return slog.New(handler)
}
