package main

type App struct {
	message string
	env     string
}

func NewApp(message, env string) *App {
	return &App{message: message, env: env}
}
