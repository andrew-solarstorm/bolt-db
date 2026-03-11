package bolt_db

import "github.com/boltdb/bolt"

// GenericDB provides type-safe database operations with automatic serialization.
// It wraps a DB instance and handles encoding/decoding of generic types to/from bytes.
// Default encoding uses gob serialization, but custom encoders/decoders can be provided.
type GenericDB[T any] struct {
	db IDatabase // Underlying database instance

	dec func(d []byte, out any) error // Decoder function (gob by default)
	enc func(in any) ([]byte, error)  // Encoder function (gob by default)
}

// NewGenericDB creates a new type-safe database wrapper with optional custom serialization.
// If dec or enc are nil, default gob encoding/decoding will be used.
//
// Parameters:
//   - db: The underlying DB instance
//   - dec: Custom decoder function (nil = use gob)
//   - enc: Custom encoder function (nil = use gob)
//
// Returns:
//   - *GenericDB[T]: A new type-safe database instance
func NewGenericDB[T any](
	db IDatabase,
	dec func(d []byte, out any) error,
	enc func(in any) ([]byte, error),
) *GenericDB[T] {
	if dec == nil {
		dec = defaultDec
	}
	if enc == nil {
		enc = defaultEnc
	}
	return &GenericDB[T]{
		db:  db,
		enc: enc,
		dec: dec,
	}
}

// Get retrieves a typed value from the specified bucket and key.
// The raw bytes are automatically decoded into type T.
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to retrieve
//
// Returns:
//   - *T: Pointer to the decoded value, or nil if not found
//   - error: Any error that occurred during retrieval or decoding
func (d *GenericDB[T]) Get(bucket, key []byte) (*T, error) {
	raw, err := d.db.Get(bucket, key)
	if err != nil {
		return nil, err
	}
	out := new(T)
	err = d.dec(raw, out)
	if err != nil {
		return nil, err
	}

	copied := *out
	return &copied, nil
}

// Set stores a typed value in the specified bucket and key.
// The value is automatically encoded to bytes before storage.
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to store
//   - data: Pointer to the value to store
//
// Returns:
//   - error: Any error that occurred during encoding or storage
func (d *GenericDB[T]) Set(bucket, key []byte, data *T) error {
	raw, err := d.enc(data)
	if err != nil {
		return err
	}

	return d.db.Set(bucket, key, raw)
}

// Delete removes a key-value pair from the specified bucket.
//
// Parameters:
//   - bucket: The bucket name
//   - key: The key to delete
//
// Returns:
//   - error: Any error that occurred during deletion
func (d *GenericDB[T]) Delete(bucket, key []byte) error {
	return d.db.Delete(bucket, key)
}

// Close closes the underlying database connection.
//
// Returns:
//   - error: Any error that occurred during closing
func (d *GenericDB[T]) Close() error {
	return d.db.Close()
}

// ForEach iterates over all key-value pairs in the specified bucket.
// Values are automatically decoded into type T before being passed to the callback.
//
// Parameters:
//   - bucket: The bucket name to iterate over
//   - fn: A function called for each key-value pair with decoded value
//
// Returns:
//   - error: Any error that occurred during iteration or decoding
func (d *GenericDB[T]) ForEach(bucket []byte, fn func(key []byte, value *T) error) error {
	return d.db.ForEach(bucket, func(key []byte, value []byte) error {
		out := new(T)
		err := d.dec(value, out)
		if err != nil {
			return err
		}
		return fn(key, out)
	})
}

// Write executes a custom write transaction on the underlying database.
//
// Parameters:
//   - fn: The transaction function to execute
//
// Returns:
//   - error: Any error that occurred during the transaction
func (d *GenericDB[T]) Write(fn func(tx *bolt.Tx) error) error {
	return d.db.Write(fn)
}

// Read executes a read-only transaction on the underlying database.
//
// Parameters:
//   - fn: The read transaction function to execute
//
// Returns:
//   - error: Any error that occurred during the transaction
func (d *GenericDB[T]) Read(fn func(tx *bolt.Tx) error) error {
	return d.db.Read(fn)
}
