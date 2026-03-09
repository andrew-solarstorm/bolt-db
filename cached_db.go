package bolt_db

import (
	"errors"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

// cached represents a single cached entry with metadata for cache management.
// It tracks the data, number of updates, and timing information for cache eviction
// and persistence decisions.
type cached struct {
	data     []byte    // The actual cached data
	updates  int       // Number of updates since last persistence
	updateAt time.Time // Time of the last update
	storedAt time.Time // Time when data was last persisted to disk
}

// CachedDB provides an in-memory caching layer on top of a Bolt database.
// It implements intelligent write-through caching with configurable persistence
// strategies based on update frequency, time intervals, and velocity.
//
// The caching strategy uses three key parameters:
//   - updateThreshold: Number of updates before considering persistence
//   - updateInterval: Maximum time between disk writes
//   - deleteInterval: Time of inactivity before cache eviction
//
// Data is persisted when:
//  1. Updates exceed threshold AND velocity is low (≤ 1 update/second)
//  2. Time since last persist exceeds updateInterval
//  3. Manual Flush() is called
type CachedDB struct {
	db              *DB                                         // Underlying database instance
	store           *InMemKV[string, *InMemKV[string, *cached]] // Two-level cache: bucket -> key -> cached data
	updateThreshold int                                         // Number of updates before persistence check
	updateInterval  time.Duration                               // Maximum time between disk writes
	deleteInterval  time.Duration                               // Time of inactivity before cache eviction
	lck             sync.RWMutex                                // Lock for coordinating multi-DB transactions
}

// copy creates a deep copy of byte slice data to prevent external modifications
// from affecting cached data.
//
// Parameters:
//   - data: The byte slice to copy
//
// Returns:
//   - []byte: A new byte slice with copied data, or nil if input is nil
func (cd *CachedDB) copy(data []byte) []byte {
	if data == nil {
		return nil
	}
	copied := make([]byte, len(data))
	copy(copied, data)
	return copied
}

// NewCachedDB creates a new cached database instance with the specified caching parameters.
// If any parameter is <= 0, default values are used:
//   - updateThreshold: 300 updates
//   - updateInterval: 5 minutes
//   - deleteInterval: 15 minutes
//
// The cache uses a two-level map structure (bucket -> key -> cached data) with
// thread-safe operations via read-write locks.
//
// Parameters:
//   - db: The underlying DB instance to cache
//   - updateThreshold: Number of updates before considering persistence (0 = use default)
//   - updateInterval: Maximum time between disk writes (0 = use default)
//   - deleteInterval: Time of inactivity before cache eviction (0 = use default)
//
// Returns:
//   - *CachedDB: A new cached database instance
func NewCachedDB(
	db *DB,
	updateThreshold int,
	updateInterval, deleteInterval time.Duration,
) *CachedDB {
	if updateThreshold <= 0 {
		updateThreshold = 300
	}
	if updateInterval <= 0 {
		updateInterval = 5 * time.Minute
	}
	if deleteInterval <= 0 {
		deleteInterval = 15 * time.Minute
	}
	return &CachedDB{
		db:              db,
		store:           NewInmemMap[string, *InMemKV[string, *cached]](),
		updateThreshold: updateThreshold,
		updateInterval:  updateInterval,
		deleteInterval:  deleteInterval,
	}
}

// Name returns the underlying database path prefixed with "cached:".
//
// Returns:
//   - string: The cached database identifier
func (cd *CachedDB) Name() string {
	return "cached:" + cd.db.Name()
}

// Get retrieves a value from the specified bucket and key.
// The method first checks the cache. If not found in cache, it retrieves
// from the underlying database and stores in cache for future reads.
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to retrieve
//
// Returns:
//   - []byte: A copy of the cached or retrieved data, or nil if not found
//   - error: Any error that occurred during the operation
func (cd *CachedDB) Get(bucket, key []byte) ([]byte, error) {
	b := cd.store.Get(string(bucket))
	if b != nil {
		c := b.Get(string(key))
		if c != nil {
			return cd.copy(c.data), nil
		}
	}

	data, err := cd.db.Get(bucket, key)
	if err != nil {
		return nil, err
	}

	b = cd.store.Get(string(bucket))
	if b == nil {
		b = NewInmemMap[string, *cached]()
		cd.store.Set(string(bucket), b)
	}

	now := time.Now()
	b.Set(string(key), &cached{
		data:     cd.copy(data),
		updates:  0,
		storedAt: now,
		updateAt: now,
	})

	return cd.copy(data), nil
}

// Set stores a key-value pair in the cache and conditionally persists to disk.
// The data is always written to cache immediately. Persistence to disk happens
// based on the caching strategy (update threshold, interval, and velocity).
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to store
//   - data: The value to store
//
// Returns:
//   - error: Any error that occurred during the operation
func (cd *CachedDB) Set(bucket, key []byte, data []byte) error {
	b := cd.store.Get(string(bucket))
	if b == nil {
		b = NewInmemMap[string, *cached]()
		cd.store.Set(string(bucket), b)
	}

	item := b.Get(string(key))
	now := time.Now()
	if item == nil {
		item = &cached{
			data:     cd.copy(data),
			updates:  0,
			storedAt: now,
			updateAt: now,
		}
		b.Set(string(key), item)
	} else {
		item.data = cd.copy(data)
		item.updates++
		item.updateAt = now
	}

	if cd.canSave(item) {
		err := cd.db.Set(bucket, key, data)
		if err != nil {
			return err
		}
		item.updates = 0
		item.storedAt = now
	}

	return nil
}

// Exist checks if a key exists in the specified bucket.
// It first checks the cache, then falls back to the underlying database if not cached.
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to check
//
// Returns:
//   - bool: true if the key exists, false otherwise
//   - error: Any error that occurred during the operation
func (cd *CachedDB) Exist(bucket, key []byte) (bool, error) {
	b := cd.store.Get(string(bucket))
	if b != nil {
		exists := b.Exists(string(key))
		if exists {
			return true, nil
		}
	}
	data, err := cd.db.Get(bucket, key)
	if err != nil {
		return false, err
	}
	return data != nil, nil
}

// Delete removes a key-value pair from both the cache and the underlying database.
// The deletion is applied to cache immediately and to disk synchronously.
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to delete
//
// Returns:
//   - error: Any error that occurred during the operation
func (cd *CachedDB) Delete(bucket, key []byte) error {
	b := cd.store.Get(string(bucket))
	if b != nil {
		b.Delete(string(key))
	}

	return cd.db.Delete(bucket, key)
}

// canSave determines if a cached item should be persisted to disk.
// Persistence occurs when:
//  1. Updates >= threshold AND velocity <= 1.0 update/second
//  2. Time since last persist >= updateInterval AND item has updates
//
// Parameters:
//   - item: The cached item to evaluate
//
// Returns:
//   - bool: true if the item should be persisted
func (cd *CachedDB) canSave(item *cached) bool {
	elapsed := time.Since(item.storedAt)
	if elapsed == 0 {
		return false
	}

	velocity := float64(item.updates) / elapsed.Seconds()

	if item.updates >= cd.updateThreshold && velocity <= 1.0 {
		return true
	}

	if elapsed >= cd.updateInterval && item.updates > 0 {
		return true
	}

	return false
}

// canDelete determines if a cached item should be evicted from cache.
// Eviction occurs when the item has been inactive for longer than deleteInterval.
//
// Parameters:
//   - item: The cached item to evaluate
//
// Returns:
//   - bool: true if the item should be evicted from cache
func (cd *CachedDB) canDelete(item *cached) bool {
	return time.Since(item.updateAt) >= cd.deleteInterval
}

// Flush persists all cached data to disk and evicts inactive entries.
// For each bucket and key in the cache:
//   - If inactive for > deleteInterval, evict from cache
//   - If has pending updates, persist to disk and reset update counter
//
// This method should be called:
//   - Before application shutdown
//   - Periodically for eventual consistency
//   - Before critical operations requiring data persistence
//
// Returns:
//   - error: Any error that occurred during flushing
func (cd *CachedDB) Flush() error {
	err := cd.store.ForEach(func(bucketName string, bucket *InMemKV[string, *cached]) error {
		return bucket.ForEach(func(key string, item *cached) error {
			if cd.canDelete(item) {
				bucket.Delete(key)
				return nil
			}
			if item.updates > 0 {
				if err := cd.db.Set([]byte(bucketName), []byte(key), item.data); err != nil {
					return err
				}
				item.updates = 0
				item.storedAt = time.Now()
			}
			return nil
		})
	})
	return err
}

// Close flushes all cached data, clears the cache, and closes the underlying database.
// This ensures no data loss when shutting down the cached database.
//
// Returns:
//   - error: Any error that occurred during flush or close
func (cd *CachedDB) Close() error {
	if err := cd.Flush(); err != nil {
		return err
	}
	cd.store.Clear()
	return cd.db.Close()
}

// ForEach iterates over all key-value pairs in the specified bucket.
// It combines cached entries with persisted data from the underlying database,
// ensuring each key is visited only once (cached entries take precedence).
//
// Parameters:
//   - bucket: The bucket name to iterate over
//   - fn: A function called for each key-value pair
//
// Returns:
//   - error: Any error that occurred during iteration
func (cd *CachedDB) ForEach(bucket []byte, fn func(key []byte, value []byte) error) error {
	b := cd.store.Get(string(bucket))
	if b != nil {
		visited := make(map[string]struct{})
		err := b.ForEach(func(k string, item *cached) error {
			if _, ok := visited[k]; ok {
				return nil
			}
			visited[k] = struct{}{}
			return fn([]byte(k), item.data)
		})
		if err != nil {
			return err
		}

		wrapFn := func(k []byte, v []byte) error {
			if _, ok := visited[string(k)]; ok {
				return nil
			}
			visited[string(k)] = struct{}{}
			return fn(k, v)
		}
		return cd.db.ForEach(bucket, wrapFn)
	}
	return cd.db.ForEach(bucket, fn)
}

// Write executes a custom write transaction on the underlying database.
// IMPORTANT: This method clears the entire cache to maintain consistency,
// as the transaction may modify data that bypasses the cache layer.
//
// Use this method sparingly. Prefer Set/Delete/Batch operations when possible
// to maintain cache effectiveness.
//
// Parameters:
//   - fn: The transaction function to execute
//
// Returns:
//   - error: Any error that occurred during the transaction
func (cd *CachedDB) Write(fn func(tx *bolt.Tx) error) error {
	defer cd.store.Clear()
	return cd.db.Write(fn)
}

// Read executes a read-only transaction on the underlying database.
// This method does not interact with the cache and reads directly from disk.
//
// Parameters:
//   - fn: The read transaction function to execute
//
// Returns:
//   - error: Any error that occurred during the transaction
func (cd *CachedDB) Read(fn func(tx *bolt.Tx) error) error {
	return cd.db.Read(fn)
}

// NewBatch creates a new write batch for the cached database.
// Operations added to the batch will be written to cache first when executed.
//
// Returns:
//   - *Batch: A new write batch instance
func (cd *CachedDB) NewBatch() *Batch {
	return NewBatch(cd)
}

// List returns all key-value pairs from the specified bucket.
// This method combines cached entries with persisted data.
//
// Parameters:
//   - bucket: The bucket name to list
//
// Returns:
//   - map[string][]byte: A map of all key-value pairs in the bucket
//   - error: Any error that occurred during the operation
func (cd *CachedDB) List(bucket []byte) (map[string][]byte, error) {
	result := make(map[string][]byte)
	err := cd.ForEach(bucket, func(key, value []byte) error {
		result[string(key)] = value
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Buckets returns a list of all bucket names in the database.
//
// Returns:
//   - [][]byte: A list of all bucket names
func (cd *CachedDB) Buckets() [][]byte {
	return cd.db.Buckets()
}

// DropBucket deletes an entire bucket from both cache and database.
//
// Parameters:
//   - bucketName: The name of the bucket to delete
//
// Returns:
//   - error: Any error that occurred during the operation
func (cd *CachedDB) DropBucket(bucketName string) error {
	cd.store.Delete(bucketName)
	_ = cd.db.DropBucket(bucketName)
	return nil
}

// executeBatch executes multiple write operations through the cache layer.
// Each operation is processed via Set/Delete methods to ensure proper cache behavior.
// Operations are written to cache first and persisted based on cache thresholds.
//
// Parameters:
//   - ops: Map of bucket names to their operations
//
// Returns:
//   - error: Any error that occurred during execution
func (cd *CachedDB) executeBatch(ops map[string][]*WriteOperation) error {
	for bucketName, bucketOps := range ops {
		bucketBytes := []byte(bucketName)
		for _, op := range bucketOps {
			switch op.Op {
			case OpSet:
				if op.Value == nil {
					return errors.New("value is nil")
				}
				if err := cd.Set(bucketBytes, op.Key, *op.Value); err != nil {
					return err
				}
			case OpDelete:
				if err := cd.Delete(bucketBytes, op.Key); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Lock acquires a write lock on this cached database.
// This is used by WriteLock to coordinate multi-database transactions.
func (cd *CachedDB) Lock() {
	cd.lck.Lock()
}

// Unlock releases the write lock on this cached database.
func (cd *CachedDB) Unlock() {
	cd.lck.Unlock()
}
