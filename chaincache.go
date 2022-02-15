package chaincache

import (
	"fmt"
	"sync/atomic"
)

type Cacher interface {
	Init() error
	Close()
	GetHits() uint32
	GetMisses() uint32

	Get(key string) ([]byte, error)
	GetWithTTL(key string) ([]byte, int, error)
	Set(key string, payload []byte, ttl int) error
	Del(key string) error
}

var (
	ErrMiss      = fmt.Errorf("key missed in cache")
	ErrNotInited = fmt.Errorf("cacher has not been inited")
)

// ------------------------------------------------------------------------------------------------

type ChainCache struct {
	chain []Cacher
	// Auto store found data to all cachers to the left side with the rest of data TTL, default=false
	NoBackwardCache bool

	// All internal errors will be interpreted as ErrMiss
	IgnoreErrors bool

	inited bool
	hits   uint32
	misses uint32
}

func NewChainCache(cachers ...Cacher) (*ChainCache, error) {
	c := &ChainCache{
		chain: cachers,
	}
	atomic.StoreUint32(&c.hits, 0)
	atomic.StoreUint32(&c.misses, 0)
	err := c.Init()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *ChainCache) Init() error {
	if c.inited {
		return nil
	}
	if len(c.chain) == 0 {
		return fmt.Errorf("cannot init ChainCache without cachers")
	}
	for _, cacher := range c.chain {
		if err := cacher.Init(); err != nil {
			return err
		}
	}
	c.inited = true
	return nil
}

func (c *ChainCache) Get(key string) ([]byte, error) {
	var (
		val []byte
		ix  int
		ttl int
		err error
	)
	if !c.inited {
		return nil, ErrNotInited
	}

	for ix = 0; ix < len(c.chain); ix++ {
		cacher := c.chain[ix]
		if !c.NoBackwardCache {
			val, ttl, err = cacher.GetWithTTL(key)
		} else {
			val, err = cacher.Get(key)
		}

		if err == nil {
			atomic.AddUint32(&c.hits, 1)
			break
		}
		if err != ErrMiss && c.IgnoreErrors {
			err = ErrMiss
		}
		if err != ErrMiss {
			return nil, err
		}
	}

	if err == ErrMiss {
		atomic.AddUint32(&c.misses, 1)
		return nil, err
	}

	if !c.NoBackwardCache {
		for ix -= 1; ix >= 0; ix-- {
			cacher := c.chain[ix]
			if err = cacher.Set(key, val, ttl); err != nil {
				if !c.IgnoreErrors {
					return nil, err
				}
			}
		}
	}

	return val, nil
}

func (c *ChainCache) Set(key string, payload []byte, ttlSeconds []int) error {
	if !c.inited {
		return ErrNotInited
	}
	if len(ttlSeconds) != len(c.chain) {
		return fmt.Errorf("ttl slice size must be equal to your chain size")
	}
	for ix := 0; ix < len(c.chain); ix++ {
		cacher := c.chain[ix]
		if err := cacher.Set(key, payload, ttlSeconds[ix]); err != nil {
			if !c.IgnoreErrors {
				return err
			}
		}
	}
	return nil
}

func (c *ChainCache) Del(key string) error {
	if !c.inited {
		return ErrNotInited
	}
	for _, cacher := range c.chain {
		if err := cacher.Del(key); err != nil && err != ErrMiss {
			if !c.IgnoreErrors {
				return err
			}
		}
	}
	return nil
}

func (c *ChainCache) Close() {
	if !c.inited {
		return
	}
	for _, cacher := range c.chain {
		cacher.Close()
	}
	c.inited = false
}

func (c *ChainCache) GetHits() uint32 {
	return atomic.LoadUint32(&c.hits)
}

func (c *ChainCache) GetMisses() uint32 {
	return atomic.LoadUint32(&c.misses)
}
