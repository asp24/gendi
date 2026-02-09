package main

import (
	"fmt"
	"io"
)

type Writer struct {
	out io.Writer
}

func NewWriter(out io.Writer) *Writer {
	return &Writer{out: out}
}

func (w *Writer) Write(msg string) {
	fmt.Fprintln(w.out, msg)
}
