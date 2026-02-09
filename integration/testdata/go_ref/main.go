package main

func main() {
	container := NewContainer(nil)
	writer, err := container.GetWriter()
	if err != nil {
		panic(err)
	}
	writer.Write("hello from go ref")
}
