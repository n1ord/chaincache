package main

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/n1ord/chaincache"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var (
	maxMem                = 1024 * 1024 * 5
	limit                 = 1024 * 20
	dataMaxLen            = 2024
	seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))
)

func RandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// SET ------------------

func BenchmarkFastCacherSet(b *testing.B) {
	cacher, _ := chaincache.NewFastCacher(maxMem, true, false)

	keys, data := getKeysData(limit)

	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}
}

func BenchmarkFastCacherBigSet(b *testing.B) {
	cacher, _ := chaincache.NewFastCacher(maxMem, true, true)

	keys, data := getKeysData(limit)

	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}
}

func BenchmarkFastCacherBSet(b *testing.B) {
	cacher, _ := chaincache.NewFastCacher(maxMem, true, false)

	keys, data := getBKeysData(limit)

	for i := 0; i < b.N; i++ {
		cacher.BSet(keys[seededRand.Intn(limit)], data, 3600)
	}
}

func BenchmarkFastCacherNoTTLSet(b *testing.B) {
	cacher, _ := chaincache.NewFastCacher(maxMem, false, false)

	keys, data := getKeysData(limit)

	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}
}

func BenchmarkFastCacherNoTTLBSet(b *testing.B) {
	cacher, _ := chaincache.NewFastCacher(maxMem, false, false)

	keys, data := getBKeysData(limit)

	for i := 0; i < b.N; i++ {
		cacher.BSet(keys[seededRand.Intn(limit)], data, 3600)
	}
}

func BenchmarkFreeCacherSet(b *testing.B) {
	cacher, _ := chaincache.NewFreeCacher(maxMem)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}
}

func BenchmarkFreeCacherBSet(b *testing.B) {
	cacher, _ := chaincache.NewFreeCacher(maxMem)
	keys, data := getBKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.BSet(keys[seededRand.Intn(limit)], data, 3600)
	}
}

func BenchmarkProbeCacherSet(b *testing.B) {
	cacher, _ := chaincache.NewProbecacher(50, maxMem, 1024*1024*48, 7, chaincache.STORAGE_LRU)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}
}

func BenchmarkProbeCacherBSet(b *testing.B) {
	cacher, _ := chaincache.NewProbecacher(50, maxMem, 1024*1024*48, 7, chaincache.STORAGE_LRU)
	keys, data := getBKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.BSet(keys[seededRand.Intn(limit)], data, 3600)
	}
}

// GET -------------------------

func BenchmarkFastCacherGet(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFastCacher(maxMem, true, false)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cacher.Get(keys[seededRand.Intn(limit)])
	}
}

func BenchmarkFastCacherBigGet(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFastCacher(maxMem, true, true)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cacher.Get(keys[seededRand.Intn(limit)])
	}
}

func BenchmarkFastCacherBGet(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFastCacher(maxMem, true, false)
	keys, data := getBKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.BSet(keys[seededRand.Intn(limit)], data, 3600)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cacher.BGet(keys[seededRand.Intn(limit)])
	}
}

func BenchmarkFastCacherNoTTLGet(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFastCacher(maxMem, false, false)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cacher.Get(keys[seededRand.Intn(limit)])
	}
}

func BenchmarkFastCacherNoTTLBGet(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFastCacher(maxMem, false, false)
	keys, data := getBKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.BSet(keys[seededRand.Intn(limit)], data, 3600)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cacher.BGet(keys[seededRand.Intn(limit)])
	}
}

func BenchmarkFreeCacherGet(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFreeCacher(maxMem)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cacher.Get(keys[seededRand.Intn(limit)])
	}
}

func BenchmarkFreeCacherBGet(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFreeCacher(maxMem)
	keys, data := getBKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.BSet(keys[seededRand.Intn(limit)], data, 3600)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cacher.BGet(keys[seededRand.Intn(limit)])
	}
}

func BenchmarkProbeCacherGet(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewProbecacher(50, maxMem, 1024*1024*48, 7, chaincache.STORAGE_LFU)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cacher.Get(keys[seededRand.Intn(limit)])
	}
}

