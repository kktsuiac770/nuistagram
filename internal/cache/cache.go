package cache

import (
	"sync"
	"time"
)

type entry struct {
	value      interface{}
	expiration time.Time
}

type Cache struct {
	items sync.Map
	ttl   time.Duration
}

func New(ttl time.Duration) *Cache {
	c := &Cache{
		ttl: ttl,
	}
	go c.cleanup()
	return c
}

func (c *Cache) Get(key string) (interface{}, bool) {
	val, ok := c.items.Load(key)
	if !ok {
		return nil, false
	}
	e := val.(*entry)
	if time.Now().After(e.expiration) {
		c.items.Delete(key)
		return nil, false
	}
	return e.value, true
}

func (c *Cache) Set(key string, value interface{}) {
	c.items.Store(key, &entry{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	})
}

func (c *Cache) Delete(key string) {
	c.items.Delete(key)
}

func (c *Cache) DeleteByPrefix(prefix string) {
	c.items.Range(func(key, value interface{}) bool {
		if k, ok := key.(string); ok {
			if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
				c.items.Delete(key)
			}
		}
		return true
	})
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		now := time.Now()
		c.items.Range(func(key, value interface{}) bool {
			if e, ok := value.(*entry); ok {
				if now.After(e.expiration) {
					c.items.Delete(key)
				}
			}
			return true
		})
	}
}
