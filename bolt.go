package boltdb

import (
	"errors"

	"github.com/boltdb/bolt"
)

// BoltDatabase represents a single Bolt database instance with basic CRUD operations.
// It provides a simple interface for key-value storage operations on Bolt databases.
type BoltDatabase struct {
	db     *bolt.DB // The underlying Bolt database instance
	dbPath string   // File path where the database is stored
}

// NewBoltDatabase creates a new Bolt database instance at the specified path.
// The database file will be created with read/write permissions (0600).
//
// Parameters:
//   - dbPath: The file path where the database should be created/opened
//
// Returns:
//   - *BoltDatabase: A new database instance, or nil if opening fails
func NewBoltDatabase(dbPath string) *BoltDatabase {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil
	}
	return &BoltDatabase{db: db, dbPath: dbPath}
}

// NewBatch creates a new write batch for the database.
// The batch can be used to perform multiple write operations in a single transaction.
//
// Returns:
//   - *BoltBatch: A new write batch instance
func (b *BoltDatabase) NewBatch() *BoltBatch {
	return NewBoltBatch(b)
}

// Close closes the database connection and releases all resources.
// This method should be called when the database is no longer needed.
//
// Returns:
//   - error: Any error that occurred during closing, or nil if successful
func (b *BoltDatabase) Close() error {
	return b.db.Close()
}

// Delete removes a key-value pair from the specified bucket.
// If the bucket doesn't exist, an error is returned.
//
// Parameters:
//   - bucketName: The name of the bucket to delete from
//   - key: The key to delete
//
// Returns:
//   - error: An error if the bucket doesn't exist or deletion fails
func (b *BoltDatabase) Delete(bucketName string, key string) error {
	return b.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return errors.New("bucket not found")
		}
		return bucket.Delete([]byte(key))
	})
}

// Set stores a key-value pair in the specified bucket.
// If the bucket doesn't exist, it will be created automatically.
//
// Parameters:
//   - bucketName: The name of the bucket to store the data in
//   - key: The key to store
//   - value: The value to store (as bytes)
//
// Returns:
//   - error: An error if the operation fails
func (b *BoltDatabase) Set(bucketName string, key string, value []byte) error {
	return b.db.Batch(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		return bucket.Put([]byte(key), value)
	})
}

// Get retrieves a value from the specified bucket by key.
// If the bucket doesn't exist or the key is not found, nil is returned.
//
// Parameters:
//   - bucketName: The name of the bucket to retrieve from
//   - key: The key to retrieve
//
// Returns:
//   - []byte: The value associated with the key, or nil if not found
//   - error: Any error that occurred during the operation
func (b *BoltDatabase) Get(bucketName, key string) ([]byte, error) {
	var result []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return nil
		}

		result = bucket.Get([]byte(key))
		return nil
	})

	return result, err
}

// List returns all key-value pairs from the specified bucket.
// If the bucket doesn't exist, an empty map is returned.
//
// Parameters:
//   - bucketName: The name of the bucket to list
//
// Returns:
//   - map[string][]byte: A map of all key-value pairs in the bucket
//   - error: Any error that occurred during the operation
func (b *BoltDatabase) List(bucketName string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			result[string(k)] = v
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Buckets returns a list of all bucket names in the database.
//
// Returns:
//   - []string: A list of all bucket names in the database
//   - error: Any error that occurred during the operation
func (b *BoltDatabase) Buckets() []string {
	result := make([]string, 0)
	err := b.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			result = append(result, string(name))
			return nil
		})
	})
	if err != nil {
		return nil
	}
	return result
}

// ForEach iterates over all key-value pairs in the specified bucket.
//
// Parameters:
//   - bucketName: The name of the bucket to iterate over
//   - fn: A function that will be called for each key-value pair
//
// Returns:
//   - error: Any error that occurred during the operation
func (b *BoltDatabase) ForEach(bucketName string, fn func(key, value []byte) error) error {
	return b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			return fn(k, v)
		})
	})
}
