package bolt_db

import (
	"errors"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
)

// ShardDB provides horizontal partitioning across multiple database instances.
// It distributes buckets across multiple database files based on the bucket name's
// first and last bytes, improving write throughput and reducing contention for
// high-concurrency workloads.
//
// The sharding strategy:
//   - Uses bucket's first byte + last byte to determine shard
//   - Each shard is an independent database file
//   - All keys within a bucket are stored in the same shard
//   - Aggregation operations (Buckets) combine results from all shards
type ShardDB struct {
	factory  *Factory                 // Factory managing the shard databases
	shards   int                      // Total number of shards
	shardDBs *InMemKV[int, IDatabase] // Map of shard index to database instance
	useCache bool                     // Whether to use cache for the shard databases
}

// NewShardDB creates a new sharded database with the specified number of shards.
// Each shard is created as a separate database file named "shard{N}.db".
//
// Parameters:
//   - factory: The factory to manage shard databases
//   - shards: The number of shards to create (must be > 	0)
//
// Returns:
//   - *ShardDB: A new sharded database instance, or nil if creation fails
//   - error: Any error that occurred during shard creation
func NewShardDB(factory *Factory, dbPath string, shards int, useCache bool) (*ShardDB, error) {
	if shards <= 0 {
		return nil, errors.New("shards must be greater than 0")
	}

	shardDBs := NewInmemMap[int, IDatabase]()

	for i := range shards {
		shardDB, err := factory.Open(fmt.Sprintf("%s_shard%d", dbPath, i), fmt.Sprintf("%s_shard%d.db", dbPath, i))
		if err != nil {
			return nil, fmt.Errorf("failed to open shard %d: %w", i, err)
		}

		if useCache {
			cachedDB := NewCachedDB(shardDB, 300, 5*time.Minute, 15*time.Minute)
			shardDBs.Set(i, cachedDB)
		} else {
			shardDBs.Set(i, shardDB)
		}
	}

	return &ShardDB{
		factory:  factory,
		shards:   shards,
		shardDBs: shardDBs,
	}, nil
}

// getShard determines which shard a bucket belongs to using first + last byte.
// This ensures all keys in the same bucket are stored in the same shard.
//
// Parameters:
//   - bucket: The bucket name
//
// Returns:
//   - int: The shard index for this bucket
func (s *ShardDB) getShard(bucket []byte) int {
	if len(bucket) == 0 {
		return 0
	}
	if len(bucket) == 1 {
		return int(bucket[0]) % s.shards
	}
	hash := int(bucket[0]) + int(bucket[len(bucket)-1])
	return hash % s.shards
}

// getShardDB retrieves the database instance for a specific shard index.
//
// Parameters:
//   - shard: The shard index
//
// Returns:
//   - *DB: The database instance for this shard
//   - error: Error if shard doesn't exist
func (s *ShardDB) getShardDB(shard int) (IDatabase, error) {
	db := s.shardDBs.Get(shard)
	if db == nil {
		return nil, fmt.Errorf("shard %d not found", shard)
	}
	return db, nil
}

// Get retrieves a value from the appropriate shard based on the bucket hash.
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to retrieve
//
// Returns:
//   - []byte: The value associated with the key, or nil if not found
//   - error: Any error that occurred during the operation
func (s *ShardDB) Get(bucket, key []byte) ([]byte, error) {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return nil, err
	}
	return db.Get(bucket, key)
}

// Set stores a key-value pair in the appropriate shard based on the bucket hash.
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to store
//   - value: The value to store
//
// Returns:
//   - error: Any error that occurred during the operation
func (s *ShardDB) Set(bucket, key, value []byte) error {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return err
	}
	return db.Set(bucket, key, value)
}

// Delete removes a key-value pair from the appropriate shard based on the bucket hash.
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to delete
//
// Returns:
//   - error: Any error that occurred during the operation
func (s *ShardDB) Delete(bucket, key []byte) error {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return err
	}
	return db.Delete(bucket, key)
}

