package entityindex

import (
	"cognitive-server/pkg/ecs"
	"sync"
)

type shard struct {
	mu sync.RWMutex
	// buckets: Ключ бакета -> Слайс ID
	buckets map[BucketKey][]ecs.EntityID
}

func newShard() *shard {
	return &shard{
		buckets: make(map[BucketKey][]ecs.EntityID),
	}
}

func (s *shard) add(key BucketKey, id ecs.EntityID) {
	s.mu.Lock()
	s.buckets[key] = append(s.buckets[key], id)
	s.mu.Unlock()
}

func (s *shard) remove(key BucketKey, id ecs.EntityID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bucket := s.buckets[key]
	// Fast remove (Swap & Pop)
	for i, entID := range bucket {
		if entID == id {
			lastIdx := len(bucket) - 1
			bucket[i] = bucket[lastIdx]
			s.buckets[key] = bucket[:lastIdx]

			if len(s.buckets[key]) == 0 {
				delete(s.buckets, key)
			}
			return
		}
	}
}

// getCopy возвращает копию слайса, чтобы читатель мог работать без лока шарда.
func (s *shard) getCopy(key BucketKey) []ecs.EntityID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	src := s.buckets[key]
	if len(src) == 0 {
		return nil
	}

	dst := make([]ecs.EntityID, len(src))
	copy(dst, src)
	return dst
}
