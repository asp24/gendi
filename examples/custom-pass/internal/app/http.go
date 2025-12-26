package app

import (
	"log/slog"
)

type HTTPHandler interface {
	Handle() string
}

type Server struct {
	logger   *slog.Logger
	handlers []HTTPHandler
}

func NewServer(logger *slog.Logger, handlers []HTTPHandler) *Server {
	return &Server{logger: logger, handlers: handlers}
}

func (s *Server) Start() {
	s.logger.Info("Starting server", slog.Int("handlers", len(s.handlers)))
	for i, h := range s.handlers {
		s.logger.Info(h.Handle(), slog.Int("handler", i))
	}
}
