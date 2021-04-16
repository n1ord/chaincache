package chaincache

import (
	"context"
	"sync/atomic"
	"time"

	redis "github.com/go-redis/redis/v8"
)

type RediscacherCfg struct {
	Hosts          []string `yaml:"hosts"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	MaxRetries     int      `yaml:"max_retries"`
	DialTimeoutMs  int64    `yaml:"dial_timeout_ms"`
	ReadTimeoutMs  int64    `yaml:"read_timeout_ms"`
	WriteTimeoutMs int64    `yaml:"write_timeout_ms"`
	PoolSize       int      `yaml:"pool_size"`
}

type Rediscacher struct {
	client *redis.ClusterClient
	cfg    *RediscacherCfg

	inited         bool
	ctx            context.Context
	hits           uint32
	misses         uint32
	requestTimeSum float64
	requestCount   uint32
}

func NewRediscacher(cfg *RediscacherCfg) (*Rediscacher, error) {
	c := &Rediscacher{}
	c.cfg = cfg

	if err := c.Init(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Rediscacher) Init() error {
	if c.inited {
		return nil
	}
	rc := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    c.cfg.Hosts,
		Username: c.cfg.Username,
		Password: c.cfg.Password,

		MaxRetries:   c.cfg.MaxRetries,
		DialTimeout:  time.Duration(c.cfg.DialTimeoutMs) * time.Millisecond,
		ReadTimeout:  time.Duration(c.cfg.ReadTimeoutMs) * time.Millisecond,
		WriteTimeout: time.Duration(c.cfg.WriteTimeoutMs) * time.Millisecond,

		PoolSize: c.cfg.PoolSize,
	})
	c.ctx = context.Background()
	c.client = rc

	err := c.client.Ping(c.ctx).Err()
	if err != nil {
		return err
	}
	c.inited = true
	return nil
}

func (c *Rediscacher) Set(key string, payload []byte, ttlSeconds int) error {
	if !c.inited {
		return ErrNotInited
	}
	start := time.Now()
	err := c.client.Set(c.ctx, key, payload, time.Duration(ttlSeconds*int(time.Second))).Err()
	c.requestCount += 1
	c.requestTimeSum += time.Since(start).Seconds()
	if err != nil {
		return err
	}
	return nil
}

func (c *Rediscacher) Get(key string) ([]byte, error) {
	if !c.inited {
		return nil, ErrNotInited
	}

	start := time.Now()
	cmd := c.client.Get(c.ctx, key)
	c.requestCount += 1
	c.requestTimeSum += time.Since(start).Seconds()
	res, err := cmd.Bytes()
	if err != nil {
		if err == redis.Nil {
			atomic.AddUint32(&c.misses, 1)
			return nil, ErrMiss
		}
		return nil, err
	}
	atomic.AddUint32(&c.hits, 1)
	return res, nil
}

func (c *Rediscacher) GetWithTTL(key string) ([]byte, int, error) {
	if !c.inited {
		return nil, 0, ErrNotInited
	}
	start := time.Now()
	res, err := c.client.Get(c.ctx, key).Bytes()
	c.requestCount += 1
	c.requestTimeSum += time.Since(start).Seconds()
	if err != nil {
		if err == redis.Nil {
			atomic.AddUint32(&c.misses, 1)
			return nil, 0, ErrMiss
		}
		return nil, 0, err
	}

	start = time.Now()
	cmd := c.client.TTL(c.ctx, key)
	c.requestCount += 1
	c.requestTimeSum += time.Since(start).Seconds()
	if cmd.Err() != nil {
		return nil, 0, cmd.Err()
	}
	ttl := int(cmd.Val().Seconds())

	atomic.AddUint32(&c.hits, 1)
	return res, ttl, nil
}

func (c *Rediscacher) Del(key string) error {
	if !c.inited {
		return ErrNotInited
	}

	res, err := c.client.Del(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrMiss
		}
		return err
	}
	if res == 0 {
		return ErrMiss
	}

	return nil
}

func (c *Rediscacher) GetHits() uint32 {
	return atomic.LoadUint32(&c.hits)
}

func (c *Rediscacher) GetMisses() uint32 {
	return atomic.LoadUint32(&c.misses)
}

func (c *Rediscacher) Close() {
	if !c.inited {
		return
	}
	c.client.Close()
	c.inited = false
}

func (c *Rediscacher) GetAvgRequestTime() float64 {
	if c.requestCount == 0 {
		return 0.
	}
	return c.requestTimeSum / float64(c.requestCount)
}