func BenchmarkProbeCacherBGet(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewProbecacher(50, maxMem, 1024*1024*48, 7, chaincache.STORAGE_LFU)
	keys, data := getBKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.BSet(keys[seededRand.Intn(limit)], data, 3600)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cacher.BGet(keys[seededRand.Intn(limit)])
	}
}

// SET Parralel -------------------------
func BenchmarkFastCacherSetParallel(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFastCacher(maxMem, true, false)
	keys, data := getKeysData(limit)

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		ix := 0
		for pb.Next() {
			cacher.Set(keys[ix], data, 3600)
			ix++
			if ix >= limit {
				ix = 0
			}
		}
	})
}
func BenchmarkFastCacherNoTTLSetParallel(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFastCacher(maxMem, false, false)
	keys, data := getKeysData(limit)

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		ix := 0
		for pb.Next() {
			cacher.Set(keys[ix], data, 3600)
			ix++
			if ix >= limit {
				ix = 0
			}
		}
	})
}

func BenchmarkFreeCacherSetParallel(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFreeCacher(maxMem)
	keys, data := getKeysData(limit)

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		ix := 0
		for pb.Next() {
			cacher.Set(keys[ix], data, 3600)
			ix++
			if ix >= limit {
				ix = 0
			}
		}
	})
}
func BenchmarkProbeCacheSetParallel(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewProbecacher(50, maxMem, int(float64(maxMem)*1.2), 7, chaincache.STORAGE_LFU)
	keys, data := getKeysData(limit)

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		ix := 0
		for pb.Next() {
			cacher.Set(keys[ix], data, 3600)
			ix++
			if ix >= limit {
				ix = 0
			}
		}
	})
}

// GET Parralel -------------------------
func BenchmarkFastCacherGetParallel(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFastCacher(maxMem, true, false)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		ix := 0
		for pb.Next() {
			cacher.Get(keys[ix])
			ix++
			if ix >= limit {
				ix = 0
			}
		}
	})
}

func BenchmarkFastCacherNoTTLGetParallel(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFastCacher(maxMem, false, false)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 1000)
	}

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		ix := 0
		for pb.Next() {
			cacher.Get(keys[ix])
			ix++
			if ix >= limit {
				ix = 0
			}
		}
	})
}

func BenchmarkFreeCacherGetParallel(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewFreeCacher(maxMem)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		ix := 0
		for pb.Next() {
			cacher.Get(keys[ix])
			ix++
			if ix >= limit {
				ix = 0
			}
		}
	})
}

func BenchmarkProbeCacherGetParallel(b *testing.B) {
	b.StopTimer()
	cacher, _ := chaincache.NewProbecacher(50, maxMem, int(float64(maxMem)*1.2), 7, chaincache.STORAGE_LFU)
	keys, data := getKeysData(limit)
	for i := 0; i < b.N; i++ {
		cacher.Set(keys[seededRand.Intn(limit)], data, 3600)
	}

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		ix := 0
		for pb.Next() {
			cacher.Get(keys[ix])
			ix++
			if ix >= limit {
				ix = 0
			}
		}
	})
}

func getKeysData(limit int) ([]string, []byte) {
	keys := make([]string, limit)
	for i := 0; i < limit; i++ {
		keyLen := seededRand.Intn(32) + 32
		keys[i] = RandomString(keyLen)
	}

	dataLen := seededRand.Intn(dataMaxLen-20) + 20
	return keys, []byte(RandomString(dataLen))
}

func getBKeysData(limit int) ([][]byte, []byte) {
	keys := make([][]byte, limit)
	for i := 0; i < limit; i++ {
		keyLen := seededRand.Intn(32) + 32
		keys[i] = []byte(RandomString(keyLen))
	}

	dataLen := seededRand.Intn(dataMaxLen-20) + 20
	return keys, []byte(RandomString(dataLen))
}

func key(i int) string {
	return fmt.Sprintf("key-%010d", i)
}

func value() []byte {
	return make([]byte, 100)
}
