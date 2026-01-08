package main

import (
	"fmt"
	"testing"

	"github.com/asp24/gendi/examples/basic/internal/di"
	"github.com/asp24/gendi/parameters"
)

func TestMustGetterWithCallback(t *testing.T) {
	// Example 1: Using MustService with default behavior (no callback)
	t.Run("default behavior", func(t *testing.T) {
		container := di.NewContainer(nil)
		svc := container.MustService()
		if svc == nil {
			t.Fatal("expected service, got nil")
		}
	})

	// Example 2: Using MustService with custom error handler
	t.Run("with custom error handler", func(t *testing.T) {
		var capturedService string
		var capturedError error

		container := di.NewContainer(
			nil,
			di.WithErrorHandler(func(serviceName string, err error) {
				capturedService = serviceName
				capturedError = err
				// In tests, we could call t.Fatalf here instead of panic
			}),
		)

		svc := container.MustService()
		if svc == nil {
			t.Fatal("expected service, got nil")
		}

		// No error should have been captured
		if capturedError != nil {
			t.Fatalf("unexpected error for service %q: %v", capturedService, capturedError)
		}
	})

	// Example 3: Demonstrating panic on error
	t.Run("panic on missing parameter", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()

		// Create container with empty parameters (missing "dsn" parameter)
		container := di.NewContainer(parameters.ProviderNullInstance)

		// This should panic because "repo" service needs "dsn" parameter
		// which is not provided
		_ = container.MustService()
	})

	// Example 4: Error handler is called before panic
	t.Run("error handler called before panic", func(t *testing.T) {
		var handlerCalled bool

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
			if !handlerCalled {
				t.Fatal("error handler should have been called before panic")
			}
		}()

		container := di.NewContainer(
			parameters.ProviderNullInstance,
			di.WithErrorHandler(func(serviceName string, err error) {
				handlerCalled = true
				fmt.Printf("Error building service %q: %v\n", serviceName, err)
			}),
		)

		_ = container.MustService()
	})
}
