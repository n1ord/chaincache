package chaincache

import (
	"fmt"
	"time"

	"github.com/coocood/freecache"
)

type Freecacher struct {
	MaxSize int

	inited bool
	cache  *freecache.Cache
}

func NewFreeCacher(maxSize int) (*Freecacher, error) {
	c := &Freecacher{
		MaxSize: maxSize,
	}
	if err := c.Init(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Freecacher) Init() error {
	if c.inited {
		return nil
	}
	c.inited = true
	c.cache = freecache.NewCache(c.MaxSize)
	return nil
}

func (c *Freecacher) GetWithTTL(key string) ([]byte, int, error) {
	if !c.inited {
		return nil, 0, ErrNotInited
	}
	// log.Printf("Freecache: get %s", key)
	value, expiresAt, err := c.cache.GetWithExpiration([]byte(key))
	if err != nil {
		if err == freecache.ErrNotFound {
			return nil, 0, ErrMiss
		}
		return nil, 0, fmt.Errorf("internal cache error: %s", err)
	}
	ttl := int64(expiresAt) - time.Now().Unix()
	if ttl <= 0 {
		return nil, 0, ErrMiss
	}
	return value, int(ttl), nil
}

func (c *Freecacher) Get(key string) ([]byte, error) {
	if !c.inited {
		return nil, ErrNotInited
	}
	val, _, err := c.GetWithTTL(key)
	return val, err
}

func (c *Freecacher) Set(key string, payload []byte, ttlSeconds int) error {
	if !c.inited {
		return ErrNotInited
	}
	// log.Printf("Freecache: set %s", key)
	return c.cache.Set([]byte(key), payload, ttlSeconds)
}

func (c *Freecacher) Del(key string) error {
	if !c.inited {
		return ErrNotInited
	}
	affected := c.cache.Del([]byte(key))
	if !affected {
		return ErrMiss
	}
	return nil
}

func (c *Freecacher) Close() {
	if !c.inited {
		return
	}
	c.inited = false
}

func (c *Freecacher) GetHits() uint32 {
	return uint32(c.cache.HitCount())
}

func (c *Freecacher) GetMisses() uint32 {
	return uint32(c.cache.MissCount())
}

// ------------------------------------------------------------------------------------------------
