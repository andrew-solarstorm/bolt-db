package bolt_db

import (
	"time"

	"github.com/boltdb/bolt"
)

// gCached represents a single cached entry for generic typed data with metadata
// for cache management. It tracks the data, number of updates, and timing information
// for cache eviction and persistence decisions.
type gCached[T any] struct {
	data     *T        // Pointer to the cached data of type T
	updates  int       // Number of updates since last persistence
	updateAt time.Time // Time of the last update
	storedAt time.Time // Time when data was last persisted to disk
}

// GenericCachedDB provides an in-memory caching layer on top of a GenericDB.
// It implements intelligent write-through caching with configurable persistence
// strategies based on update frequency, time intervals, and velocity.
//
// The caching strategy uses three key parameters:
//   - updateThreshold: Number of updates before considering persistence
//   - updateInterval: Maximum time between disk writes
//   - deleteInterval: Time of inactivity before cache eviction
//
// Data is persisted when:
//   1. Updates exceed threshold AND velocity is low (≤ 1 update/second)
//   2. Time since last persist exceeds updateInterval
//   3. Manual Flush() is called
type GenericCachedDB[T any] struct {
	db              *GenericDB[T]                                      // Underlying generic database instance
	store           *InMemKV[string, *InMemKV[string, *gCached[T]]]   // Two-level cache: bucket -> key -> cached data
	updateThreshold int                                                // Number of updates before persistence check
	updateInterval  time.Duration                                      // Maximum time between disk writes
	deleteInterval  time.Duration                                      // Time of inactivity before cache eviction
}

// copy creates a shallow copy of the data to prevent external modifications
// from affecting cached data.
//
// Parameters:
//   - data: Pointer to the data to copy
//
// Returns:
//   - *T: A new pointer to copied data
func (cd *GenericCachedDB[T]) copy(data *T) *T {
	copied := *data
	return &copied
}

// NewGenericCachedDB creates a new generic cached database instance with the specified caching parameters.
// If any parameter is <= 0, default values are used:
//   - updateThreshold: 300 updates
//   - updateInterval: 5 minutes
//   - deleteInterval: 15 minutes
//
// The cache uses a two-level map structure (bucket -> key -> cached data) with
// thread-safe operations via read-write locks.
//
// Parameters:
//   - db: The underlying GenericDB instance to cache
//   - updateThreshold: Number of updates before considering persistence (0 = use default)
//   - updateInterval: Maximum time between disk writes (0 = use default)
//   - deleteInterval: Time of inactivity before cache eviction (0 = use default)
//
// Returns:
//   - *GenericCachedDB[T]: A new generic cached database instance
func NewGenericCachedDB[T any](
	db *GenericDB[T],
	updateThreshold int,
	updateInterval, deleteInterval time.Duration,
) *GenericCachedDB[T] {
	if updateThreshold <= 0 {
		updateThreshold = 300
	}
	if updateInterval <= 0 {
		updateInterval = 5 * time.Minute
	}
	if deleteInterval <= 0 {
		deleteInterval = 15 * time.Minute
	}
	return &GenericCachedDB[T]{
		db:              db,
		store:           NewInmemMap[string, *InMemKV[string, *gCached[T]]](),
		updateThreshold: updateThreshold,
		updateInterval:  updateInterval,
		deleteInterval:  deleteInterval,
	}
}

// Get retrieves a typed value from the specified bucket and key.
// The method first checks the cache. If not found in cache, it retrieves
// from the underlying database and stores in cache for future reads.
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to retrieve
//
// Returns:
//   - *T: A copy of the cached or retrieved data, or nil if not found
//   - error: Any error that occurred during the operation
func (cd *GenericCachedDB[T]) Get(bucket, key []byte) (*T, error) {
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
		b = NewInmemMap[string, *gCached[T]]()
		cd.store.Set(string(bucket), b)
	}

	now := time.Now()
	b.Set(string(key), &gCached[T]{
		data:     data,
		updates:  0,
		storedAt: now,
		updateAt: now,
	})

	return cd.copy(data), nil
}

// Set stores a typed key-value pair in the cache and conditionally persists to disk.
// The data is always written to cache immediately. Persistence to disk happens
// based on the caching strategy (update threshold, interval, and velocity).
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to store
//   - data: Pointer to the typed value to store
//
// Returns:
//   - error: Any error that occurred during the operation
func (cd *GenericCachedDB[T]) Set(bucket, key []byte, data *T) error {
	b := cd.store.Get(string(bucket))
	if b == nil {
		b = NewInmemMap[string, *gCached[T]]()
		cd.store.Set(string(bucket), b)
	}

	item := b.Get(string(key))
	now := time.Now()
	if item == nil {
		item = &gCached[T]{
			data:     data,
			updates:  0,
			storedAt: now,
			updateAt: now,
		}
		b.Set(string(key), item)
	} else {
		item.data = data
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
func (cd *GenericCachedDB[T]) Exist(bucket, key []byte) (bool, error) {
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
func (cd *GenericCachedDB[T]) Delete(bucket, key []byte) error {
	b := cd.store.Get(string(bucket))
	if b != nil {
		b.Delete(string(key))
	}

	return cd.db.Delete(bucket, key)
}

// canSave determines if a cached item should be persisted to disk.
// Persistence occurs when:
//   1. Updates >= threshold AND velocity <= 1.0 update/second
//   2. Time since last persist >= updateInterval AND item has updates
//
// Parameters:
//   - item: The cached item to evaluate
//
// Returns:
//   - bool: true if the item should be persisted
func (cd *GenericCachedDB[T]) canSave(item *gCached[T]) bool {
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
func (cd *GenericCachedDB[T]) canDelete(item *gCached[T]) bool {
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
func (cd *GenericCachedDB[T]) Flush() error {
	err := cd.store.ForEach(func(bucketName string, bucket *InMemKV[string, *gCached[T]]) error {
		return bucket.ForEach(func(key string, item *gCached[T]) error {
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
func (cd *GenericCachedDB[T]) Close() error {
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
//   - fn: A function called for each key-value pair with typed value
//
// Returns:
//   - error: Any error that occurred during iteration
func (cd *GenericCachedDB[T]) ForEach(bucket []byte, fn func(key []byte, value *T) error) error {
	b := cd.store.Get(string(bucket))
	if b != nil {
		visited := make(map[string]struct{})
		err := b.ForEach(func(k string, item *gCached[T]) error {
			if _, ok := visited[k]; ok {
				return nil
			}
			visited[k] = struct{}{}
			return fn([]byte(k), item.data)
		})
		if err != nil {
			return err
		}

		wrapFn := func(k []byte, v *T) error {
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
// Use this method sparingly. Prefer Set/Delete operations when possible
// to maintain cache effectiveness.
//
// Parameters:
//   - fn: The transaction function to execute
//
// Returns:
//   - error: Any error that occurred during the transaction
func (cd *GenericCachedDB[T]) Write(fn func(tx *bolt.Tx) error) error {
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
func (cd *GenericCachedDB[T]) Read(fn func(tx *bolt.Tx) error) error {
	return cd.db.Read(fn)
}
