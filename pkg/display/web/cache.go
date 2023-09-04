package web

import "sync"

type cacheEntry struct {
	hash uint64
	data []byte
}

type cache struct {
	cache   []*cacheEntry
	idx     int
	enabled bool
	size    int
	sync.RWMutex
}

func newCache(size int) *cache {
	c := &cache{
		cache:   make([]*cacheEntry, size),
		size:    size,
		enabled: true,
	}
	for i := 0; i < size; i++ {
		c.cache[i] = &cacheEntry{
			hash: 0,
			data: []byte{},
		}
	}

	return c
}

func (c *cache) has(hash uint64) bool {
	if !c.enabled {
		return false
	}

	for _, e := range c.cache {
		if e.hash == hash {
			return true
		}
	}

	return false
}

func (c *cache) add(hash uint64, output []byte) {
	c.cache[c.idx].data = output
	c.cache[c.idx].hash = hash

	c.idx = (c.idx + 1) % c.size
}

func (c *cache) index(hash uint64) int {
	for i, e := range c.cache {
		if e.hash == hash {
			return i
		}
	}

	return -1
}
