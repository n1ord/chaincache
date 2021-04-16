package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/n1ord/chaincache"
)

var (
	AEROSPIKE_TEST_HOSTS = []string{
		"127.0.0.1:3000",
		"127.0.0.1:3000",
	}
	AEROSPIKE_TEST_NAMESPACE = "TEST_NS"
	AEROSPIKE_TEST_SETNAME   = "CACHE_TEST"

	REDIS_TEST_HOSTS = []string{
		"127.0.0.1:30001",
		"127.0.0.1:30002",
		"127.0.0.1:30003",
		"127.0.0.1:30004",
		"127.0.0.1:30005",
		"127.0.0.1:30006",
	}
)

func testCacher(t *testing.T, cacher chaincache.Cacher) {
	{
		//Base Set/Get functional
		ttl := 10
		N := 100
		for i := 0; i < N; i++ {
			key := fmt.Sprintf("%d", i)
			value := []byte(fmt.Sprintf("somevalue %d", i))
			err := cacher.Set(key, value, ttl)
			assert.Equal(t, err, nil)
			checkHit(t, cacher, key, value)
		}
		assert.Equal(t, cacher.GetHits(), uint32(N))

		for i := 0; i < N; i++ {
			key := fmt.Sprintf("new key %d", i)
			checkMiss(t, cacher, key)
		}
		assert.Equal(t, cacher.GetMisses(), uint32(N))
	}

	{
		//Test Del and ErrMiss
		key := "notexistskey"
		checkMiss(t, cacher, key)
		err := cacher.Del(key)
		assert.Equal(t, err, chaincache.ErrMiss)

		key = "existskey"
		value := []byte("newvalue")
		cacher.Set(key, value, 20)
		err = cacher.Del(key)
		assert.Equal(t, err, nil)

	}

	{
		//GetWithTTL test
		ttl := 3
		k := "somekey"
		v := []byte("somevalue")
		cacher.Set(k, v, ttl)
		for i := 0; i < ttl; i++ {
			got, gotTTL, err := cacher.GetWithTTL(k)
			assert.Equal(t, err, nil)
			assert.Equal(t, gotTTL, ttl-i)
			assert.Equal(t, got, v)
			time.Sleep(time.Second)
		}
		time.Sleep(time.Second)
		got, gotTTL, err := cacher.GetWithTTL(k)
		assert.Equal(t, err, chaincache.ErrMiss)
		assert.Equal(t, gotTTL, 0)
		assert.Equal(t, len(got), 0)
	}
}

func checkMiss(t *testing.T, c chaincache.Cacher, key string) {
	val, err := c.Get(key)
	assert.Equal(t, err, chaincache.ErrMiss)
	assert.Equal(t, len(val), 0)
}

func checkHit(t *testing.T, c chaincache.Cacher, key string, value []byte) {
	val, err := c.Get(key)
	assert.Equal(t, err, nil)
	assert.Equal(t, val, value)
}

//Freecacher tests
func TestFreecacher(t *testing.T) {
	//Base fucntionality
	fc, err := chaincache.NewFreeCacher(1024 * 10)
	if err != nil {
		panic(err)
	}
	testCacher(t, fc)
}

func TestAerocacher(t *testing.T) {
	cfg := &chaincache.AerocacherCfg{
		Hosts:     AEROSPIKE_TEST_HOSTS,
		Namespace: AEROSPIKE_TEST_NAMESPACE,
		SetName:   AEROSPIKE_TEST_SETNAME,
		BinName:   "data",
	}
	ac, err := chaincache.NewAerocacher(cfg)
	if err != nil {
		panic(err)
	}
	testCacher(t, ac)
	// fmt.Printf("Cacher avg request time: %fsec\n", ac.GetAvgRequestTime())
}

func TestRediscacher(t *testing.T) {
	cfg := chaincache.RediscacherCfg{
		Hosts: REDIS_TEST_HOSTS,
	}
	rc, err := chaincache.NewRediscacher(&cfg)
	if err != nil {
		panic(err)
	}
	testCacher(t, rc)
	// fmt.Printf("Cacher avg request time: %fsec\n", rc.GetAvgRequestTime())
}

func TestChainCache(t *testing.T) {
	{
		// checks chain set applying to all cachers
		fc1, _ := chaincache.NewFreeCacher(1024 * 10)
		fc2, _ := chaincache.NewFreeCacher(1024 * 10)
		fc3, _ := chaincache.NewFreeCacher(1024 * 10)
		chain, _ := chaincache.NewChainCache(fc1, fc2, fc3)
		key := "key"
		value := []byte("value")

		err := chain.Set(key, value, []int{60, 60, 60})
		assert.Equal(t, err, nil)

		checkHit(t, fc1, key, value)
		checkHit(t, fc2, key, value)
		checkHit(t, fc3, key, value)

		val, err := chain.Get(key)
		assert.Equal(t, err, nil)
		assert.Equal(t, val, value)

		// checks disabled backward cache
		key = "key2"
		value = []byte("value2")
		chain.NoBackwardCache = true

		err = fc3.Set(key, value, 60)
		assert.Equal(t, err, nil)

		checkMiss(t, fc1, key)
		checkMiss(t, fc2, key)
		checkHit(t, fc3, key, value)

		val, err = chain.Get(key)
		assert.Equal(t, err, nil)
		assert.Equal(t, val, value)

		checkMiss(t, fc1, key)
		checkMiss(t, fc2, key)

		// checks enabled backward cache
		chain.NoBackwardCache = false
		key = "key3"
		value = []byte("value3")

		err = fc3.Set(key, value, 60)
		assert.Equal(t, err, nil)

		checkMiss(t, fc1, key)
		checkMiss(t, fc2, key)
		checkHit(t, fc3, key, value)

		val, err = chain.Get(key)
		assert.Equal(t, err, nil)
		assert.Equal(t, val, value)

		checkHit(t, fc1, key, value)
		checkHit(t, fc2, key, value)
		checkHit(t, fc3, key, value)

		// checks backward ttl spreading
		chain.NoBackwardCache = false
		key = "key4"
		value = []byte("value4")

		fc3.Set(key, value, 5)
		time.Sleep(3 * time.Second)
		chain.Get(key)
		_, ttl, _ := fc2.GetWithTTL(key)
		assert.Equal(t, ttl, 2)
		_, ttl, _ = fc1.GetWithTTL(key)
		assert.Equal(t, ttl, 2)

		chain.Get("somekeynotexisted")

		assert.Equal(t, fc1.GetHits(), uint32(4))
		assert.Equal(t, fc1.GetMisses(), uint32(7))

		assert.Equal(t, fc2.GetHits(), uint32(3))
		assert.Equal(t, fc2.GetMisses(), uint32(7))

		assert.Equal(t, fc3.GetHits(), uint32(7))
		assert.Equal(t, fc3.GetMisses(), uint32(1))

		assert.Equal(t, chain.GetHits(), uint32(4))
		assert.Equal(t, chain.GetMisses(), uint32(1))
	}
}