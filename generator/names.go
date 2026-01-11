package generator

import "fmt"

func defaultParametersName(containerName string) string {
	return fmt.Sprintf("Default%sParameters", containerName)
}

func withErrorHandlerName(containerName string) string {
	return fmt.Sprintf("With%sErrorHandler", containerName)
}
