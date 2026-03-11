package bolt_db

import (
	"fmt"
)

// GenericShardDB provides horizontal partitioning across multiple database instances
// with generic type support for values.
type GenericShardDB[V any] struct {
	factory  *Factory
	shards   int
	shardDBs *InMemKV[int, GenericIDatabase[V]]
	useCache bool
}

// NewGenericShardDB creates a new generic sharded database.
// Note: This creates an empty shard map. You need to manually add shard databases
// using SetShard or create them externally.
func NewGenericShardDB[V any](factory *Factory, shards int, useCache bool) *GenericShardDB[V] {
	return &GenericShardDB[V]{
		factory:  factory,
		shards:   shards,
		shardDBs: NewInmemMap[int, GenericIDatabase[V]](),
		useCache: useCache,
	}
}

// SetShard manually sets a database instance for a specific shard.
func (s *GenericShardDB[V]) SetShard(shardIdx int, db GenericIDatabase[V]) {
	s.shardDBs.Set(shardIdx, db)
}

// getShard determines which shard a bucket belongs to using the first byte.
func (s *GenericShardDB[V]) getShard(bucket []byte) int {
	if len(bucket) == 0 {
		return 0
	}
	if len(bucket) == 1 {
		return int(bucket[0]) % s.shards
	}
	// Use first + last byte for better distribution
	hash := int(bucket[0]) + int(bucket[len(bucket)-1])
	return hash % s.shards
}

func (s *GenericShardDB[V]) getShardDB(shard int) (GenericIDatabase[V], error) {
	db := s.shardDBs.Get(shard)
	if db == nil {
		return nil, fmt.Errorf("shard %d not found", shard)
	}
	return db, nil
}

// Get retrieves a value from the appropriate shard based on the bucket hash.
func (s *GenericShardDB[V]) Get(bucket []byte, key []byte) (*V, error) {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return nil, err
	}
	value, err := db.Get(bucket, key)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *GenericShardDB[V]) Set(bucket []byte, key []byte, value *V) error {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return err
	}
	return db.Set(bucket, key, value)
}

func (s *GenericShardDB[V]) Delete(bucket []byte, key []byte) error {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return err
	}
	return db.Delete(bucket, key)
}

// Exist checks if a key exists in the appropriate shard.
func (s *GenericShardDB[V]) Exist(bucket, key []byte) (bool, error) {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return false, err
	}
	return db.Exist(bucket, key)
}

// ForEach iterates over all key-value pairs in the specified bucket.
func (s *GenericShardDB[V]) ForEach(bucket []byte, fn func(key []byte, value *V) error) error {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return err
	}
	return db.ForEach(bucket, fn)
}

// List returns all key-value pairs from the specified bucket.
func (s *GenericShardDB[V]) List(bucket []byte) (map[string]*V, error) {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return nil, err
	}
	return db.List(bucket)
}

// Buckets returns a list of all bucket names from all shards combined.
func (s *GenericShardDB[V]) Buckets() [][]byte {
	result := make([][]byte, 0)
	s.shardDBs.ForEach(func(shardIdx int, db GenericIDatabase[V]) error {
		buckets := db.Buckets()
		result = append(result, buckets...)
		return nil
	})
	return result
}

// DropBucket deletes a bucket from the shard it belongs to.
func (s *GenericShardDB[V]) DropBucket(bucketName string) error {
	bucket := []byte(bucketName)
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return err
	}
	return db.DropBucket(bucketName)
}

// Close flushes and closes all shard databases.
func (s *GenericShardDB[V]) Close() error {
	return s.shardDBs.ForEach(func(shardIdx int, db GenericIDatabase[V]) error {
		return db.Close()
	})
}

// GetShardForBucket returns the database instance for the shard containing the given bucket.
func (s *GenericShardDB[V]) GetShardForBucket(bucket []byte) (GenericIDatabase[V], error) {
	shard := s.getShard(bucket)
	return s.getShardDB(shard)
}

// Lock acquires locks on all shard databases.
func (s *GenericShardDB[V]) Lock() {
	s.shardDBs.ForEach(func(shardIdx int, db GenericIDatabase[V]) error {
		db.Lock()
		return nil
	})
}

// Unlock releases locks on all shard databases.
func (s *GenericShardDB[V]) Unlock() {
	s.shardDBs.ForEach(func(shardIdx int, db GenericIDatabase[V]) error {
		db.Unlock()
		return nil
	})
}

// Name returns a descriptive name for this sharded database.
func (s *GenericShardDB[V]) Name() string {
	return fmt.Sprintf("GenericShardDB[%d shards]", s.shards)
}
