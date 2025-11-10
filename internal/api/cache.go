package api

import (
	"sync"
	"time"

	"github.com/h4sh5/sdlookup/internal/models"
)

// LRUCache implements a simple LRU cache with TTL
type LRUCache struct {
	mu       sync.RWMutex
	items    map[string]*cacheItem
	maxSize  int
	ttl      time.Duration
	keyOrder []string
}

type cacheItem struct {
	value     *models.ShodanIPInfo
	expiresAt time.Time
}

// NewLRUCache creates a new LRU cache
func NewLRUCache(maxSize int, ttl time.Duration) *LRUCache {
	return &LRUCache{
		items:    make(map[string]*cacheItem),
		maxSize:  maxSize,
		ttl:      ttl,
		keyOrder: make([]string, 0, maxSize),
	}
}

// Get retrieves a value from the cache
func (c *LRUCache) Get(key string) (*models.ShodanIPInfo, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.removeFromOrder(key)
		c.mu.Unlock()
		return nil, false
	}

	// Move to front (most recently used)
	c.mu.Lock()
	c.moveToFront(key)
	c.mu.Unlock()

	return item.value, true
}

// Set adds a value to the cache
func (c *LRUCache) Set(key string, value *models.ShodanIPInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already exists
	if _, exists := c.items[key]; exists {
		c.items[key] = &cacheItem{
			value:     value,
			expiresAt: time.Now().Add(c.ttl),
		}
		c.moveToFront(key)
		return
	}

	// Evict oldest if at capacity
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.keyOrder = append([]string{key}, c.keyOrder...)
}

// moveToFront moves a key to the front of the order list
func (c *LRUCache) moveToFront(key string) {
	c.removeFromOrder(key)
	c.keyOrder = append([]string{key}, c.keyOrder...)
}

// removeFromOrder removes a key from the order list
func (c *LRUCache) removeFromOrder(key string) {
	for i, k := range c.keyOrder {
		if k == key {
			c.keyOrder = append(c.keyOrder[:i], c.keyOrder[i+1:]...)
			return
		}
	}
}

// evictOldest removes the least recently used item
func (c *LRUCache) evictOldest() {
	if len(c.keyOrder) == 0 {
		return
	}

	oldest := c.keyOrder[len(c.keyOrder)-1]
	delete(c.items, oldest)
	c.keyOrder = c.keyOrder[:len(c.keyOrder)-1]
}

// Size returns the current number of items in the cache
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all items from the cache
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*cacheItem)
	c.keyOrder = make([]string, 0, c.maxSize)
}
