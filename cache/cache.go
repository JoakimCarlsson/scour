package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/JoakimCarlsson/scour/query"
	"github.com/JoakimCarlsson/scour/rank"
)

type Cache interface {
	Get(key string) ([]rank.Ranked, bool)
	Set(key string, val []rank.Ranked, ttl time.Duration)
}

func KeyFor(q query.Query) string {
	engs := append([]string(nil), q.Engines...)
	sort.Strings(engs)
	canonical := strings.Join([]string{
		strings.ToLower(strings.TrimSpace(q.Terms)),
		string(q.Category),
		strings.ToLower(q.Language),
		q.SafeSearch.String(),
		strings.Join(engs, ","),
	}, "\x00")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

type entry struct {
	val     []rank.Ranked
	expires time.Time
}

type MemoryCache struct {
	mu      sync.RWMutex
	data    map[string]entry
	stop    chan struct{}
	stopped bool
}

func NewMemory(sweepInterval time.Duration) *MemoryCache {
	c := &MemoryCache{
		data: map[string]entry{},
		stop: make(chan struct{}),
	}
	if sweepInterval > 0 {
		go c.sweep(sweepInterval)
	}
	return c
}

func (c *MemoryCache) Get(key string) ([]rank.Ranked, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expires) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return nil, false
	}
	return e.val, true
}

func (c *MemoryCache) Set(key string, val []rank.Ranked, ttl time.Duration) {
	c.mu.Lock()
	c.data[key] = entry{val: val, expires: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *MemoryCache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return
	}
	c.stopped = true
	close(c.stop)
}

func (c *MemoryCache) sweep(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-t.C:
			now := time.Now()
			c.mu.Lock()
			for k, e := range c.data {
				if now.After(e.expires) {
					delete(c.data, k)
				}
			}
			c.mu.Unlock()
		}
	}
}
