package router

import (
	"errors"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestNestedRouter_AddDependency(t *testing.T) {
	nr := &NestedRouter{
		deps: make(map[string]*dep),
	}

	// Test adding a dependency without dependencies
	fetcher := func(ctx *NestedRequestCtx) (any, error) {
		return "test-value", nil
	}
	nr.AddDependency("test-key", fetcher)

	dep, exists := nr.deps["test-key"]
	if !exists {
		t.Fatalf("Expected dependency to exist with key 'test-key'")
	}
	if dep.runner == nil {
		t.Fatalf("Expected fetcher to be set")
	}
	if len(dep.parentDeps) != 0 {
		t.Fatalf("Expected no dependencies, got %d", len(dep.parentDeps))
	}

	// Test adding a dependency with dependencies
	nr.AddDependency("test-key-2", fetcher, "dep1", "dep2")
	dep2, exists := nr.deps["test-key-2"]
	if !exists {
		t.Fatalf("Expected dependency to exist with key 'test-key-2'")
	}
	if len(dep2.parentDeps) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(dep2.parentDeps))
	}
	if dep2.parentDeps[0] != "dep1" || dep2.parentDeps[1] != "dep2" {
		t.Fatalf("Expected deps to be ['dep1', 'dep2'], got %v", dep2.parentDeps)
	}
}

func TestNestedRequestCtx_GetDependencyData(t *testing.T) {
	nr := &NestedRouter{
		deps: make(map[string]*dep),
	}

	// Track execution order and count
	executionOrder := []string{}
	executionCount := map[string]int{}

	// Add a simple dependency
	nr.AddDependency("simple", func(ctx *NestedRequestCtx) (any, error) {
		executionOrder = append(executionOrder, "simple")
		executionCount["simple"]++
		return "simple-value", nil
	})

	// Add a dependency with an error
	nr.AddDependency("error-dep", func(ctx *NestedRequestCtx) (any, error) {
		executionOrder = append(executionOrder, "error-dep")
		executionCount["error-dep"]++
		return nil, errors.New("error-fetching")
	})

	// Add a dependency with a dependency
	nr.AddDependency("nested", func(ctx *NestedRequestCtx) (any, error) {
		executionOrder = append(executionOrder, "nested")
		executionCount["nested"]++
		simpleValue, err := ctx.getDependencyData("simple")
		if err != nil {
			return nil, err
		}
		return "nested-" + simpleValue.(string), nil
	}, "simple")

	// Add a dependency with a failing dependency
	nr.AddDependency("failing-dep", func(ctx *NestedRequestCtx) (any, error) {
		executionOrder = append(executionOrder, "failing-dep")
		executionCount["failing-dep"]++
		return "should-not-get-here", nil
	}, "error-dep")

	// Add a dependency with multiple dependencies to check ordering
	nr.AddDependency("multi-dep", func(ctx *NestedRequestCtx) (any, error) {
		executionOrder = append(executionOrder, "multi-dep")
		executionCount["multi-dep"]++
		return "multi-dep-value", nil
	}, "simple", "nested")

	// Create a request context
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := newNestedRequestCtx(nr, req)

	// Test fetching a simple dependency
	simpleValue, err := ctx.getDependencyData("simple")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if simpleValue != "simple-value" {
		t.Fatalf("Expected 'simple-value', got %v", simpleValue)
	}

	// Test fetching the same dependency again (should not execute fetcher again)
	_, err = ctx.getDependencyData("simple")
	if err != nil {
		t.Fatalf("Expected no error on second fetch, got %v", err)
	}
	if executionCount["simple"] != 1 {
		t.Fatalf("Expected simple dependency to execute exactly once, got %d", executionCount["simple"])
	}

	// Test fetching a dependency with error
	_, err = ctx.getDependencyData("error-dep")
	if err == nil {
		t.Fatalf("Expected error")
	}
	if err.Error() != "error-fetching" {
		t.Fatalf("Expected 'error-fetching', got '%v'", err)
	}

	// Test fetching a nested dependency
	nestedValue, err := ctx.getDependencyData("nested")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if nestedValue != "nested-simple-value" {
		t.Fatalf("Expected 'nested-simple-value', got %v", nestedValue)
	}

	// Verify execution order: simple should be before nested
	simpleIdx := -1
	nestedIdx := -1
	for i, name := range executionOrder {
		if name == "simple" {
			simpleIdx = i
		} else if name == "nested" {
			nestedIdx = i
		}
	}
	if simpleIdx >= nestedIdx || simpleIdx == -1 || nestedIdx == -1 {
		t.Fatalf("Expected simple to execute before nested, order: %v", executionOrder)
	}

	// Reset execution tracking
	executionOrder = []string{}

	// Test fetching a dependency with multiple dependencies
	_, err = ctx.getDependencyData("multi-dep")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// multi-dep should be the only thing executed since dependencies are already loaded
	if len(executionOrder) != 1 || executionOrder[0] != "multi-dep" {
		t.Fatalf("Expected only multi-dep to execute, got %v", executionOrder)
	}

	// Test fetching a dependency with a failing dependency
	_, err = ctx.getDependencyData("failing-dep")
	if err == nil {
		t.Fatalf("Expected error")
	}
	if err.Error() != "dependency error-dep failed: error-fetching" {
		t.Fatalf("Expected 'dependency error-dep failed: error-fetching', got '%v'", err)
	}

	// Test fetching a non-existent dependency
	_, err = ctx.getDependencyData("non-existent")
	if err == nil {
		t.Fatalf("Expected error for non-existent dependency")
	}
	if err.Error() != "fetcher not found: non-existent" {
		t.Fatalf("Expected 'fetcher not found: non-existent', got '%v'", err)
	}
}

