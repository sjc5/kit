package tasks

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestTasks(t *testing.T) {
	t.Run("BasicTaskExecution", func(t *testing.T) {
		registry := NewRegistry()
		task := New(registry, func(c *TasksCtxWithInput[string]) (string, error) {
			return "Hello, " + c.Input, nil
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		result, err := task.Prep(ctx, "World").Get()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "Hello, World" {
			t.Errorf("Expected 'Hello, World', got '%s'", result)
		}
	})

	t.Run("ParallelExecution", func(t *testing.T) {
		registry := NewRegistry()

		task1 := New(registry, func(c *TasksCtxWithInput[int]) (int, error) {
			time.Sleep(100 * time.Millisecond)
			return c.Input * 2, nil
		})

		task2 := New(registry, func(c *TasksCtxWithInput[int]) (int, error) {
			time.Sleep(100 * time.Millisecond)
			return c.Input * 3, nil
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		start := time.Now()

		twi1 := task1.Prep(ctx, 5)
		twi2 := task2.Prep(ctx, 5)
		ok := ctx.ParallelPreload(twi1, twi2)

		if !ok {
			t.Error("ParallelPreload failed")
		}

		result1, err1 := twi1.Get()
		result2, err2 := twi2.Get()

		duration := time.Since(start)

		if err1 != nil || err2 != nil {
			t.Errorf("Expected no errors, got %v, %v", err1, err2)
		}
		if result1 != 10 || result2 != 15 {
			t.Errorf("Expected 10 and 15, got %d and %d", result1, result2)
		}
		if duration > 150*time.Millisecond {
			t.Errorf("Expected parallel execution (<150ms), took %v", duration)
		}
	})

	t.Run("TaskDependencies", func(t *testing.T) {
		registry := NewRegistry()

		authTask := New(registry, func(c *TasksCtxWithInput[string]) (string, error) {
			return "token-" + c.Input, nil
		})

		userTask := New(registry, func(c *TasksCtxWithInput[string]) (string, error) {
			token, err := authTask.Prep(c.TasksCtx, c.Input).Get()
			if err != nil {
				return "", err
			}
			return "user-" + token, nil
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		result, err := userTask.Prep(ctx, "123").Get()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "user-token-123" {
			t.Errorf("Expected 'user-token-123', got '%s'", result)
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		registry := NewRegistry()

		task := New(registry, func(c *TasksCtxWithInput[string]) (string, error) {
			time.Sleep(200 * time.Millisecond)
			return "done", nil
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			ctx.CancelNativeContext()
		}()

		_, err := task.Prep(ctx, "test").Get()
		if err == nil {
			t.Error("Expected context cancellation error, got nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		registry := NewRegistry()

		task := New(registry, func(c *TasksCtxWithInput[string]) (string, error) {
			return "", errors.New("task failed")
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		result, err := task.Prep(ctx, "test").Get()

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if err.Error() != "task failed" {
			t.Errorf("Expected 'task failed' error, got '%v'", err)
		}
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})

	t.Run("OnceExecution", func(t *testing.T) {
		registry := NewRegistry()

		var counter int
		var mu sync.Mutex
		task := New(registry, func(c *TasksCtxWithInput[string]) (string, error) {
			mu.Lock()
			counter++
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			return "done", nil
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		twi := task.Prep(ctx, "test")

		var wg sync.WaitGroup
		wg.Add(3)

		for range 3 {
			go func() {
				defer wg.Done()
				_, err := twi.Get()
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}()
		}

		wg.Wait()

		if counter != 1 {
			t.Errorf("Expected task to run once, ran %d times", counter)
		}
	})

	t.Run("HTTPRequestContext", func(t *testing.T) {
		registry := NewRegistry()

		task := New(registry, func(c *TasksCtxWithInput[string]) (string, error) {
			return c.Request().URL.String(), nil
		})

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		ctx := registry.NewCtxFromRequest(req)
		result, err := task.Prep(ctx, "test").Get()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "http://example.com" {
			t.Errorf("Expected 'http://example.com', got '%s'", result)
		}
	})
}
