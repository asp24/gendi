package main

import (
	"fmt"
	"log/slog"
	"net/http"
)

// App wires together the services produced by stdlib/gendi.yaml, exercising
// each stdlib constructor through the generated container.
type App struct {
	client      *http.Client
	pooled      *http.Client
	logger      *slog.Logger
	jsonHandler slog.Handler
}

func NewApp(
	client *http.Client,
	pooled *http.Client,
	logger *slog.Logger,
	jsonHandler slog.Handler,
) *App {
	return &App{
		client:      client,
		pooled:      pooled,
		logger:      logger,
		jsonHandler: jsonHandler,
	}
}

func (a *App) Run() {
	// The logger writes to stderr, so it does not interfere with the stdout
	// assertions below; emitting a record still exercises the full slog path.
	a.logger.Info("stdlib integration")

	fmt.Printf("http client timeout: %s\n", a.client.Timeout)
	fmt.Printf("pooled client timeout: %s\n", a.pooled.Timeout)
	fmt.Printf("pooled transport set: %t\n", a.pooled.Transport != nil)
	fmt.Printf("logger ready: %t\n", a.logger != nil)
	fmt.Printf("json handler ready: %t\n", a.jsonHandler != nil)
}
