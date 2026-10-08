package probe

import (
	"maps"
	"sync"
	"time"
)

type ProbeResult struct {
	Online        bool
	Players       int
	Timestamp     time.Time
	Err           error
	TotalDuration time.Duration
	RTTDuration   time.Duration
}

type Cache struct {
	mu     sync.RWMutex
	status map[string]ProbeResult
}

func NewCache() *Cache {
	return &Cache{
		status: make(map[string]ProbeResult),
	}
}
func (c *Cache) Set(name string, res ProbeResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status[name] = res
}
func (c *Cache) Snapshot() map[string]ProbeResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return maps.Clone(c.status)
}
func (c *Cache) Delete(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.status, name)
}
