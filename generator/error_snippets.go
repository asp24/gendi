package generator

import "fmt"

// errorSnippets provides helper functions for generating consistent error handling code snippets

// serviceConstructorError generates error handling for service constructor failures
func serviceConstructorError(serviceID string) string {
	return fmt.Sprintf("if err != nil {\n\t\treturn zero, fmt.Errorf(\"service %%q constructor: %%w\", %q, err)\n\t}", serviceID)
}

// serviceArgError generates error handling for service argument resolution failures
func serviceArgError(serviceID string, argIndex int) string {
	return fmt.Sprintf("if err != nil { return zero, fmt.Errorf(\"service %%q arg[%%d]: %%w\", %q, %d, err) }", serviceID, argIndex)
}

// serviceParamError generates error handling for parameter resolution failures
func serviceParamError(serviceID string, argIndex int, paramName string) string {
	return fmt.Sprintf("if err != nil { return zero, fmt.Errorf(\"service %%q arg[%%d] param %%q: %%w\", %q, %d, %q, err) }", serviceID, argIndex, paramName)
}

// serviceParamNilCheck generates nil check for parameters provider
func serviceParamNilCheck(serviceID string, argIndex int, paramName string) string {
	return fmt.Sprintf("if c.params == nil { return zero, fmt.Errorf(\"service %%q arg[%%d] param %%q: parameters provider is nil\", %q, %d, %q) }", serviceID, argIndex, paramName)
}

// serviceTagError generates error handling for tagged service resolution failures
func serviceTagError(serviceID string, argIndex int, tagName string) string {
	return fmt.Sprintf("if err != nil { return zero, fmt.Errorf(\"service %%q arg[%%d] tag %%q: %%w\", %q, %d, %q, err) }", serviceID, argIndex, tagName)
}

// serviceReceiverError generates error handling for method receiver resolution failures
func serviceReceiverError(serviceID, receiverID string) string {
	return fmt.Sprintf("if err != nil { return zero, fmt.Errorf(\"service %%q receiver %%q: %%w\", %q, %q, err) }", serviceID, receiverID)
}

// serviceBaseError generates error handling for decorator base service resolution failures
func serviceBaseError(serviceID, baseID string) string {
	return fmt.Sprintf("if err != nil {\n\t\treturn zero, fmt.Errorf(\"service %%q base %%q: %%w\", %q, %q, err)\n\t}", serviceID, baseID)
}

// serviceDecoratorError generates error handling for decorator resolution failures
func serviceDecoratorError(serviceID, decoratorID string) string {
	return fmt.Sprintf("if err != nil {\n\t\treturn zero, fmt.Errorf(\"service %%q decorator %%q: %%w\", %q, %q, err)\n\t}", serviceID, decoratorID)
}
