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

type Mailer struct {
	Host    string
	Retries int
	Prefix  string
}

func NewMailer(host string) *Mailer {
	return &Mailer{Host: host}
}

func AddRetry(inner *Mailer, retries int) *Mailer {
	inner.Retries = retries
	return inner
}

func AddPrefix(inner *Mailer, prefix string) *Mailer {
	inner.Prefix = prefix
	return inner
}

type Handler struct {
	DB     *DB
	Mailer *Mailer
	Logger *Logger
}

func NewHandler(db *DB, mailer *Mailer, logger *Logger) *Handler {
	return &Handler{DB: db, Mailer: mailer, Logger: logger}
}

type Factory struct {
	Logger *Logger
}

func NewFactory(logger *Logger) *Factory {
	return &Factory{Logger: logger}
}

func (f *Factory) NewHandler(db *DB, mailer *Mailer) *Handler {
	return &Handler{DB: db, Mailer: mailer, Logger: f.Logger}
}

type Notifier interface {
	Notify() error
}

type EmailNotifier struct {
	Mailer *Mailer
}

func NewEmailNotifier(mailer *Mailer) *EmailNotifier {
	return &EmailNotifier{Mailer: mailer}
}

func (n *EmailNotifier) Notify() error { return nil }

type SMSNotifier struct{}

func NewSMSNotifier() SMSNotifier { return SMSNotifier{} }

func (n SMSNotifier) Notify() error { return nil }

