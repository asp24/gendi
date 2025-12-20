package app

import (
	"fmt"
)

type Logger struct {
	Prefix string
}

func (l *Logger) Log(message string) {
	fmt.Printf("%s: %s\n", l.Prefix, message)
}

func NewLogger(prefix string) *Logger {
	return &Logger{Prefix: prefix}
}

type Repo struct {
	DSN string
}

func NewRepo(dsn string) *Repo {
	return &Repo{DSN: dsn}
}

type PaymentProvider struct {
	Name string
}

func NewStripeProvider() *PaymentProvider {
	return &PaymentProvider{Name: "stripe"}
}

func NewPaypalProvider() *PaymentProvider {
	return &PaymentProvider{Name: "paypal"}
}

type Service struct {
	Repo      *Repo
	Logger    *Logger
	Providers []*PaymentProvider
	Decorated bool
}

func NewService(repo *Repo, logger *Logger, providers []*PaymentProvider) (*Service, error) {
	return &Service{Repo: repo, Logger: logger, Providers: providers}, nil
}

func DecorateService(inner *Service, logger *Logger) *Service {
	inner.Decorated = true
	inner.Logger = logger
	return inner
}

type Factory struct {
	Logger *Logger
}

func NewFactory(logger *Logger) *Factory {
	return &Factory{Logger: logger}
}

type Processor struct {
	Repo   *Repo
	Logger *Logger
}

func (f *Factory) NewProcessor(repo *Repo) *Processor {
	return &Processor{Repo: repo, Logger: f.Logger}
}
