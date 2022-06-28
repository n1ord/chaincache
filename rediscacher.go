package chaincache

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	redis "github.com/go-redis/redis/v8"
)

type RedisClientIface interface {
	Ping(ctx context.Context) *redis.StatusCmd
	Set(context.Context, string, interface{}, time.Duration) *redis.StatusCmd
	Get(context.Context, string) *redis.StringCmd
	TTL(context.Context, string) *redis.DurationCmd
	Del(context.Context, ...string) *redis.IntCmd
	Close() error
}

type RediscacherCfg struct {
	Hosts          []string `yaml:"hosts"`
	Host           string   `yaml:"host"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	MaxRetries     int      `yaml:"max_retries"`
	DialTimeoutMs  int64    `yaml:"dial_timeout_ms"`
	ReadTimeoutMs  int64    `yaml:"read_timeout_ms"`
	WriteTimeoutMs int64    `yaml:"write_timeout_ms"`
	PoolSize       int      `yaml:"pool_size"`
	ClusterMode    bool     `yaml:"cluster_mode"`
}

type Rediscacher struct {
	client RedisClientIface
	cfg    *RediscacherCfg

	inited         bool
	ctx            context.Context
	hits           uint32
	misses         uint32
	requestTimeSum float64
	requestCount   uint32
}

func (c *Rediscacher) newRedisClient(cfg *RediscacherCfg) RedisClientIface {
	return redis.NewClient(&redis.Options{
		Addr:     c.cfg.Host,
		Username: c.cfg.Username,
		Password: c.cfg.Password,

		MaxRetries:   c.cfg.MaxRetries,
		DialTimeout:  time.Duration(c.cfg.DialTimeoutMs) * time.Millisecond,
		ReadTimeout:  time.Duration(c.cfg.ReadTimeoutMs) * time.Millisecond,
		WriteTimeout: time.Duration(c.cfg.WriteTimeoutMs) * time.Millisecond,

		PoolSize: c.cfg.PoolSize,
	})
}

func (c *Rediscacher) newRedisClusterClient(cfg *RediscacherCfg) RedisClientIface {
	return redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    c.cfg.Hosts,
		Username: c.cfg.Username,
		Password: c.cfg.Password,

		MaxRetries:   c.cfg.MaxRetries,
		DialTimeout:  time.Duration(c.cfg.DialTimeoutMs) * time.Millisecond,
		ReadTimeout:  time.Duration(c.cfg.ReadTimeoutMs) * time.Millisecond,
		WriteTimeout: time.Duration(c.cfg.WriteTimeoutMs) * time.Millisecond,

		PoolSize: c.cfg.PoolSize,
	})
}

func NewRediscacher(cfg *RediscacherCfg) (*Rediscacher, error) {
	c := &Rediscacher{}
	c.cfg = cfg

	if err := c.Init(); err != nil {
		return nil, err
	}

	return c, nil
}

func NewRediccacherWithClient(client RedisClientIface) (*Rediscacher, error) {
	c := &Rediscacher{}
	c.client = client
	c.cfg = nil

	c.ctx = context.Background()

	err := c.client.Ping(c.ctx).Err()
	if err != nil {
		return nil, err
	}
	c.inited = true
	return c, nil
}

func (c *Rediscacher) Init() error {
	if c.inited {
		return nil
	}

	if len(c.cfg.Host) == 0 && len(c.cfg.Hosts) == 0 {
		return errors.New("no one redis host is defined")
	}

	if c.cfg.ClusterMode && len(c.cfg.Hosts) == 0 {
		c.cfg.Hosts = []string{c.cfg.Host}
	}

	if !c.cfg.ClusterMode && len(c.cfg.Host) == 0 {
		c.cfg.Host = c.cfg.Hosts[0]
	}

	if c.cfg.ClusterMode {
		c.client = c.newRedisClusterClient(c.cfg)
	} else {
		c.client = c.newRedisClient(c.cfg)
	}

	c.ctx = context.Background()

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

func (c *Rediscacher) BSet(key []byte, payload []byte, ttlSeconds int) error {
	if !c.inited {
		return ErrNotInited
	}
	start := time.Now()
	err := c.client.Set(c.ctx, string(key), payload, time.Duration(ttlSeconds*int(time.Second))).Err()
	c.requestCount += 1
	c.requestTimeSum += time.Since(start).Seconds()
	if err != nil {
		return err
	}
	return nil
}

func (c *Rediscacher) BGet(key []byte) ([]byte, error) {
	if !c.inited {
		return nil, ErrNotInited
	}

	start := time.Now()
	cmd := c.client.Get(c.ctx, string(key))
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

func (c *Rediscacher) BGetWithTTL(key []byte) ([]byte, int, error) {
	if !c.inited {
		return nil, 0, ErrNotInited
	}
	skey := string(key)
	start := time.Now()
	res, err := c.client.Get(c.ctx, skey).Bytes()
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
	cmd := c.client.TTL(c.ctx, skey)
	c.requestCount += 1
	c.requestTimeSum += time.Since(start).Seconds()
	if cmd.Err() != nil {
		return nil, 0, cmd.Err()
	}
	ttl := int(cmd.Val().Seconds())

	atomic.AddUint32(&c.hits, 1)
	return res, ttl, nil
}

func (c *Rediscacher) BDel(key []byte) error {
	if !c.inited {
		return ErrNotInited
	}

	res, err := c.client.Del(c.ctx, string(key)).Result()
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
