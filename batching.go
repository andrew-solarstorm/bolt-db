package bolt_db

import (
	"errors"
	"sync"
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

// Batch provides a thread-safe way to batch multiple write operations.
// It groups operations by bucket and can execute them either sequentially or concurrently.
// This is useful for improving performance when performing many write operations.
type Batch struct {
	lck sync.Mutex
	// bucket -> operations
	ops map[string][]*WriteOperation

	db IDatabase
}

// NewBatch creates a new write batch for the specified database.
//
// Parameters:
//   - db: The database instance to create a batch for
//
// Returns:
//   - *BoltBatch: A new write batch instance
func NewBatch(db IDatabase) *Batch {
	return &Batch{
		ops: make(map[string][]*WriteOperation, 0),
		db:  db,
	}
}

// Add adds a write operation to the batch.
// Operations are grouped by bucket for efficient execution.
//
// Parameters:
//   - op: The write operation to add to the batch
func (b *Batch) Add(op *WriteOperation) error {
	b.lck.Lock()
	defer b.lck.Unlock()
	
	// Count total operations across all buckets
	totalOps := 0
	for _, ops := range b.ops {
		totalOps += len(ops)
	}
	
	if totalOps >= MAX_SEQUENTIAL_OPERATIONS {
		return errors.New("max sequential operations reached")
	}
	b.ops[string(op.Bucket)] = append(b.ops[string(op.Bucket)], op)
	return nil
}

// SetDB sets the database instance for this batch.
// This is useful when you need to change the target database after creating the batch.
// The database must implement the IDatabase interface (DB, CachedDB, or ShardDB).
//
// Parameters:
//   - db: The new database instance
func (b *Batch) SetDB(db IDatabase) {
	b.db = db
}

// Execute executes all operations in the batch.
// The execution strategy depends on the database type:
//   - DB: Writes directly to database in transactions
//   - CachedDB: Writes to cache first, respecting cache thresholds
//   - ShardDB: Routes operations to appropriate shards based on bucket
//
// Returns:
//   - error: Any error that occurred during execution
func (b *Batch) Execute() error {
	b.lck.Lock()
	defer b.lck.Unlock()
	if len(b.ops) == 0 {
		return nil
	}

	return b.db.executeBatch(b.ops)
}
