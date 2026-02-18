package segmentlockreplacegloballock

import "sync"

const ShardCount = 32

type Map interface {
	Set(key, value string)
	Get(key string) (value string)
}

type LockedMap struct {
	buckets map[string]string
	m       sync.RWMutex
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
	// ShardCount  默认32个分片
	m := make(SegmentMap, ShardCount)
	for i := 0; i < ShardCount; i++ {
		m[i] = &SubMap{
			items: make(map[string]string, 30),
		}
	}
	return m
}

func (m SegmentMap) GetShard(key string) *SubMap {
	return m[uint(fnv32(key))%uint(ShardCount)]
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
