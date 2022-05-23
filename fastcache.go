package chaincache

import (
	"bytes"
	"encoding/binary"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/fastcache"
)

type Fastcacher struct {
	MaxSize int
	UseTTL  bool

	inited       bool
	ttlKeySuffix []byte
	cache        *fastcache.Cache
	hits         uint32
	misses       uint32
}

func NewFastCacher(maxSize int, useTTL bool) (*Fastcacher, error) {
	c := &Fastcacher{
		MaxSize:      maxSize,
		UseTTL:       useTTL,
		ttlKeySuffix: []byte("@"),
	}
	if err := c.Init(); err != nil {
		return nil, err
	}

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

	ret := c.cache.Get(nil, []byte(key))
	if ret == nil {
		atomic.AddUint32(&c.misses, 1)
		return nil, 0, ErrMiss
	}

	if c.UseTTL {
		ttlBytes := c.cache.Get(nil, bytes.Join([][]byte{[]byte(key), c.ttlKeySuffix}, nil))
		if ttlBytes == nil {
			atomic.AddUint32(&c.misses, 1)
			return nil, 0, ErrMiss
		}
		ttl := int64(binary.LittleEndian.Uint64(ttlBytes)) - time.Now().Unix()
		if ttl <= 0 {
			atomic.AddUint32(&c.misses, 1)
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

	c.cache.Set([]byte(key), payload)
	if c.UseTTL {
		ts := time.Now().Unix() + int64(ttlSeconds)
		tsBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(tsBytes, uint64(ts))
		c.cache.Set(bytes.Join([][]byte{[]byte(key), c.ttlKeySuffix}, nil), tsBytes)
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

func (c *Fastcacher) Close() {
	if !c.inited {
		return
	}
	c.inited = false
}

func (c *Fastcacher) GetHits() uint32 {
	return atomic.LoadUint32(&c.hits)
}

func (c *Fastcacher) GetMisses() uint32 {
	return atomic.LoadUint32(&c.misses)
}

// ------------------------------------------------------------------------------------------------
