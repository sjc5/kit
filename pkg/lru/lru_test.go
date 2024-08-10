package lru

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache := NewCache[string, int](5)
	if cache.maxItems != 5 {
		t.Errorf("Expected maxItems to be 5, got %d", cache.maxItems)
	}
	if len(cache.items) != 0 {
		t.Errorf("Expected empty items map, got %d items", len(cache.items))
	}
	if cache.order.Len() != 0 {
		t.Errorf("Expected empty order list, got %d items", cache.order.Len())
	}
}

func TestSet(t *testing.T) {
	cache := NewCache[string, int](3)

	// Test basic set
	cache.Set("a", 1, false)
	if v, found := cache.Get("a"); !found || v != 1 {
		t.Errorf("Expected to find 'a' with value 1, got %v, %v", v, found)
	}

	// Test overwrite
	cache.Set("a", 2, false)
	if v, found := cache.Get("a"); !found || v != 2 {
		t.Errorf("Expected to find 'a' with value 2, got %v, %v", v, found)
	}

	// Test eviction
	cache.Set("b", 3, false)
	cache.Set("c", 4, false)
	cache.Set("d", 5, false) // This should evict "a"
	if _, found := cache.Get("a"); found {
		t.Errorf("Expected 'a' to be evicted")
	}

	// Test spam flag
	cache.Set("e", 6, true)
	if v, found := cache.Get("e"); !found || v != 6 {
		t.Errorf("Expected to find 'e' with value 6, got %v, %v", v, found)
	}
}

func TestGet(t *testing.T) {
	cache := NewCache[string, int](3)

	// Test get non-existent item
	if _, found := cache.Get("a"); found {
		t.Errorf("Expected not to find 'a'")
	}

	// Test get existing item
	cache.Set("a", 1, false)
	if v, found := cache.Get("a"); !found || v != 1 {
		t.Errorf("Expected to find 'a' with value 1, got %v, %v", v, found)
	}

	// Test LRU order update
	cache.Set("b", 2, false)
	cache.Set("c", 3, false)
	cache.Get("a")           // This should move "a" to the front
	cache.Set("d", 4, false) // This should evict "b"
	if _, found := cache.Get("b"); found {
		t.Errorf("Expected 'b' to be evicted")
	}

	// Test spam item
	cache.Set("e", 5, true)
	cache.Get("e")           // This should not move "e" to the front
	cache.Set("f", 6, false) // This should evict "c", not "e"
	if _, found := cache.Get("c"); found {
		t.Errorf("Expected 'c' to be evicted")
	}
	if _, found := cache.Get("e"); !found {
		t.Errorf("Expected 'e' to still be in cache")
	}
}

func TestDelete(t *testing.T) {
	cache := NewCache[string, int](3)

	// Test delete non-existent item
	cache.Delete("a")
	if cache.order.Len() != 0 {
		t.Errorf("Expected empty cache after deleting non-existent item")
	}

	// Test delete existing item
	cache.Set("a", 1, false)
	cache.Set("b", 2, false)
	cache.Delete("a")
	if _, found := cache.Get("a"); found {
		t.Errorf("Expected 'a' to be deleted")
	}
	if cache.order.Len() != 1 {
		t.Errorf("Expected cache to have 1 item, got %d", cache.order.Len())
	}

	// Test delete spam item
	cache.Set("c", 3, true)
	cache.Delete("c")
	if _, found := cache.Get("c"); found {
		t.Errorf("Expected spam item 'c' to be deleted")
	}
}

func TestEdgeCases(t *testing.T) {
	// Test cache with size 0
	cache := NewCache[string, int](0)
	cache.Set("a", 1, false)
	if _, found := cache.Get("a"); found {
		t.Errorf("Expected item not to be stored in size 0 cache")
	}

	// Test cache with size 1
	cache = NewCache[string, int](1)
	cache.Set("a", 1, false)
	cache.Set("b", 2, false)
	if _, found := cache.Get("a"); found {
		t.Errorf("Expected 'a' to be evicted in size 1 cache")
	}
	if v, found := cache.Get("b"); !found || v != 2 {
		t.Errorf("Expected to find 'b' with value 2 in size 1 cache")
	}

	// Test setting same key multiple times
	cache = NewCache[string, int](2)
	cache.Set("a", 1, false)
	cache.Set("a", 2, false)
	cache.Set("a", 3, false)
	if v, found := cache.Get("a"); !found || v != 3 {
		t.Errorf("Expected to find 'a' with value 3, got %v, %v", v, found)
	}

	// Test changing spam status
	cache = NewCache[string, int](2)
	cache.Set("a", 1, true)
	cache.Set("b", 2, false)
	cache.Set("a", 1, false) // Change "a" from spam to non-spam
	cache.Set("c", 3, false) // This should evict "b", not "a"
	if _, found := cache.Get("b"); found {
		t.Errorf("Expected 'b' to be evicted")
	}
	if _, found := cache.Get("a"); !found {
		t.Errorf("Expected 'a' to still be in cache")
	}
}

func TestConcurrency(t *testing.T) {
	t.Run("Concurrent Set and Get", func(t *testing.T) {
		cache := NewCache[string, int](100)
		var wg sync.WaitGroup
		iterations := 1000
		goroutines := 10

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					key := fmt.Sprintf("key%d-%d", id, j)
					cache.Set(key, j, false)
					_, _ = cache.Get(key)
				}
			}(i)
		}

		wg.Wait()

		if cache.order.Len() > cache.maxItems {
			t.Errorf("Cache exceeded max items: %d > %d", cache.order.Len(), cache.maxItems)
		}
	})

	t.Run("Concurrent Set, Get, and Delete", func(t *testing.T) {
		cache := NewCache[string, int](100)
		var wg sync.WaitGroup
		iterations := 1000
		goroutines := 10

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					key := fmt.Sprintf("key%d-%d", id, j)
					cache.Set(key, j, j%2 == 0)
					_, _ = cache.Get(key)
					if j%3 == 0 {
						cache.Delete(key)
					}
				}
			}(i)
		}

		wg.Wait()

		if cache.order.Len() > cache.maxItems {
			t.Errorf("Cache exceeded max items: %d > %d", cache.order.Len(), cache.maxItems)
		}
	})

	t.Run("Concurrent access with mixed workload", func(t *testing.T) {
		cache := NewCache[string, int](1000)
		var wg sync.WaitGroup
		iterations := 10000
		goroutines := 20

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					key := fmt.Sprintf("key%d-%d", id%10, j%100)
					switch j % 10 {
					case 0, 1, 2, 3, 4:
						// 50% reads
						_, _ = cache.Get(key)
					case 5, 6, 7:
						// 30% writes
						cache.Set(key, j, false)
					case 8:
						// 10% spam writes
						cache.Set(key, j, true)
					case 9:
						// 10% deletes
						cache.Delete(key)
					}
				}
			}(i)
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Test passed
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out, possible deadlock")
		}

		if cache.order.Len() > cache.maxItems {
			t.Errorf("Cache exceeded max items: %d > %d", cache.order.Len(), cache.maxItems)
		}
	})
}
