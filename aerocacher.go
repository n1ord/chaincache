package chaincache

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	aero "github.com/aerospike/aerospike-client-go"
)

type AerocacherCfg struct {
	Hosts []string `yaml:"hosts"`

	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Namespace string `yaml:"namespace"`
	SetName   string `yaml:"set"`
	BinName   string `yaml:"bin"`

	ConnectTimeoutMs int64 `yaml:"connect_timeout_ms"` //=30 sec
	IdleTimeoutMs    int64 `yaml:"idle_timeout_ms"`    //=55 sec
	LoginTimeoutMs   int64 `yaml:"login_timeout_ms"`   //=10 sec

	ConnectionQueueSize        int `yaml:"connection_queue_size"`        //=256
	OpeningConnectionThreshold int `yaml:"opening_connection_threshold"` //=0
	MinConnectionsPerNode      int `yaml:"min_connections_per_node"`     //=0
}

type Aerocacher struct {
	cfg    *AerocacherCfg
	client *aero.Client

	inited         bool
	hits           uint32
	misses         uint32
	requestTimeSum float64
	requestCount   uint32
}

func NewAerocacher(cfg *AerocacherCfg) (*Aerocacher, error) {
	aerocacher := &Aerocacher{
		cfg: cfg,
	}
	atomic.StoreUint32(&aerocacher.hits, 0)
	atomic.StoreUint32(&aerocacher.misses, 0)

	err := aerocacher.Init()
	if err != nil {
		return nil, err
	}
	return aerocacher, nil
}

func (c *Aerocacher) Init() error {
	if c.inited {
		return nil
	}

	policy := aero.NewClientPolicy()
	policy.User = c.cfg.Username
	policy.Password = c.cfg.Password
	if c.cfg.ConnectTimeoutMs != 0 {
		policy.Timeout = time.Duration(c.cfg.ConnectTimeoutMs) * time.Millisecond
	}
	if c.cfg.IdleTimeoutMs != 0 {
		policy.IdleTimeout = time.Duration(c.cfg.IdleTimeoutMs) * time.Millisecond
	}
	if c.cfg.LoginTimeoutMs != 0 {
		policy.LoginTimeout = time.Duration(c.cfg.LoginTimeoutMs) * time.Millisecond
	}
	if c.cfg.ConnectionQueueSize != 0 {
		policy.ConnectionQueueSize = c.cfg.ConnectionQueueSize
	}
	if c.cfg.OpeningConnectionThreshold != 0 {
		policy.OpeningConnectionThreshold = c.cfg.OpeningConnectionThreshold
	}
	if c.cfg.MinConnectionsPerNode != 0 {
		policy.MinConnectionsPerNode = c.cfg.MinConnectionsPerNode
	}

	aeroHosts := make([]*aero.Host, 0, len(c.cfg.Hosts))
	if len(c.cfg.Hosts) == 0 {
		return fmt.Errorf("NewAerocacher: hosts not defined")
	}
	for _, h := range c.cfg.Hosts {
		t := strings.Split(h, ":")
		if len(t) != 2 {
			return fmt.Errorf("NewAerocacher: bad host format, format - 'host:port'")
		}
		port, err := strconv.ParseInt(t[1], 10, 32)
		if err != nil {
			return fmt.Errorf("NewAerocacher: bad host format, format - 'host:port'")
		}
		aeroHosts = append(aeroHosts, aero.NewHost(t[0], int(port)))
	}
	client, err := aero.NewClientWithPolicyAndHost(policy, aeroHosts...)
	if err != nil {
		return fmt.Errorf("NewAerocacher: aerospike error %s", err)
	}
	c.client = client
	c.inited = true
	return nil
}

func (c *Aerocacher) Set(key string, payload []byte, ttlSeconds int) error {
	if !c.inited {
		return ErrNotInited
	}
	// log.Printf("Aerocache: set %s", key)

	aeroKey, err := aero.NewKey(c.cfg.Namespace, c.cfg.SetName, key)
	if err != nil {
		return err
	}

	aeroBins := aero.BinMap{}
	aeroBins[c.cfg.BinName] = payload

	wpolicy := aero.NewWritePolicy(0, uint32(ttlSeconds))
	start := time.Now()
	err = c.client.Put(wpolicy, aeroKey, aeroBins)
	c.requestCount++
	c.requestTimeSum += time.Since(start).Seconds()
	return err
}

