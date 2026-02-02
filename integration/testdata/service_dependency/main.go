package main

func main() {
	container := NewContainer(nil)
	service, err := container.GetService()
	if err != nil {
		panic(err)
	}
	service.Run()
}
