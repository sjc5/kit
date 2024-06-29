package swr

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sjc5/kit/pkg/typed"
)

type GetterFunc func(r *http.Request) ([]byte, error)
type RequestToKeyFunc func(r *http.Request) string

type Cache struct {
	cache      typed.SyncMap[string, *ResponseData]
	refreshing typed.SyncMap[string, chan struct{}]
	opts       CacheOpts
}

type CacheOpts struct {
	MaxAge           time.Duration
	SWR              time.Duration
	CleanupInterval  time.Duration
	GetterFunc       GetterFunc
	RequestToKeyFunc RequestToKeyFunc
}

type ResponseData struct {
	Bytes     []byte
	ETag      string
	UpdatedAt time.Time
}

func NewCache(opts CacheOpts) *Cache {
	if opts.CleanupInterval == 0 {
		opts.CleanupInterval = getDefaultCleanupInterval(opts.SWR)
	}
	c := &Cache{opts: opts}
	go c.periodicCleanup()
	return c
}

func (c *Cache) Get(r *http.Request) (*ResponseData, error) {
	ctx := r.Context()

	res, err := c.loadOrInitCache(r)
	if err != nil {
		return nil, err
	}
	if time.Since(res.UpdatedAt) <= c.opts.MaxAge {
		return res, nil
	}

	key := c.opts.RequestToKeyFunc(r)
	if time.Since(res.UpdatedAt) <= c.opts.SWR {
		go func() {
			if _, refreshing := c.refreshing.LoadOrStore(key, make(chan struct{})); !refreshing {
				defer c.refreshing.Delete(key)
				_, err := c.refreshCache(r)
				if err != nil {
					log.Printf("Background refresh failed for key %s: %v", key, err)
				}
			}
		}()
		return res, nil
	}

	ch, refreshing := c.refreshing.LoadOrStore(key, make(chan struct{}))
	if !refreshing {
		defer c.refreshing.Delete(key)
		return c.refreshCache(r)
	}

	select {
	case <-ch:
		return c.loadOrInitCache(r)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func IsNotModified(r *http.Request, etag string) bool {
	match := r.Header.Get("If-None-Match")
	return match != "" && match == etag
}

func GenerateETag(bytes []byte) string {
	hash := sha256.Sum256(bytes)
	return fmt.Sprintf(`"%x"`, hash)
}

func (c *Cache) refreshCache(r *http.Request) (*ResponseData, error) {
	bytes, err := c.opts.GetterFunc(r)
	if err != nil {
		return nil, err
	}
	res := &ResponseData{
		Bytes:     bytes,
		ETag:      GenerateETag(bytes),
		UpdatedAt: time.Now(),
	}
	c.cache.Store(c.opts.RequestToKeyFunc(r), res)
	return res, nil
}

func (c *Cache) loadOrInitCache(r *http.Request) (*ResponseData, error) {
	key := c.opts.RequestToKeyFunc(r)
	res, ok := c.cache.Load(key)
	if !ok {
		return c.refreshCache(r)
	}
	return res, nil
}

func (c *Cache) periodicCleanup() {
	ticker := time.NewTicker(c.opts.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.cache.Range(func(key string, value *ResponseData) bool {
			if now.Sub(value.UpdatedAt) > c.opts.SWR {
				c.cache.Delete(key)
			}
			return true
		})
	}
}

func getDefaultCleanupInterval(swr time.Duration) time.Duration {
	defaultInterval := swr / 10
	if defaultInterval < time.Minute {
		return time.Minute
	}
	if defaultInterval > time.Hour {
		return time.Hour
	}
	return defaultInterval
}
