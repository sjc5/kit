// Package safecache provides a generic, thread-safe, lazily initiated cache that
// ensures initialization occurs only once unless bypass is requested.
package safecache

import (
	"sync"
)

// Cache is a generic, thread-safe cache that ensures initialization occurs only
// once unless bypass is requested.
type Cache[T any] struct {
	val           T
	once          sync.Once
	mu            sync.RWMutex
	initFunc      func() (T, error)
	bypassFunc    func() bool
	isInitialized bool
}

// New creates a new Cache instance. If bypassFunc is provided and returns true,
// initFunc will run every time. Panics if initFunc is nil.
func New[T any](initFunc func() (T, error), bypassFunc func() bool) *Cache[T] {
	if initFunc == nil {
		panic("initFunc must not be nil")
	}
	return &Cache[T]{
		initFunc:   initFunc,
		bypassFunc: bypassFunc,
	}
}

// Get retrieves the cached value, initializing it if necessary or bypassing the cache.
func (c *Cache[T]) Get() (T, error) {
	// If bypassFunc is provided and returns true, always run initFunc
	if c.bypassFunc != nil && (c.bypassFunc)() {
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

	// Return the initialized value
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.val, err
}

// mapInitFunc defines the initialization function for each key in the CacheMap.
type mapInitFunc[K any, V any] func(key K) (V, error)

// mapBypassFunc defines the bypass function for each key in the CacheMap.
type mapBypassFunc[K any] func(key K) bool

// mapToKeyFunc defines the function to derive the key used to store/retrieve
// values in the CacheMap.
type mapToKeyFunc[K any, DK comparable] func(key K) DK

// CacheMap is a generic, thread-safe cache map that caches values based on
// derived keys.
type CacheMap[K any, DK comparable, V any] struct {
	cache         map[DK]*Cache[V]
	mu            sync.RWMutex
	mapInitFunc   mapInitFunc[K, V]
	mapBypassFunc mapBypassFunc[K]
	mapToKeyFunc  mapToKeyFunc[K, DK]
}

// NewMap creates a new CacheMap instance. If bypassFunc is provided and returns
// true, initFunc will run every time. Panics if initFunc or mapToKeyFunc is nil.
func NewMap[K any, DK comparable, V any](
	initFunc mapInitFunc[K, V],
	mapToKeyFunc mapToKeyFunc[K, DK],
	bypassFunc mapBypassFunc[K],
) *CacheMap[K, DK, V] {
	if initFunc == nil {
		panic("initFunc must not be nil")
	}
	if mapToKeyFunc == nil {
		panic("mapToKeyFunc must not be nil")
	}
	return &CacheMap[K, DK, V]{
		cache:         make(map[DK]*Cache[V]),
		mapInitFunc:   initFunc,
		mapToKeyFunc:  mapToKeyFunc,
		mapBypassFunc: bypassFunc,
	}
}

// Get retrieves the cached value for the given key, initializing it if necessary
// or bypassing the cache.
func (c *CacheMap[K, DK, V]) Get(key K) (V, error) {
	derivedKey := c.mapToKeyFunc(key)

	c.mu.RLock()
	cache, ok := c.cache[derivedKey]
	c.mu.RUnlock()
	if !ok {
		c.mu.Lock()
		defer c.mu.Unlock()

		// Double-check whether the cache has been created during the lock acquisition
		if cache, ok = c.cache[derivedKey]; !ok {
			var bypassFunc func() bool
			if c.mapBypassFunc != nil {
				bypassFunc = func() bool {
					return c.mapBypassFunc(key)
				}
			}
			cache = New(
				func() (V, error) {
					return c.mapInitFunc(key)
				},
				bypassFunc,
			)
			c.cache[derivedKey] = cache
		}
	}

	// If bypass is requested, the Cache's Get method will handle it appropriately
	return cache.Get()
}
