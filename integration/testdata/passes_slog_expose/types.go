package main

type Logger struct{ channel string }

func NewLogger() *Logger { return &Logger{} }

func (l *Logger) With(key, val string) *Logger { return &Logger{channel: val} }

type Worker struct{ log *Logger }

func NewWorker(log *Logger) *Worker { return &Worker{log: log} }

func (w *Worker) Channel() string { return w.log.channel }
