package main

import "fmt"

func (a *App) Run() {
	// Send events
	a.events <- Event{Name: "start"}
	a.events <- Event{Name: "process"}
	a.events <- Event{Name: "end"}
	close(a.events)

	// Receive events
	count := 0
	for event := range a.events {
		fmt.Println(event.Name)
		count++
	}
	fmt.Printf("Processed %d events\n", count)
}

func main() {
	NewContainer(nil).MustApp().Run()
}
