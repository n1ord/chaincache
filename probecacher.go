package chaincache

import (
	"fmt"
	"sync/atomic"

	"github.com/n1ord/probecache"
)

type StorageStrategy uint8

const (
	STORAGE_LRU StorageStrategy = iota
	STORAGE_LFU StorageStrategy = iota
)

type Probecacher struct {
	inited bool
	cache  probecache.IStorage
	hits   uint32
	misses uint32
}

func NewProbecacher(shards int, maxSize int, maxCritSize int, maxDepth int, strategy StorageStrategy) (*Probecacher, error) {
	var (
		c   probecache.IStorage
		err error
	)
	if strategy == STORAGE_LFU {
		c, err = probecache.NewLFUStorage(shards, maxSize, maxCritSize, maxDepth)
	} else {
		c, err = probecache.NewLRUStorage(shards, maxSize, maxCritSize, maxDepth)
	}
	if err != nil {
		return nil, err
	}

	pc := &Probecacher{
		cache: c,
	}

	if err := pc.Init(); err != nil {
		return nil, err
	}

	return pc, nil
}

func (c *Probecacher) Init() error {
	if c.inited {
		return nil
	}
	atomic.StoreUint32(&c.hits, 0)
	atomic.StoreUint32(&c.misses, 0)
	c.inited = true
	return nil
}

func (c *Probecacher) GetWithTTL(key string) ([]byte, int, error) {
	if !c.inited {
		return nil, 0, ErrNotInited
	}
	// log.Printf("Freecache: get %s", key)
	value, ttl, err := c.cache.GetWithTTL(key)
	if err != nil {
		if err == probecache.ErrMissing {
			atomic.AddUint32(&c.misses, 1)
			return nil, 0, ErrMiss
		}
		return nil, 0, fmt.Errorf("internal cache error: %s", err)
	}
	atomic.AddUint32(&c.hits, 1)
	return value, int(ttl), nil
}

func (c *Probecacher) Get(key string) ([]byte, error) {
	if !c.inited {
		return nil, ErrNotInited
	}
	val, _, err := c.GetWithTTL(key)
	return val, err
}

func (c *Probecacher) Set(key string, payload []byte, ttlSeconds int) error {
	if !c.inited {
		return ErrNotInited
	}
	// log.Printf("Freecache: set %s", key)
	return c.cache.Set(key, payload, uint64(ttlSeconds))
}

func (c *Probecacher) Del(key string) error {
	if !c.inited {
		return ErrNotInited
	}
	c.cache.Del(key)

	return nil
}

func (c *Probecacher) BGetWithTTL(key []byte) ([]byte, int, error) {
	if !c.inited {
		return nil, 0, ErrNotInited
	}
	// log.Printf("Freecache: get %s", key)
	value, ttl, err := c.cache.GetWithTTL(string(key))
	if err != nil {
		if err == probecache.ErrMissing {
			atomic.AddUint32(&c.misses, 1)
			return nil, 0, ErrMiss
		}
		return nil, 0, fmt.Errorf("internal cache error: %s", err)
	}
	atomic.AddUint32(&c.hits, 1)
	return value, int(ttl), nil
}

func (c *Probecacher) BGet(key []byte) ([]byte, error) {
	if !c.inited {
		return nil, ErrNotInited
	}
	val, _, err := c.GetWithTTL(string(key))
	return val, err
}

func (c *Probecacher) BSet(key []byte, payload []byte, ttlSeconds int) error {
	if !c.inited {
		return ErrNotInited
	}
	// log.Printf("Freecache: set %s", key)
	return c.cache.Set(string(key), payload, uint64(ttlSeconds))
}

func (c *Probecacher) BDel(key []byte) error {
	if !c.inited {
		return ErrNotInited
	}
	c.cache.Del(string(key))

	return nil
}

func (c *Probecacher) Close() {
	if !c.inited {
		return
	}
	c.inited = false
}

func (c *Probecacher) Reset() {
	c.cache.Clear()
}

func (c *Probecacher) GetHits() uint32 {
	return atomic.LoadUint32(&c.hits)
}

func (c *Probecacher) GetMisses() uint32 {
	return atomic.LoadUint32(&c.misses)
}

// ------------------------------------------------------------------------------------------------
