// Package lru implements a generic Least Recently Used (LRU) cache
// with support for "spam" items. The cache maintains a maximum
// number of items, evicting the least recently used item when the
// limit is reached. Items can be added, retrieved, and deleted from
// the cache. Regular items are moved to the front of the cache when
// accessed or updated, while "spam" items maintain their position.
// This allows for preferential treatment of non-spam items in terms
// of retention, while still caching spam items. The cache is safe
// for concurrent use.
package lru

import (
	"container/list"
	"sync"
)

// item represents a key-value pair in the cache.
type item[K comparable, V any] struct {
	key     K
	value   V
	element *list.Element
	isSpam  bool
}

// Cache is a generic LRU cache that supports "spam" items.
type Cache[K comparable, V any] struct {
	mu       sync.RWMutex
	items    map[K]*item[K, V]
	order    *list.List
	maxItems int
}

// NewCache creates a new LRU cache with the specified maximum number of items.
func NewCache[K comparable, V any](maxItems int) *Cache[K, V] {
	if maxItems < 0 {
		maxItems = 0
	}
	return &Cache[K, V]{
		items:    make(map[K]*item[K, V]),
		order:    list.New(),
		maxItems: maxItems,
	}
}

// Get retrieves an item from the cache, moving non-spam items to the front.
// It returns the value and a boolean indicating whether the key was found.
func (c *Cache[K, V]) Get(key K) (v V, found bool) {
	c.mu.RLock()
	itm, found := c.items[key]
	c.mu.RUnlock()

	if !found {
		return
	}

	if !itm.isSpam {
		c.mu.Lock()
		c.order.MoveToFront(itm.element)
		c.mu.Unlock()
	}

	return itm.value, true
}

// Set adds or updates an item in the cache, evicting the LRU item if necessary.
// The isSpam parameter determines whether the item should be treated as spam.
// If the item is spam, it will not be moved to the front of the cache.
func (c *Cache[K, V]) Set(key K, value V, isSpam bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if itm, found := c.items[key]; found {
		itm.value = value
		itm.isSpam = isSpam
		if !isSpam {
			c.order.MoveToFront(itm.element)
		}
		return
	}

	if c.maxItems <= 0 {
		return
	}

	if len(c.items) >= c.maxItems {
		c.evict()
	}

	itm := &item[K, V]{key: key, value: value, isSpam: isSpam}
	element := c.order.PushFront(itm)
	itm.element = element
	c.items[key] = itm
}

// evict removes the least recently used item from the cache.
func (c *Cache[K, V]) evict() {
	back := c.order.Back()
	if back != nil {
		itm := back.Value.(*item[K, V])
		delete(c.items, itm.key)
		c.order.Remove(back)
	}
}

// Delete removes an item from the cache if it exists.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	itm, found := c.items[key]
	if !found {
		return
	}

	delete(c.items, key)
	c.order.Remove(itm.element)
}
