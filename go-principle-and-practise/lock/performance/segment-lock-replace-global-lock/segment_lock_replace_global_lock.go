package segment_lock_replace_global_lock

import "sync"

const SHARD_COUNT = 32

type Map interface {
	Set(key, value string)
	Get(key string) (value string)
}

type LockedMap struct {
	m       sync.RWMutex
	buckets map[string]string
}

func NewLockedMap() Map {
	return &LockedMap{
		buckets: make(map[string]string, 1000),
	}
}

func (lm *LockedMap) Set(key, value string) {
	lm.m.Lock()
	defer lm.m.Unlock()
	lm.buckets[key] = value
}

func (lm *LockedMap) Get(key string) (value string) {
	lm.m.RLock()
	defer lm.m.RUnlock()
	value = lm.buckets[key]
	return
}

type SegmentMap []*SubMap

// SubMap 每一个Map 是一个加锁的并发安全Map
type SubMap struct {
	items        map[string]string
	sync.RWMutex // 各个分片Map各自的锁
}

func NewSegmentMap() SegmentMap {
	// SHARD_COUNT  默认32个分片
	m := make(SegmentMap, SHARD_COUNT)
	for i := 0; i < SHARD_COUNT; i++ {
		m[i] = &SubMap{
			items: make(map[string]string, 30),
		}
	}
	return m
}

func (m SegmentMap) GetShard(key string) *SubMap {
	return m[uint(fnv32(key))%uint(SHARD_COUNT)]
}

// FNV hash
func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

func (m SegmentMap) Set(key string, value string) {
	shard := m.GetShard(key) // 段定位找到分片
	shard.Lock()             // 分片上锁
	shard.items[key] = value // 分片操作
	shard.Unlock()           // 分片解锁
}

func (m SegmentMap) Get(key string) (value string) {
	shard := m.GetShard(key)
	shard.RLock()
	value = shard.items[key]
	shard.RUnlock()
	return
}

// Count 统计当前分段map中item(键值对)的个数
func (m SegmentMap) Count() int {
	count := 0
	for i := 0; i < SHARD_COUNT; i++ {
		shard := m[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Keys 获取所有的key
func (m SegmentMap) Keys() []string {
	count := m.Count()
	ch := make(chan string, count)

	// 每一个分片启动一个协程 遍历key
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(SHARD_COUNT)
		for _, shard := range m {
			go func(shard *SubMap) {
				defer wg.Done()
				shard.RLock()
				// 每个分片中的key遍历后都写入统计用的channel
				for key := range shard.items {
					ch <- key
				}
				shard.RUnlock()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()

	keys := make([]string, count)
	// 统计各个协程并发读取Map分片的key
	for k := range ch {
		keys = append(keys, k)
	}
	return keys
}
