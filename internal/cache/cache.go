package cache

import (
	"sync"
)

type MediaType int

const (
	TypeDocument MediaType = iota
	TypePhoto
)

type CachedMedia struct {
	ID            int64
	AccessHash    int64
	FileReference []byte
	Type          MediaType
	Title         string
	Size          int64
	Provider      string
}

type Cache struct {
	data map[string]*CachedMedia
	mu   sync.RWMutex
}

var instance *Cache
var once sync.Once

func GetInstance() *Cache {
	once.Do(func() {
		instance = &Cache{
			data: make(map[string]*CachedMedia),
		}
	})
	return instance
}

func (c *Cache) Get(key string) *CachedMedia {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data[key]
}

func (c *Cache) Set(key string, media *CachedMedia) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = media
}
