package safecache

import (
	"sync"
)

type Cache[T any] struct {
	val           T
	once          sync.Once
	mu            sync.RWMutex
	initFunc      func() (T, error)
	bypassFunc    func() bool
	isInitialized bool
}

func New[T any](initFunc func() (T, error), bypassFunc func() bool) *Cache[T] {
	return &Cache[T]{
		initFunc:   initFunc,
		bypassFunc: bypassFunc,
	}
}

func (c *Cache[T]) Get() (T, error) {
	// Bypass if bypassFunc is provided and returns true
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

	// Check for initialization error
	if err != nil {
		return c.val, err // c.val will be zero value of T
	}

	// Return the initialized value
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.val, nil
}

type mapInitFunc[K any, V any] func(key K) (V, error)
type mapBypassFunc[K any] func(key K) bool
type mapToKeyFunc[K any, DK comparable] func(key K) DK

type CacheMap[K any, DK comparable, V any] struct {
	cache         map[DK]*Cache[V]
	mu            sync.RWMutex
	mapInitFunc   mapInitFunc[K, V]
	mapBypassFunc mapBypassFunc[K]
	mapToKeyFunc  mapToKeyFunc[K, DK]
}

func NewMap[K any, DK comparable, V any](initFunc mapInitFunc[K, V], mapToKeyFunc mapToKeyFunc[K, DK], bypassFunc mapBypassFunc[K]) *CacheMap[K, DK, V] {
	return &CacheMap[K, DK, V]{
		cache:         make(map[DK]*Cache[V]),
		mapInitFunc:   initFunc,
		mapToKeyFunc:  mapToKeyFunc,
		mapBypassFunc: bypassFunc,
	}
}

func (c *CacheMap[K, DK, V]) Get(key K) (V, error) {
	c.mu.RLock()
	var derivedKey DK
	if c.mapToKeyFunc != nil {
		derivedKey = c.mapToKeyFunc(key)
	}
	cache, ok := c.cache[derivedKey]
	c.mu.RUnlock()
	if !ok {
		c.mu.Lock()
		defer c.mu.Unlock()
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
	return cache.Get()
}
