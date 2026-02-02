package main

type Database struct{}

func NewDatabase() *Database { return &Database{} }

type Handler interface{ Name() string }
type HomeHandler struct{}
type APIHandler struct{}

func NewHomeHandler() Handler { return &HomeHandler{} }
func NewAPIHandler() Handler  { return &APIHandler{} }

func (h *HomeHandler) Name() string { return "home" }
func (h *APIHandler) Name() string  { return "api" }

type App struct {
	name     string
	db       *Database
	handlers []Handler
}

func NewApp(name string, db *Database, handlers []Handler) *App {
	return &App{name: name, db: db, handlers: handlers}
}
