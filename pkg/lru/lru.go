package lru

import (
	"container/list"
	"sync"
)

type item[K comparable, V any] struct {
	key     K
	value   V
	element *list.Element
	isSpam  bool
}

type Cache[K comparable, V any] struct {
	mu       sync.RWMutex
	items    map[K]*item[K, V]
	order    *list.List
	maxItems int
}

func NewCache[K comparable, V any](maxItems int) *Cache[K, V] {
	return &Cache[K, V]{
		items:    make(map[K]*item[K, V]),
		order:    list.New(),
		maxItems: maxItems,
	}
}

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

func (c *Cache[K, V]) Set(key K, value V, isSpam bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if itm, found := c.items[key]; found {
		if !itm.isSpam {
			c.order.MoveToFront(itm.element)
			itm.value = value
			itm.isSpam = isSpam
		}
		return
	}

	if c.order.Len() > c.maxItems {
		c.evict()
	}

	itm := &item[K, V]{key: key, value: value, isSpam: isSpam}
	element := c.order.PushFront(itm)
	itm.element = element
	c.items[key] = itm
}

func (c *Cache[K, V]) evict() {
	back := c.order.Back()
	if back != nil {
		itm := back.Value.(*item[K, V])
		delete(c.items, itm.key)
		c.order.Remove(back)
	}
}

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
