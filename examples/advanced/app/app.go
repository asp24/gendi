package app

import (
	"fmt"
	"strings"
)

type Logger struct {
	Prefix string
}

func NewLogger(prefix string) *Logger {
	return &Logger{Prefix: prefix}
}

type DB struct {
	DSN string
}

func NewDB(dsn string) (*DB, error) {
	if !strings.HasPrefix(dsn, "postgres://") {
		return nil, fmt.Errorf("invalid dsn")
	}
	return &DB{DSN: dsn}, nil
}

type Handler struct {
	logger   *Logger
	db       *DB
	notifier Notifier
}

func NewHandler(logger *Logger, db *DB, notifier Notifier) *Handler {
	return &Handler{db: db, notifier: notifier, logger: logger}
}

type Factory struct {
	logger *Logger
}

func NewFactory(logger *Logger) *Factory {
	return &Factory{logger: logger}
}

func (f *Factory) NewHandler(db *DB, notifier Notifier) *Handler {
	return NewHandler(f.logger, db, notifier)
}