func (c *Aerocacher) Get(key string) ([]byte, error) {
	data, _, err := c.GetWithTTL(key)
	return data, err
}

func (c *Aerocacher) GetWithTTL(key string) ([]byte, int, error) {
	if !c.inited {
		return nil, 0, ErrNotInited
	}
	// log.Printf("Aerocache: get %s", key)

	aeroKey, err := aero.NewKey(c.cfg.Namespace, c.cfg.SetName, key)
	if err != nil {
		return nil, 0, err
	}

	start := time.Now()
	rec, err := c.client.Get(nil, aeroKey)
	c.requestCount++
	c.requestTimeSum += time.Since(start).Seconds()
	if err != nil {
		atomic.AddUint32(&c.misses, 1)
		return nil, 0, ErrMiss
	}

	atomic.AddUint32(&c.hits, 1)
	return rec.Bins[c.cfg.BinName].([]byte), int(rec.Expiration), nil
}

func (c *Aerocacher) Del(key string) error {
	if !c.inited {
		return ErrNotInited
	}
	aeroKey, err := aero.NewKey(c.cfg.Namespace, c.cfg.SetName, key)
	if err != nil {
		return err
	}

	wpolicy := aero.NewWritePolicy(0, 0)
	start := time.Now()
	deleted, err := c.client.Delete(wpolicy, aeroKey)
	c.requestCount++
	c.requestTimeSum += time.Since(start).Seconds()
	if err != nil {
		return err
	}
	if !deleted {
		return ErrMiss
	}

	return nil
}

func (c *Aerocacher) BSet(key []byte, payload []byte, ttlSeconds int) error {
	if !c.inited {
		return ErrNotInited
	}
	// log.Printf("Aerocache: set %s", key)

	aeroKey, err := aero.NewKey(c.cfg.Namespace, c.cfg.SetName, key)
	if err != nil {
		return err
	}

	aeroBins := aero.BinMap{}
	aeroBins[c.cfg.BinName] = payload

	wpolicy := aero.NewWritePolicy(0, uint32(ttlSeconds))
	start := time.Now()
	err = c.client.Put(wpolicy, aeroKey, aeroBins)
	c.requestCount++
	c.requestTimeSum += time.Since(start).Seconds()
	return err
}

func (c *Aerocacher) BGet(key []byte) ([]byte, error) {
	data, _, err := c.BGetWithTTL(key)
	return data, err
}

func (c *Aerocacher) BGetWithTTL(key []byte) ([]byte, int, error) {
	if !c.inited {
		return nil, 0, ErrNotInited
	}
	// log.Printf("Aerocache: get %s", key)

	aeroKey, err := aero.NewKey(c.cfg.Namespace, c.cfg.SetName, key)
	if err != nil {
		return nil, 0, err
	}

	start := time.Now()
	rec, err := c.client.Get(nil, aeroKey)
	c.requestCount++
	c.requestTimeSum += time.Since(start).Seconds()
	if err != nil {
		atomic.AddUint32(&c.misses, 1)
		return nil, 0, ErrMiss
	}

	atomic.AddUint32(&c.hits, 1)
	return rec.Bins[c.cfg.BinName].([]byte), int(rec.Expiration), nil
}

func (c *Aerocacher) BDel(key []byte) error {
	if !c.inited {
		return ErrNotInited
	}
	aeroKey, err := aero.NewKey(c.cfg.Namespace, c.cfg.SetName, key)
	if err != nil {
		return err
	}

	wpolicy := aero.NewWritePolicy(0, 0)
	start := time.Now()
	deleted, err := c.client.Delete(wpolicy, aeroKey)
	c.requestCount++
	c.requestTimeSum += time.Since(start).Seconds()
	if err != nil {
		return err
	}
	if !deleted {
		return ErrMiss
	}

	return nil
}

func (c *Aerocacher) Close() {
	if !c.inited {
		return
	}
	c.client.Close()
	c.inited = false
}

func (c *Aerocacher) GetHits() uint32 {
	return atomic.LoadUint32(&c.hits)
}

func (c *Aerocacher) GetMisses() uint32 {
	return atomic.LoadUint32(&c.misses)
}

func (c *Aerocacher) GetAvgRequestTime() float64 {
	if c.requestCount == 0 {
		return 0.
	}
	return c.requestTimeSum / float64(c.requestCount)
}

// ------------------------------------------------------------------------------------------------
