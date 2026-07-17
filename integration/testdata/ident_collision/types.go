package main

import "strings"

type Notifier struct {
	Name string
}

func NewDotNotifier() *Notifier {
	return &Notifier{Name: "dot"}
}

func NewCamelNotifier() *Notifier {
	return &Notifier{Name: "camel"}
}

func NewSnakeNotifier() *Notifier {
	return &Notifier{Name: "snake"}
}

type App struct {
	notifiers []*Notifier
}

func NewApp(dot, camel, snake *Notifier) *App {
	return &App{notifiers: []*Notifier{dot, camel, snake}}
}

func (a *App) Describe() string {
	names := make([]string, 0, len(a.notifiers))
	for _, n := range a.notifiers {
		names = append(names, n.Name)
	}
	return strings.Join(names, " ")
}
