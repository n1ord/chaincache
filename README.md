# ChainCache

# 1. Штоэто?
Интерфейс над набором key/value стораджей/библиотек, умеющий заодно объединять их в цепочки.

Умеет:
- Локально во [Freecache](github.com/coocood/freecache)
- Локально в [FastCache](github.com/VictoriaMetrics/fastcache) (+ttl over fastcache)
- Локально в [Probecache](https://github.com/n1ord/probecache) (простой кеш на мапах)
- Удаленно в кластере [Aerospike](github.com/aerospike/aerospike-client-go)
- Удаленно в [Redis](github.com/go-redis/redis/v8)
- Работать сразу с цепочкой стораджей
- Задавать отдельные TTL на каждую запись/сторадж
- Собирать стату: Hits, Misses, AvgRequestTime (для редиса и аэроспайка)
- Все thread-safe
- Все умеют в expiration записей

# 2. Подробнее

Либа реализует набор оберток над несколькими либами/хранилищами - Fastcache, Freecache, Aerospike, Redis, Probecache. В каждой из них реализован базовый набор операций: Get, Set, Del и каждый может
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
- Get: цепочка ищет ключ по стораджам слева направо, пока, собственно, не найдет. Найденное вернется, плюс найденные данные будут записаны туда, где этот ключ не нашелся, причем в качестве TTL (сколько записи жить осталось) будет использовано оставшееся время из найденного стороджа. Это поведение можно отключить (выставив NoBackwardWrite=true), тогда результат будет возвращен немедленно
- Set: данные пишутся всюду, можно задать свой TTL для каждого стораджа отдельно
- Del: данные удаляются везде

## Опции
```go
// Если true, отключает логику, когда кеш, найденный в дальнем элементе цепочки будет автоматически записан во все предшествующие элементы с остатком его ttl, по-умолчанию false
<chaincache instance>.NoBackwardCache = true

// Если true, цепочка игнорирует все внутренние ошибки кешеров (за исключением паник), интерпретируя их как ErrMiss. Может быть полезно при разработке или в ситуациях, когда один из кешеров цепочки не критичен и может отвалиться. По-умолчанию false
<chaincache instance>.IgnoreErrors = true
```

# 3. Пример
```go
// Create 40mb local cache
maxSize := 1024*1024*40
localcacher, err := chaincache.NewFastCacher(maxSize) 
if err != nil {
    panic(err)
}

// Create redis cacher
rediscacher, err := chaincache.NewRediscacher(&chaincache.RediscacherCfg{
    Host: "127.0.0.1:30001",
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


## FastCache
Интерфейс над fastcache реализован с вариантом работы через ttl (искаропки его там нет). Без ttl данный сторадж игнорирует все передаваемые значения ttl_expiration - значения живут вечно вплоть до вытеснения по памяти. Его включение радикально замедляет операции записи, но гарантирует совместимость с прочими стораджами, где используется ttl
```go
MaxSizeInBytes := 1024*1024*32 //32mb - minimum
useTTL := true
fc, err := chaincache.NewFastcacher(MaxSizeInBytes, useTTL)
```

## FreeCache
```go
MaxSizeInBytes := 1024*1024*20 //20mb
fc, err := chaincache.NewFreecacher(MaxSizeInBytes)
```

## Probecache
Это локальный кеш на мапах с хитрым вытеснением. В силу мап медленнен на запись, побаивается GC, но космически быстр на чтениях.
```go
// create local LRU cache storing 20-25mb
maxMemSize := 1024*1024*20
critMemSize := 1024*1024*25
shards  := 10
maxCleanDepth := 6
localcacher, err := chaincache.NewProbecacher(shards, maxMemSize, critMemSize, maxCleanDepth, chaincache.STORAGE_LRU) 
if err != nil {
    panic(err)
}
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
