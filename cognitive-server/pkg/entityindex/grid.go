package entityindex

import (
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/geo"
)

const ShardCount = 64

type Grid struct {
	shards [ShardCount]*shard
}

func New() *Grid {
	g := &Grid{}
	for i := 0; i < ShardCount; i++ {
		g.shards[i] = newShard()
	}
	return g
}

// Add регистрирует сущность.
func (g *Grid) Add(id ecs.EntityID, pos geo.Location) {
	key := makeBucketKey(pos)
	shardIdx := getShardIndex(key)
	g.shards[shardIdx].add(key, id)
}

// Remove удаляет сущность.
func (g *Grid) Remove(id ecs.EntityID, pos geo.Location) {
	key := makeBucketKey(pos)
	shardIdx := getShardIndex(key)
	g.shards[shardIdx].remove(key, id)
}

// Move атомарно (с точки зрения логики шарда) переносит сущность.
func (g *Grid) Move(id ecs.EntityID, oldPos, newPos geo.Location) {
	oldKey := makeBucketKey(oldPos)
	newKey := makeBucketKey(newPos)

	if oldKey == newKey {
		return // Остались в том же бакете
	}

	oldShardIdx := getShardIndex(oldKey)
	newShardIdx := getShardIndex(newKey)

	// Если шарды разные — нужно быть аккуратными.
	// Блокировать оба шарда одновременно (Lock Coupling) чревато дедлоками.
	// Самый простой и быстрый способ:
	// 1. Удалить из старого.
	// 2. Добавить в новый.
	// Риск: микроскопический момент времени сущности нет в индексе.
	// Для MoveSystem это допустимо, так как она эксклюзивно владеет сущностью в этот момент.

	g.shards[oldShardIdx].remove(oldKey, id)
	g.shards[newShardIdx].add(newKey, id)
}

// QueryBucket возвращает всех кандидатов в бакете (область 16x16),
// содержащем указанную точку.
// ВАЖНО: Это Broad Phase. Вызывающий код ОБЯЗАН проверить точные координаты.
func (g *Grid) QueryBucket(pos geo.Location) []ecs.EntityID {
	key := makeBucketKey(pos)
	shardIdx := getShardIndex(key)
	return g.shards[shardIdx].getCopy(key)
}

// Reset очищает индекс.
func (g *Grid) Reset() {
	for _, s := range g.shards {
		s.mu.Lock()
		for k := range s.buckets {
			delete(s.buckets, k)
		}
		s.mu.Unlock()
	}
}
