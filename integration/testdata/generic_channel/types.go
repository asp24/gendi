package main

type Event struct {
	Name string
}

func NewEventChannel(size int) chan Event {
	return make(chan Event, size)
}

type App struct {
	events chan Event
}

func NewApp(events chan Event) *App {
	return &App{events: events}
}
