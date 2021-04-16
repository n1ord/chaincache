# ChainCache

# 1. Штоэто?
Интерфейс над набором key/value стораджей, умеющий заодно объединять их в цепочки. Прямо как https://github.com/eko/gocache, но ламповей

Умеет:
- Хранить данные локально во Freecache
- Хранить данные удаленно в кластере Aerospike
- Хранить данные удаленно в кластере Redis
- Работать сразу с цепочкой стораджей
- Задавать отдельные TTL на каждую запись/сторадж
- Собирать стату: Hits, Misses, AvgRequestTime (для редиса и аэроспайка)

# 2. Подробнее

Либа реализует набор оберток над тремя хранилищами - Freecache, Aerospike, Redis. В каждой из них реализован базовый набор операций: Get, Set, Del и каждый может
использоваться сам по себе как кешер или key/value сторадж.

Методы подробнее:
```go
	// Возвращает данные по ключу, err == chaincache.ErrMiss если ключ не найден
	Get(key string) ([]byte, error)
	// Тоже, что и Get, но возвращает также оставшийся у записи TTL в секундах
	GetWithTTL(key string) ([]byte, int, error)
	// Пишет данные по ключу
	Set(key string, payload []byte, ttl int) error
	// Удаляет
	Del(key string) error
	// Число хитов стораджа
	GetHits() uint32
	// Число промахов
	GetMisses() uint32
```	

У стораджей Rediscacher и Aerocacher также есть метод, отдающий среднее время запроса в базу
```go
	// Отдает время в секундах
    GetAvgRequestTime() float64
```

До кучи добавлена возможность объединения стораджей в цепочки.
Цепочка имеет реализацию методов: Get, Set, Del и работает следующим образом:
- Get: цепочка ищет ключ по стораджам слева направо, пока, собственно, не найдет, после чего найденные данные будут записаны туда, где этот ключ не нашелся, причем в качестве TTL (сколько записи жить осталось) будет использовано оставшееся время из найденного стороджа. Это поведение можно отключить (выставив NoBackwardWrite=true), тогда результат будет возвращен немедленно
- Set: данные пишутся всюду, можно задать свой TTL для каждого стораджа отдельно
- Del: данные удаляются везде

# 3. Пример
```go
// Create 20mb local cache
localcacher, err := chaincache.NewFreecacher(1024*1024*20) 
if err != nil {
    panic(err)
}

// Create redis cacher
rediscacher, err := chaincache.NewRediscacher(&chaincache.RediscacherCfg{
    Hosts: []string{
        "127.0.0.1:30001",
		"127.0.0.1:30002",
    },
})
if err != nil {
    panic(err)
}

// Join them to chain
chain, _ := chaincache.NewChainCache(localcacher, rediscacher)
defer chain.Close()

// Disable backward writing for missed keys, if we need it
chain.NoBackwardCache = true

// save in chain: for 1 minute in localcacher and 2 minutes in redis
chain.Set("somekey", []byte("somedata"), []int{60, 120})

// load data from chain (get it from localcacher in real)
val, err := chain.Get("somekey")
if err == nil {
	//HIT
} else if err == chaincache.ErrMiss {
    //MISS
}
```

# 4. Конфигурация

## FreeCache
```go
MaxSizeInBytes := 1024*1024*20 //20mb
fc, err := chaincache.NewFreecacher(MaxSizeInBytes)
```

## Aerocacher
Конфиг размечен yaml-тегами, можно добавлять в общий конфиг приложки
```go
cfg := &chaincache.AerocacherCfg{
    Hosts:     []string{"host1:port", "host2:port"},
    Namespace: "MyNamespace",   // aka Database
    SetName:   "MySetname",     // aka Table
    BinName:   "data",          // aka Column to store data

	Username  "",
	Password  "",
    
    //Can be omitted
	ConnectTimeoutMs 0,           // zero equals to 30 000 (30 sec)
	IdleTimeoutMs    0,           // zero equals to 55 000 (55 sec)
	LoginTimeoutMs   0,           // zero equals to 10 000 (10 sec)
	ConnectionQueueSize        0, // zero equals t0 256
	OpeningConnectionThreshold 0, // default 0
	MinConnectionsPerNode      0, // default 0
}
ac, err := chaincache.NewAerocacher(cfg)
```

## Rediscacher
Конфиг размечен yaml-тегами
```go
cfg := &chaincache.RediscacherCfg{
	Hosts          []string{"host1:port", "host2:port"},
	Username       "",
	Password       "",

    //Can be omitted
	MaxRetries     0,    // zero equals to 3, -1 disable retries
	DialTimeoutMs  0,    // default 0
	ReadTimeoutMs  0,    // zero equals to 3000 (3 sec)
	WriteTimeoutMs 0,    // zero equals to ReadTimeoutMs
    // PoolSize applies per cluster node and not for the whole cluster.
	PoolSize       0,    // zero equals to 5 * runtime.NumCPU()
}
rc, err := chaincache.NewRediscacher(cfg)

```