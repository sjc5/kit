package lru

import (
	"container/list"
	"sync"
)

type item[K comparable, V any] struct {
	key              K
	value            V
	element          *list.Element
	neverMoveToFront bool
}

type cache[K comparable, V any] struct {
	mu       sync.RWMutex
	items    map[K]*item[K, V]
	order    *list.List
	maxItems int
}

func NewCache[K comparable, V any](maxItems int) *cache[K, V] {
	return &cache[K, V]{
		items:    make(map[K]*item[K, V]),
		order:    list.New(),
		maxItems: maxItems,
	}
}

func (c *cache[K, V]) Get(key K) (v V, found bool) {
	c.mu.RLock()
	itm, found := c.items[key]
	c.mu.RUnlock()
	if !found {
		return
	}
	if !itm.neverMoveToFront {
		c.mu.Lock()
		c.order.MoveToFront(itm.element)
		c.mu.Unlock()
	}
	return itm.value, true
}

func (c *cache[K, V]) Set(key K, value V, neverMoveToFront bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if itm, found := c.items[key]; found {
		if !itm.neverMoveToFront {
			c.order.MoveToFront(itm.element)
			itm.value = value
			itm.neverMoveToFront = neverMoveToFront
		}
		return
	}

	if c.order.Len() > c.maxItems {
		c.evict()
	}

	itm := &item[K, V]{key: key, value: value, neverMoveToFront: neverMoveToFront}
	element := c.order.PushFront(itm)
	itm.element = element
	c.items[key] = itm
}

func (c *cache[K, V]) evict() {
	back := c.order.Back()
	if back != nil {
		itm := back.Value.(*item[K, V])
		delete(c.items, itm.key)
		c.order.Remove(back)
	}
}
