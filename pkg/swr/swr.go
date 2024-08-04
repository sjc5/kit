package swr

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sjc5/kit/pkg/lru"
)

type isSpam = bool
type GetterFunc func(r *http.Request) ([]byte, isSpam, error)
type RequestToKeyFunc func(r *http.Request) string

type Cache struct {
	cache      *lru.Cache[string, *ResponseData]
	refreshing *lru.Cache[string, chan struct{}]
	opts       CacheOpts
}

type CacheOpts struct {
	MaxAge     time.Duration
	SWR        time.Duration
	MaxItems   int
	GetterFunc GetterFunc

	// RequestToKeyFunc default returns r.URL.Path
	// If you want to take into account search params, pass a custom func
	RequestToKeyFunc RequestToKeyFunc
}

type ResponseData struct {
	Bytes     []byte
	ETag      string
	UpdatedAt time.Time
}

func NewCache(opts CacheOpts) *Cache {
	if opts.RequestToKeyFunc == nil {
		opts.RequestToKeyFunc = func(r *http.Request) string {
			return r.URL.Path
		}
	}
	return &Cache{
		cache:      lru.NewCache[string, *ResponseData](opts.MaxItems),
		refreshing: lru.NewCache[string, chan struct{}](opts.MaxItems), // Initialize LRU for refreshing
		opts:       opts,
	}
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
		go c.refreshInBackground(key, r)
		return res, nil
	}

	ch, refreshing := c.loadOrStoreRefreshing(key)
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
	bytes, isSpam, err := c.opts.GetterFunc(r)
	if err != nil {
		return nil, err
	}
	res := &ResponseData{
		Bytes:     bytes,
		ETag:      GenerateETag(bytes),
		UpdatedAt: time.Now(),
	}
	c.cache.Set(c.opts.RequestToKeyFunc(r), res, isSpam)
	return res, nil
}

func (c *Cache) loadOrInitCache(r *http.Request) (*ResponseData, error) {
	key := c.opts.RequestToKeyFunc(r)
	res, ok := c.cache.Get(key)
	if !ok {
		return c.refreshCache(r)
	}
	return res, nil
}

func (c *Cache) refreshInBackground(key string, r *http.Request) {
	ch, refreshing := c.loadOrStoreRefreshing(key)
	if !refreshing {
		defer c.refreshing.Delete(key)
		_, err := c.refreshCache(r)
		if err != nil {
			log.Printf("Background refresh failed for key %s: %v", key, err)
		}
		close(ch)
	}
}

func (c *Cache) loadOrStoreRefreshing(key string) (chan struct{}, bool) {
	ch := make(chan struct{})
	actual, loaded := c.refreshing.Get(key)
	if loaded {
		return actual, true
	}
	c.refreshing.Set(key, ch, false)
	return ch, false
}
