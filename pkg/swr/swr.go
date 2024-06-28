package swr

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type GetterFunc func(r *http.Request) ([]byte, error)
type RequestToKeyFunc func(r *http.Request) string

type Cache struct {
	cache            sync.Map
	MaxAge           time.Duration
	SWR              time.Duration
	GetterFunc       GetterFunc
	RequestToKeyFunc RequestToKeyFunc
}

type ResponseData struct {
	Bytes     []byte
	ETag      string
	UpdatedAt time.Time
}

func (c *Cache) Get(r *http.Request) (*ResponseData, error) {
	res, err := c.loadOrInitCache(r)
	if err != nil {
		return nil, err
	}

	if time.Since(res.UpdatedAt) <= c.MaxAge {
		return res, nil
	}

	// If within SWR window, start background refresh
	if time.Since(res.UpdatedAt) <= c.SWR {
		go func() {
			_, _ = c.refreshCache(r)
		}()
		return res, nil
	}

	// If past SWR deadline, wait for refresh
	return c.refreshCache(r)
}

func IsNotModified(r *http.Request, etag string) bool {
	match := r.Header.Get("If-None-Match")
	return match != "" && match == etag
}

func GenerateETag(bytes []byte) string {
	hash := md5.Sum(bytes)
	return fmt.Sprintf(`"%x"`, hash)
}

func (c *Cache) refreshCache(r *http.Request) (*ResponseData, error) {
	bytes, err := c.GetterFunc(r)
	if err != nil {
		return nil, err
	}
	res := &ResponseData{
		Bytes:     bytes,
		ETag:      GenerateETag(bytes),
		UpdatedAt: time.Now(),
	}
	c.cache.Store(c.RequestToKeyFunc(r), res)
	return res, nil
}

func (c *Cache) loadOrInitCache(r *http.Request) (*ResponseData, error) {
	key := c.RequestToKeyFunc(r)
	res, ok := c.cache.Load(key)
	if !ok {
		return c.refreshCache(r)
	}
	return res.(*ResponseData), nil
}
