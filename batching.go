package boltfactory

import (
	"errors"
	"sync"

	"github.com/boltdb/bolt"
	"golang.org/x/sync/errgroup"
)

// WriteOp represents the type of write operation
type WriteOp = string

// Operation type constants
const (
	OpSet    WriteOp = "set"    // Set operation to store a key-value pair
	OpDelete WriteOp = "delete" // Delete operation to remove a key

	MAX_CONCURRENT_OPERATIONS = 10
	MAX_SEQUENTIAL_OPERATIONS = 5_000 // recommended by bolt docs batch should be less than 10_000
)

// WriteOperation represents a single write operation to be executed in a batch.
// It contains all the information needed to perform the operation.
type WriteOperation struct {
	Bucket []byte  // The bucket name as bytes
	Key    []byte  // The key as bytes
	Value  *[]byte // The value as bytes (nil for delete operations)
	Op     WriteOp // The operation type (set or delete)
}

// BoltBatch provides a thread-safe way to batch multiple write operations.
// It groups operations by bucket and can execute them either sequentially or concurrently.
// This is useful for improving performance when performing many write operations.
type BoltBatch struct {
	lck sync.Mutex
	// bucket -> operations
	ops map[string][]*WriteOperation

	boltdb *BoltDatabase
}

// NewBoltBatch creates a new write batch for the specified database.
//
// Parameters:
//   - db: The database instance to create a batch for
//
// Returns:
//   - *BoltBatch: A new write batch instance
func NewBoltBatch(db *BoltDatabase) *BoltBatch {
	return &BoltBatch{
		ops:    make(map[string][]*WriteOperation, 0),
		boltdb: db,
	}
}

// Add adds a write operation to the batch.
// Operations are grouped by bucket for efficient execution.
//
// Parameters:
//   - op: The write operation to add to the batch
func (b *BoltBatch) Add(op *WriteOperation) error {
	b.lck.Lock()
	defer b.lck.Unlock()
	if len(b.ops) >= MAX_SEQUENTIAL_OPERATIONS {
		return errors.New("max sequential operations reached")
	}
	b.ops[string(op.Bucket)] = append(b.ops[string(op.Bucket)], op)
	return nil
}

// SetDB sets the database instance for this batch.
// This is useful when you need to change the target database after creating the batch.
//
// Parameters:
//   - db: The new database instance
func (b *BoltBatch) SetDB(db *BoltDatabase) {
	b.boltdb = db
}

// ExecuteConcurrent executes all operations in the batch concurrently.
// Operations are grouped by bucket and executed in separate goroutines.
// A semaphore limits the number of concurrent operations to 10.
//
// Returns:
//   - error: Any error that occurred during execution
func (b *BoltBatch) ExecuteConcurrent() error {
	b.lck.Lock()
	defer b.lck.Unlock()

	wg := errgroup.Group{}
	semaphore := make(chan struct{}, min(MAX_CONCURRENT_OPERATIONS, len(b.ops)))

	for bucket, ops := range b.ops {
		wg.Go(func() error {
			semaphore <- struct{}{}

			defer func() {
				<-semaphore
			}()
			return b.boltdb.db.Batch(func(tx *bolt.Tx) error {
				return b.execOpsByBucket(tx, bucket, ops)
			})
		})
	}
	return wg.Wait()
}

// execOpsByBucket executes all operations for a specific bucket within a transaction.
// This is an internal method used by both Execute and ExecuteConcurrent.
//
// Parameters:
//   - tx: The Bolt transaction
//   - bucket: The bucket name
//   - ops: The operations to execute for this bucket
//
// Returns:
//   - error: Any error that occurred during execution
func (b *BoltBatch) execOpsByBucket(tx *bolt.Tx, bucket string, ops []*WriteOperation) error {
	bucketByte := []byte(bucket)
	boltBucket, err := tx.CreateBucketIfNotExists(bucketByte)
	if err != nil {
		return err
	}
	for _, op := range ops {
		switch op.Op {
		case OpSet:
			if op.Value == nil {
				return errors.New("value is nil")
			}
			return boltBucket.Put(op.Key, *op.Value)
		case OpDelete:
			return boltBucket.Delete(op.Key)
		}
	}
	return nil
}

// Execute executes all operations in the batch sequentially.
// Operations are grouped by bucket and executed in separate transactions.
// This method is thread-safe and uses a mutex to prevent concurrent access.
//
// Returns:
//   - error: Any error that occurred during execution
func (b *BoltBatch) Execute() error {
	b.lck.Lock()
	defer b.lck.Unlock()
	for bucket, ops := range b.ops {
		err := b.boltdb.db.Batch(func(tx *bolt.Tx) error {
			return b.execOpsByBucket(tx, bucket, ops)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
