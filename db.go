package bolt_db

import (
	"errors"
	"sync"

	"github.com/boltdb/bolt"
)

type IDatabase interface {
	Exist(bucket, key []byte) (bool, error)
	Get(bucket, key []byte) ([]byte, error)
	Set(bucket, key, value []byte) error
	Delete(bucket, key []byte) error
	Close() error
	ForEach(bucket []byte, fn func(key, value []byte) error) error
	Write(fn func(tx *bolt.Tx) error) error
	Read(fn func(tx *bolt.Tx) error) error
	NewBatch() *Batch
	List(bucket []byte) (map[string][]byte, error)
	Buckets() [][]byte
	DropBucket(bucketName string) error
	executeBatch(ops map[string][]*WriteOperation) error
	Lock()
	Unlock()
	Name() string
}

type GenericIDatabase[V any] interface {
	Exist(bucket, key []byte) (bool, error)
	Get(bucket, key []byte) (*V, error)
	Set(bucket, key []byte, value *V) error
	Delete(bucket, key []byte) error
	Close() error
	ForEach(bucket []byte, fn func(key []byte, value *V) error) error
	List(bucket []byte) (map[string]*V, error)
	Buckets() [][]byte
	DropBucket(bucketName string) error
	Lock()
	Unlock()
	Name() string
}

// DB represents a single Bolt database instance with basic CRUD operations.
// It provides a simple interface for key-value storage operations on Bolt databases.
type DB struct {
	db     *bolt.DB     // The underlying Bolt database instance
	dbPath string       // File path where the database is stored
	lck    sync.RWMutex // Lock for coordinating multi-DB transactions
}

// NewDB creates a new Bolt database instance at the specified path.
// The database file will be created with read/write permissions (0600).
//
// Parameters:
//   - dbPath: The file path where the database should be created/opened
//
// Returns:
//   - *DB: A new database instance, or nil if opening fails
func NewDB(dbPath string) *DB {
	db, err := bolt.Open(dbPath, 0o600, nil)
	if err != nil {
		return nil
	}
	return &DB{db: db, dbPath: dbPath}
}

// Name returns the database path as its unique identifier.
//
// Returns:
//   - string: The database file path
func (d *DB) Name() string {
	return d.dbPath
}

// NewBatch creates a new write batch for the database.
// The batch can be used to perform multiple write operations in a single transaction.
//
// Returns:
//   - *BoltBatch: A new write batch instance
func (d *DB) NewBatch() *Batch {
	return NewBatch(d)
}

func (d *DB) Exist(bucket, key []byte) (bool, error) {
	var result bool
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		result = b.Get(key) != nil
		return nil
	})
	return result, err
}

// Close closes the database connection and releases all resources.
// This method should be called when the database is no longer needed.
//
// Returns:
//   - error: Any error that occurred during closing, or nil if successful
func (d *DB) Close() error {
	return d.db.Close()
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
func (d *DB) Delete(bucket, key []byte) error {
	return d.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		return b.Delete(key)
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
func (d *DB) Set(bucket, key, value []byte) error {
	return d.db.Batch(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}
		return b.Put(key, value)
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
func (d *DB) Get(bucket, key []byte) ([]byte, error) {
	var result []byte
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}

		result = b.Get(key)
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
func (d *DB) List(bucket []byte) (map[string][]byte, error) {
	result := make(map[string][]byte)
	err := d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
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
func (d *DB) Buckets() [][]byte {
	result := make([][]byte, 0)
	err := d.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			result = append(result, name)
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
func (d *DB) ForEach(bucket []byte, fn func(key, value []byte) error) error {
	return d.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			return fn(k, v)
		})
	})
}

func (d *DB) Write(fn func(tx *bolt.Tx) error) error {
	return d.db.Batch(fn)
}

func (d *DB) Read(fn func(tx *bolt.Tx) error) error {
	return d.db.View(fn)
}

func (d *DB) DropBucket(bucketName string) error {
	_ = d.db.Batch(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(bucketName))
	})

	return nil
}

// executeBatch executes multiple write operations in batched transactions.
// Operations are grouped by bucket and each bucket's operations are executed
// in a single transaction for optimal performance.
//
// Parameters:
//   - ops: Map of bucket names to their operations
//
// Returns:
//   - error: Any error that occurred during execution
func (d *DB) executeBatch(ops map[string][]*WriteOperation) error {
	for bucketName, bucketOps := range ops {
		err := d.db.Batch(func(tx *bolt.Tx) error {
			bucketBytes := []byte(bucketName)
			boltBucket, err := tx.CreateBucketIfNotExists(bucketBytes)
			if err != nil {
				return err
			}
			for _, op := range bucketOps {
				switch op.Op {
				case OpSet:
					if op.Value == nil {
						return errors.New("value is nil")
					}
					if err := boltBucket.Put(op.Key, *op.Value); err != nil {
						return err
					}
				case OpDelete:
					if err := boltBucket.Delete(op.Key); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Lock acquires a write lock on this database.
// This is used by WriteLock to coordinate multi-database transactions.
func (d *DB) Lock() {
	d.lck.Lock()
}

// Unlock releases the write lock on this database.
func (d *DB) Unlock() {
	d.lck.Unlock()
}
