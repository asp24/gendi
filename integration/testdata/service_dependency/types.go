package main

import "fmt"

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Log(msg string) {
	fmt.Println("[LOG]", msg)
}

type Service struct {
	logger *Logger
}

func NewService(logger *Logger) *Service {
	return &Service{logger: logger}
}

func (s *Service) Run() {
	s.logger.Log("Service running")
}
