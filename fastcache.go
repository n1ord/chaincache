package chaincache

import (
	"encoding/binary"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/fastcache"
)

type Fastcacher struct {
	MaxSize       int
	UseTTL        bool
	waitBigValues bool

	inited       bool
	ttlKeySuffix []byte
	cache        *fastcache.Cache
	hits         uint32
	misses       uint32
}

func NewFastCacher(maxSize int, useTTL bool, waitBigValues bool) (*Fastcacher, error) {
	c := &Fastcacher{
		MaxSize:       maxSize,
		UseTTL:        useTTL,
		ttlKeySuffix:  []byte("@"),
		waitBigValues: waitBigValues,
	}
	if err := c.Init(); err != nil {
		return nil, err
	}

	return c, nil
}

func NewFastCacherFromInstance(cache *fastcache.Cache, useTTL bool, waitBigValues bool) (*Fastcacher, error) {
	c := &Fastcacher{
		cache:         cache,
		inited:        true,
		MaxSize:       0,
		UseTTL:        useTTL,
		waitBigValues: waitBigValues,
		ttlKeySuffix:  []byte("@"),
	}
	atomic.StoreUint32(&c.hits, 0)
	atomic.StoreUint32(&c.misses, 0)
	return c, nil
}

func (c *Fastcacher) Init() error {
	if c.inited {
		return nil
	}
	c.inited = true
	c.cache = fastcache.New(c.MaxSize)
	atomic.StoreUint32(&c.hits, 0)
	atomic.StoreUint32(&c.misses, 0)
	return nil
}

func (c *Fastcacher) GetWithTTL(key string) ([]byte, int, error) {
	if !c.inited {
		return nil, 0, ErrNotInited
	}
	k := []byte(key)
	var ret []byte
	if c.waitBigValues {
		ret = c.cache.GetBig(nil, k)
	} else {
		ret = c.cache.Get(nil, k)
	}
	if ret == nil {
		atomic.AddUint32(&c.misses, 1)
		return nil, 0, ErrMiss
	}

	if c.UseTTL {
		ttlBytes := ret[len(ret)-8:]
		ret := ret[:len(ret)-8]
		ttl := int64(binary.LittleEndian.Uint64(ttlBytes)) - time.Now().Unix()
		if ttl <= 0 {
			atomic.AddUint32(&c.misses, 1)
			c.cache.Del(k)
			return nil, 0, ErrMiss
		}

		atomic.AddUint32(&c.hits, 1)
		return ret, int(ttl), nil
	}
	atomic.AddUint32(&c.hits, 1)
	return ret, 0, nil
}

func (c *Fastcacher) Get(key string) ([]byte, error) {
	if !c.inited {
		return nil, ErrNotInited
	}
	val, _, err := c.GetWithTTL(key)
	return val, err
}

func (c *Fastcacher) Set(key string, payload []byte, ttlSeconds int) error {
	if !c.inited {
		return ErrNotInited
	}

	if c.UseTTL {
		tsBytes := make([]byte, 8)
		ts := time.Now().Unix() + int64(ttlSeconds)
		binary.LittleEndian.PutUint64(tsBytes, uint64(ts))
		payload = append(payload, tsBytes...)
	}
	if c.waitBigValues {
		c.cache.SetBig([]byte(key), payload)
	} else {
		c.cache.Set([]byte(key), payload)
	}

	return nil
}

func (c *Fastcacher) Del(key string) error {
	if !c.inited {
		return ErrNotInited
	}
	c.cache.Del([]byte(key))
	return nil
}

func (c *Fastcacher) BGetWithTTL(key []byte) ([]byte, int, error) {
	if !c.inited {
		return nil, 0, ErrNotInited
	}

	var ret []byte
	if c.waitBigValues {
		ret = c.cache.GetBig(nil, key)
	} else {
		ret = c.cache.Get(nil, key)
	}
	if ret == nil {
		atomic.AddUint32(&c.misses, 1)
		return nil, 0, ErrMiss
	}

	if c.UseTTL {
		ttlBytes := ret[len(ret)-8:]
		ret := ret[:len(ret)-8]
		ttl := int64(binary.LittleEndian.Uint64(ttlBytes)) - time.Now().Unix()
		if ttl <= 0 {
			atomic.AddUint32(&c.misses, 1)
			c.cache.Del(key)
			return nil, 0, ErrMiss
		}

		atomic.AddUint32(&c.hits, 1)
		return ret, int(ttl), nil
	}
	atomic.AddUint32(&c.hits, 1)

	return ret, 0, nil
}

func (c *Fastcacher) BGet(key []byte) ([]byte, error) {
	if !c.inited {
		return nil, ErrNotInited
	}
	val, _, err := c.BGetWithTTL(key)
	return val, err
}

func (c *Fastcacher) BSet(key []byte, payload []byte, ttlSeconds int) error {
	if !c.inited {
		return ErrNotInited
	}

	if c.UseTTL {
		tsBytes := make([]byte, 8)
		ts := time.Now().Unix() + int64(ttlSeconds)
		binary.LittleEndian.PutUint64(tsBytes, uint64(ts))
		payload = append(payload, tsBytes...)
	}
	if c.waitBigValues {
		c.cache.SetBig(key, payload)
	} else {
		c.cache.Set(key, payload)
	}
	return nil
}

func (c *Fastcacher) BDel(key []byte) error {
	if !c.inited {
		return ErrNotInited
	}
	c.cache.Del(key)
	return nil
}

func (c *Fastcacher) Close() {
	if !c.inited {
		return
	}
	c.inited = false
}

func (c *Fastcacher) Reset() {
	c.cache.Reset()
	atomic.StoreUint32(&c.hits, 0)
	atomic.StoreUint32(&c.misses, 0)
}

func (c *Fastcacher) GetHits() uint32 {
	return atomic.LoadUint32(&c.hits)
}

func (c *Fastcacher) GetMisses() uint32 {
	return atomic.LoadUint32(&c.misses)
}

// ------------------------------------------------------------------------------------------------
