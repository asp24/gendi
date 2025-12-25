package generator

import (
	"fmt"
	"strings"
)

// ErrorSnippetBuilder generates consistent error handling code snippets.
type ErrorSnippetBuilder struct {
	serviceID   string
	context     []string
	contextVars []interface{}
	multiline   bool
	customCheck string // For non-standard conditions like nil checks
}

// NewErrorSnippet creates a new error snippet builder for a service.
func NewErrorSnippet(serviceID string) *ErrorSnippetBuilder {
	return &ErrorSnippetBuilder{
		serviceID:   serviceID,
		context:     []string{},
		contextVars: []interface{}{},
		multiline:   false,
	}
}

// WithContext adds a context component to the error message.
func (b *ErrorSnippetBuilder) WithContext(format string, args ...interface{}) *ErrorSnippetBuilder {
	b.context = append(b.context, format)
	b.contextVars = append(b.contextVars, args...)
	return b
}

// Multiline sets whether to use multiline formatting.
func (b *ErrorSnippetBuilder) Multiline() *ErrorSnippetBuilder {
	b.multiline = true
	return b
}

// CustomCondition sets a custom condition instead of "err != nil".
func (b *ErrorSnippetBuilder) CustomCondition(condition string) *ErrorSnippetBuilder {
	b.customCheck = condition
	return b
}

// Build generates the error handling code snippet.
func (b *ErrorSnippetBuilder) Build() string {
	// Build the error message format
	msgParts := []string{"service %q"}
	msgArgs := []interface{}{b.serviceID}

	for i, ctx := range b.context {
		msgParts = append(msgParts, ctx)
		// Calculate how many args this format string needs
		argCount := strings.Count(ctx, "%")
		for j := 0; j < argCount; j++ {
			if i*2+j < len(b.contextVars) {
				msgArgs = append(msgArgs, b.contextVars[i*2+j])
			}
		}
	}

	// Determine the condition
	condition := "err != nil"
	wrappedErr := true
	if b.customCheck != "" {
		condition = b.customCheck
		wrappedErr = false
	}

	// Build the error format string
	errorMsg := strings.Join(msgParts, " ")
	if wrappedErr {
		errorMsg += ": %w"
	}

	// Build the fmt.Errorf call
	fmtArgs := make([]string, 0, len(msgArgs)+1)
	for _, arg := range msgArgs {
		fmtArgs = append(fmtArgs, fmt.Sprintf("%q", arg))
	}
	if wrappedErr {
		fmtArgs = append(fmtArgs, "err")
	}

	errCall := fmt.Sprintf("fmt.Errorf(%q, %s)", errorMsg, strings.Join(fmtArgs, ", "))

	// Format the final snippet
	if b.multiline {
		return fmt.Sprintf("if %s {\n\t\treturn zero, %s\n\t}", condition, errCall)
	}
	return fmt.Sprintf("if %s { return zero, %s }", condition, errCall)
}

// Convenience functions for common error patterns

func serviceConstructorError(serviceID string) string {
	return NewErrorSnippet(serviceID).
		WithContext("constructor").
		Multiline().
		Build()
}

func serviceArgError(serviceID string, argIndex int) string {
	return NewErrorSnippet(serviceID).
		WithContext("arg[%d]", argIndex).
		Build()
}

func serviceParamError(serviceID string, argIndex int, paramName string) string {
	return NewErrorSnippet(serviceID).
		WithContext("arg[%d]", argIndex).
		WithContext("param %q", paramName).
		Build()
}

func serviceParamNilCheck(serviceID string, argIndex int, paramName string) string {
	return NewErrorSnippet(serviceID).
		WithContext("arg[%d]", argIndex).
		WithContext("param %q", paramName).
		WithContext("parameters provider is nil").
		CustomCondition("c.params == nil").
		Build()
}

func serviceTagError(serviceID string, argIndex int, tagName string) string {
	return NewErrorSnippet(serviceID).
		WithContext("arg[%d]", argIndex).
		WithContext("tag %q", tagName).
		Build()
}

func serviceReceiverError(serviceID, receiverID string) string {
	return NewErrorSnippet(serviceID).
		WithContext("receiver %q", receiverID).
		Build()
}

func serviceBaseError(serviceID, baseID string) string {
	return NewErrorSnippet(serviceID).
		WithContext("base %q", baseID).
		Multiline().
		Build()
}

func serviceDecoratorError(serviceID, decoratorID string) string {
	return NewErrorSnippet(serviceID).
		WithContext("decorator %q", decoratorID).
		Multiline().
		Build()
}