// ForEach iterates over all key-value pairs in the specified bucket.
// Since buckets are sharded, this will only iterate over the shard containing this bucket.
//
// Parameters:
//   - bucket: The bucket name to iterate over
//   - fn: A function called for each key-value pair
//
// Returns:
//   - error: Any error that occurred during iteration
func (s *ShardDB) ForEach(bucket []byte, fn func(key, value []byte) error) error {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return err
	}
	return db.ForEach(bucket, fn)
}

// List returns all key-value pairs from the specified bucket.
// Since buckets are sharded, this will only query the shard containing this bucket.
//
// Parameters:
//   - bucket: The bucket name to list
//
// Returns:
//   - map[string][]byte: A map of all key-value pairs in the bucket
//   - error: Any error that occurred during the operation
func (s *ShardDB) List(bucket []byte) (map[string][]byte, error) {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return nil, err
	}
	return db.List(bucket)
}

// Buckets returns a list of all bucket names from all shards combined.
//
// Returns:
//   - [][]byte: A list of all bucket names across all shards
func (s *ShardDB) Buckets() [][]byte {
	result := make([][]byte, 0)
	s.shardDBs.ForEach(func(shardIdx int, db IDatabase) error {
		buckets := db.Buckets()
		result = append(result, buckets...)
		return nil
	})
	return result
}

// DropBucket deletes a bucket from the shard it belongs to.
//
// Parameters:
//   - bucketName: The name of the bucket to delete
//
// Returns:
//   - error: Any error that occurred during the operation
func (s *ShardDB) DropBucket(bucketName string) error {
	bucket := []byte(bucketName)
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return err
	}
	return db.DropBucket(bucketName)
}

// Write executes a write transaction on all shards.
// WARNING: This operation is executed on ALL shards and may be expensive.
//
// Parameters:
//   - fn: The transaction function to execute on each shard
//
// Returns:
//   - error: Any error that occurred during the transaction
func (s *ShardDB) Write(fn func(tx *bolt.Tx) error) error {
	return s.shardDBs.ForEach(func(shardIdx int, db IDatabase) error {
		return db.Write(fn)
	})
}

// Read executes a read-only transaction on all shards.
// WARNING: This operation is executed on ALL shards.
//
// Parameters:
//   - fn: The read transaction function to execute
//
// Returns:
//   - error: Any error that occurred during the transaction
func (s *ShardDB) Read(fn func(tx *bolt.Tx) error) error {
	return s.shardDBs.ForEach(func(shardIdx int, db IDatabase) error {
		return db.Read(fn)
	})
}

// Close flushes and closes all shard databases.
//
// Returns:
//   - error: Any error that occurred during closing
func (s *ShardDB) Close() error {
	return s.shardDBs.ForEach(func(shardIdx int, db IDatabase) error {
		name := fmt.Sprintf("shard%d", shardIdx)
		return s.factory.Close(name)
	})
}

// executeBatch executes batch operations by routing each operation to its shard.
// Operations are grouped by shard based on bucket hash for efficient execution.
//
// Parameters:
//   - ops: Map of bucket names to their operations
//
// Returns:
//   - error: Any error that occurred during execution
func (s *ShardDB) executeBatch(ops map[string][]*WriteOperation) error {
	shardOps := make(map[int]map[string][]*WriteOperation)

	for bucketName, bucketOps := range ops {
		shard := s.getShard([]byte(bucketName))
		if shardOps[shard] == nil {
			shardOps[shard] = make(map[string][]*WriteOperation)
		}
		shardOps[shard][bucketName] = bucketOps
	}

	for shardIdx, shardOpMap := range shardOps {
		db, err := s.getShardDB(shardIdx)
		if err != nil {
			return err
		}
		if err := db.executeBatch(shardOpMap); err != nil {
			return fmt.Errorf("shard %d batch failed: %w", shardIdx, err)
		}
	}

	return nil
}

func (s *ShardDB) GetShardForBucket(bucket []byte) (IDatabase, error) {
	shard := s.getShard(bucket)
	return s.getShardDB(shard)
}

func (s *ShardDB) Exist(bucket, key []byte) (bool, error) {
	shard := s.getShard(bucket)
	db, err := s.getShardDB(shard)
	if err != nil {
		return false, err
	}
	exists, err := db.Exist(bucket, key)
	if err != nil {
		return false, err
	}
	return exists, nil
}