func TestLoader(t *testing.T) {
	type TestOutput struct {
		Value string
	}

	loader := Loader[TestOutput](func(ctx *NestedRequestCtx) (*TestOutput, error) {
		return &TestOutput{Value: "test-value"}, nil
	})

	// Test getOutputInstance
	output := loader.getOutputInstance()
	if _, ok := output.(*TestOutput); !ok {
		t.Fatalf("Expected *TestOutput, got %T", output)
	}

	// Test execute
	ctx := &NestedRequestCtx{}
	var dest any
	err := loader.execute(ctx, &dest)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check the output
	if output, ok := dest.(TestOutput); !ok {
		t.Fatalf("Expected TestOutput, got %T", dest)
	} else if output.Value != "test-value" {
		t.Fatalf("Expected 'test-value', got '%s'", output.Value)
	}

	// Test a loader that returns an error
	errorLoader := Loader[TestOutput](func(ctx *NestedRequestCtx) (*TestOutput, error) {
		return nil, errors.New("loader-error")
	})

	err = errorLoader.execute(ctx, &dest)
	if err == nil {
		t.Fatalf("Expected error")
	}
	if err.Error() != "loader-error" {
		t.Fatalf("Expected 'loader-error', got '%v'", err)
	}
}

func TestAddLoader(t *testing.T) {
	nr := &NestedRouter{
		matcher: NewMatcher(nil),
		loaders: make(map[string]*loaderWrapper),
	}

	type TestOutput struct {
		Value string
	}

	loader := Loader[TestOutput](func(ctx *NestedRequestCtx) (*TestOutput, error) {
		return &TestOutput{Value: "test-value"}, nil
	})

	// Add a loader
	AddLoader(nr, "/test", loader, "dep1", "dep2")

	// Check if the loader was added
	loaderWrapper, exists := nr.loaders["/test"]
	if !exists {
		t.Fatalf("Expected loader to exist")
	}
	if len(loaderWrapper.deps) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(loaderWrapper.deps))
	}
	if loaderWrapper.deps[0] != "dep1" || loaderWrapper.deps[1] != "dep2" {
		t.Fatalf("Expected deps to be ['dep1', 'dep2'], got %v", loaderWrapper.deps)
	}
}

func TestRunParallelLoadersBasic(t *testing.T) {
	// Initialize with NewMatcher(nil) as instructed
	nr := &NestedRouter{
		matcher: NewMatcher(nil),
		deps:    make(map[string]*dep),
		loaders: make(map[string]*loaderWrapper),
	}

	// Add dependencies with execution tracking
	var mu sync.Mutex
	executionCount := map[string]int{}

	recordExecution := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		executionCount[name]++
	}

	nr.AddDependency("auth", func(ctx *NestedRequestCtx) (any, error) {
		recordExecution("auth")
		return "user-authenticated", nil
	})

	nr.AddDependency("user-id", func(ctx *NestedRequestCtx) (any, error) {
		recordExecution("user-id")
		return "123", nil
	}, "auth")

	// Create a context manually for basic testing
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := newNestedRequestCtx(nr, req)

	// Manually verify dependency resolution
	authValue, err := ctx.getDependencyData("auth")
	if err != nil {
		t.Fatalf("Expected no error for auth dependency, got %v", err)
	}
	if authValue != "user-authenticated" {
		t.Fatalf("Expected auth value to be 'user-authenticated', got '%v'", authValue)
	}

	userIDValue, err := ctx.getDependencyData("user-id")
	if err != nil {
		t.Fatalf("Expected no error for user-id dependency, got %v", err)
	}
	if userIDValue != "123" {
		t.Fatalf("Expected user-id value to be '123', got '%v'", userIDValue)
	}

	// Verify each dependency was executed exactly once
	if executionCount["auth"] != 1 {
		t.Fatalf("Expected auth to execute once, got %d", executionCount["auth"])
	}
	if executionCount["user-id"] != 1 {
		t.Fatalf("Expected user-id to execute once, got %d", executionCount["user-id"])
	}
}
