# Integration Tests

End-to-end integration tests for gendi that test the complete workflow:

**YAML → Config → IR → Generated Code → Compilation → Runtime**

## Test Structure

Tests are organized using embedded files in `testdata/` directories. Each test case is a directory containing:

- **`gendi.yaml`** - Service configuration
- **`types.go`** - Type definitions and constructors
- **`main.go`** - Main function that uses the generated container

Expected output is defined inline in the test struct in `integration_test.go`.

### integration_refactored_test.go

Main test runner using `embed.FS` to load test cases from `testdata/`:

- **TestWorkflow**: Runs all test cases from testdata
- **TestValidateAllTestdata**: Validates test file structure
- **TestListTests**: Lists all available tests
- **BenchmarkWorkflow**: Benchmarks the generation workflow

### Available Test Cases

Current test cases in `testdata/`:
- `basic_service` - Basic service injection with parameters
- `service_dependency` - Service depending on another service
- `tagged_injection` - Tagged services with sorting
- `decorator` - Service decoration pattern
- `method_constructor` - Method-based constructors

### Legacy Tests

- **integration_test.go** - Original inline test implementation (8 tests)
- **error_test.go** - Error case validation tests
- **advanced_test.go** - Complex scenario tests

These will be migrated to the testdata structure over time.

## Running Tests

```bash
# Run all integration tests
go test ./integration

# Run only testdata tests
go test ./integration -run TestWorkflow

# Run specific test
go test ./integration -run TestWorkflow/basic_service

# Run with verbose output
go test -v ./integration

# Run benchmarks
go test -bench=. ./integration

# Validate test structure
go test ./integration -run TestValidateAllTestdata

# List available tests
go test ./integration -run TestListTests
```

## Adding New Tests

To add a new integration test:

1. Create a new directory in `testdata/` with a descriptive name (e.g., `my_feature`)
2. Add the required files:

```bash
mkdir integration/testdata/my_feature
```

**testdata/my_feature/gendi.yaml:**
```yaml
services:
  my_service:
    constructor:
      func: "test.NewMyService"
    public: true
```

**testdata/my_feature/types.go:**
```go
package main

type MyService struct{}

func NewMyService() *MyService {
    return &MyService{}
}
```

**testdata/my_feature/main.go:**
```go
package main

import "fmt"

func main() {
    container := NewContainer(nil)
    svc, _ := container.GetMyService()
    fmt.Println("works")
}
```

3. Add the test case to `TestWorkflow` in `integration_test.go`:
```go
tests := []struct {
    name           string
    expectedOutput string
    wantCompileErr bool
    wantRuntimeErr bool
}{
    {
        name:           "my_feature",
        expectedOutput: "works\n",
    },
    // ...
}
```

The test will be automatically discovered and run!

## Test Workflow

Each test follows this workflow:

1. **Setup**: Create temporary directory with go.mod
2. **Write Files**: Copy gendi.yaml and types.go from testdata
3. **Generate**: Load config, apply passes, generate container code
4. **Compile**: Write main.go, run `go build`
5. **Execute**: Run the binary and capture output
6. **Verify**: Compare actual output with expectedOutput from test struct

## Notes

- Tests use temporary directories (automatically cleaned up)
- Each test is isolated with its own module
- Type resolution requires types.go before generation
- Main function is written after container generation
- Compilation and runtime errors are caught and reported

## Benefits of Testdata Structure

- **Clear separation**: Config and code are in separate files
- **Reusable**: Test files can be used as examples
- **Maintainable**: Easy to add, modify, or remove tests
- **Discoverable**: `TestListTests` shows all available tests
- **Validated**: `TestValidateAllTestdata` ensures structure is correct
- **Version controlled**: All test files are committed to the repo
- **IDE friendly**: Files can be edited with full IDE support

## Future Improvements

- [ ] Migrate remaining inline tests to testdata structure
- [ ] Add error case tests to testdata
- [ ] Add advanced feature tests
- [ ] Test custom compiler passes
- [ ] Test parameter providers
- [ ] Test stdlib integration
- [ ] Add negative test cases (expected failures)
- [ ] Performance regression tests
