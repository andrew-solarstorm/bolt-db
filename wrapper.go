package boltfactory

// BoltDBWrapper provides a simplified interface for working with a specific bucket
// within a Bolt database. It wraps the BoltDatabase and pre-configures all operations
// to work with a single bucket, eliminating the need to specify the bucket name
// for each operation.
type BoltDBWrapper struct {
	db         *BoltDatabase // The underlying database instance
	bucketName string        // The bucket name this wrapper operates on
}

// NewBatch creates a new write batch for the database.
// The batch can be used to perform multiple write operations in a single transaction.
//
// Returns:
//   - *BoltBatch: A new write batch instance
func (w *BoltDBWrapper) NewBatch() *BoltBatch {
	return w.db.NewBatch()
}

// NewBoltDBWrapper creates a new wrapper for a specific bucket within a database.
// All operations performed through this wrapper will target the specified bucket.
//
// Parameters:
//   - db: The BoltDatabase instance to wrap
//   - bucketName: The name of the bucket this wrapper will operate on
//
// Returns:
//   - *BoltDBWrapper: A new wrapper instance
func NewBoltDBWrapper(db *BoltDatabase, bucketName string) *BoltDBWrapper {
	return &BoltDBWrapper{db: db, bucketName: bucketName}
}

// Get retrieves a value from the configured bucket.
// This is a convenience method that automatically uses the wrapper's bucket name.
//
// Parameters:
//   - key: The key to retrieve
//
// Returns:
//   - []byte: The value associated with the key, or nil if not found
//   - error: Any error that occurred during the operation
func (w *BoltDBWrapper) Get(key string) ([]byte, error) {
	return w.db.Get(w.bucketName, key)
}

// Set stores a value in the configured bucket.
// This is a convenience method that automatically uses the wrapper's bucket name.
//
// Parameters:
//   - key: The key to store
//   - value: The value to store (as bytes)
//
// Returns:
//   - error: Any error that occurred during the operation
func (w *BoltDBWrapper) Set(key string, value []byte) error {
	return w.db.Set(w.bucketName, key, value)
}

// Delete removes a key from the configured bucket.
// This is a convenience method that automatically uses the wrapper's bucket name.
//
// Parameters:
//   - key: The key to delete
//
// Returns:
//   - error: Any error that occurred during the operation
func (w *BoltDBWrapper) Delete(key string) error {
	return w.db.Delete(w.bucketName, key)
}

// List returns all key-value pairs from the configured bucket.
// This is a convenience method that automatically uses the wrapper's bucket name.
//
// Returns:
//   - map[string][]byte: A map of all key-value pairs in the bucket
//   - error: Any error that occurred during the operation
func (w *BoltDBWrapper) List() (map[string][]byte, error) {
	return w.db.List(w.bucketName)
}

// ForEach iterates over all key-value pairs in the configured bucket.
// This is a convenience method that automatically uses the wrapper's bucket name.
//
// Parameters:
//   - fn: A function that will be called for each key-value pair
//
// Returns:
//   - error: Any error that occurred during the operation
func (w *BoltDBWrapper) ForEach(fn func(key, value []byte) error) error {
	return w.db.ForEach(w.bucketName, fn)
}
