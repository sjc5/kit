package safecache

import (
	"sync"
)

type Cache[T any] struct {
	val           T
	once          sync.Once
	mu            sync.RWMutex
	initFunc      func() (T, error)
	bypassFunc    *func() bool
	isInitialized bool
}

func New[T any](initFunc func() (T, error), bypassFunc *func() bool) *Cache[T] {
	return &Cache[T]{
		initFunc:   initFunc,
		bypassFunc: bypassFunc,
	}
}

func (c *Cache[T]) Get() (T, error) {
	// Bypass if bypassFunc is provided and returns true
	if c.bypassFunc != nil && (*c.bypassFunc)() {
		return c.initFunc()
	}

	// First, try to read without locking
	c.mu.RLock()
	if c.isInitialized {
		defer c.mu.RUnlock()
		return c.val, nil
	}
	c.mu.RUnlock()

	// If not initialized, use sync.Once to ensure single initialization
	var err error
	c.once.Do(func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.val, err = c.initFunc()
		if err == nil {
			c.isInitialized = true
		}
	})

	// Check for initialization error
	if err != nil {
		return c.val, err // c.val will be zero value of T
	}

	// Return the initialized value
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.val, nil
}
