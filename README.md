# Bolt Factory

A Go package that provides a convenient factory pattern for managing multiple [Bolt](https://github.com/boltdb/bolt) databases with dependency injection support.

## Features

- **Multiple Database Management**: Create and manage multiple Bolt databases
- **Thread-Safe Operations**: All factory operations are protected by read-write locks
- **Dependency Injection Support**: Integrates with `dicontainer-go` package
- **Bucket-Specific Wrappers**: Simplified interface for working with specific buckets
- **Batch Operations**: Efficient batch processing for multiple write operations

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "github.com/andrew-solarstorm/bolt-factory"
)

func main() {
    // Create a new database instance
    db := boltfactory.NewBoltDatabase("./myapp.db")
    defer db.Close()

    // Store a value
    err := db.Set("users", "user1", []byte("John Doe"))
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve a value
    value, err := db.Get("users", "user1")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %s\n", string(value))
}
```

## Using the Factory

```go
// Create a factory with an initial database
factory, err := boltfactory.NewBoltFactory("main", "./main.db")
if err != nil {
    log.Fatal(err)
}
defer factory.CloseAll()

// Open additional databases
userDB, err := factory.Open("users", "./users.db")
if err != nil {
    log.Fatal(err)
}

// Store data in different databases
err = factory.Get("main").Set("config", "version", []byte("1.0.0"))
err = userDB.Set("profiles", "user1", []byte("John Doe"))
```

## Using Bucket Wrappers

```go
db := boltfactory.NewBoltDatabase("./app.db")
defer db.Close()

// Create a wrapper for a specific bucket
userWrapper := boltfactory.NewBoltDBWrapper(db, "users")

// Use the wrapper without specifying bucket name
err := userWrapper.Set("user1", []byte("John Doe"))
value, err := userWrapper.Get("user1")
```

## Using ForEach for Iteration

```go
db := boltfactory.NewBoltDatabase("./app.db")
defer db.Close()

// Store some data first
db.Set("users", "user1", []byte("John Doe"))
db.Set("users", "user2", []byte("Jane Smith"))

// Iterate over all key-value pairs in a bucket
err := db.ForEach("users", func(key, value []byte) error {
    fmt.Printf("Key: %s, Value: %s\n", string(key), string(value))
    return nil
})

// Using wrapper for iteration
userWrapper := boltfactory.NewBoltDBWrapper(db, "users")
err = userWrapper.ForEach(func(key, value []byte) error {
    fmt.Printf("User %s: %s\n", string(key), string(value))
    return nil
})

// List all buckets in the database
buckets := db.Buckets()
fmt.Printf("Available buckets: %v\n", buckets)
```

## Using Batch Operations

```go
db := boltfactory.NewBoltDatabase("./app.db")
defer db.Close()

// Create a new batch
batch := db.NewBatch()

// Add multiple operations to the batch
batch.Add(&boltfactory.WriteOperation{
    Bucket: []byte("users"),
    Key:    []byte("user1"),
    Value:  &[]byte("John Doe"),
    Op:     boltfactory.OpSet,
})

batch.Add(&boltfactory.WriteOperation{
    Bucket: []byte("users"),
    Key:    []byte("user2"),
    Value:  &[]byte("Jane Smith"),
    Op:     boltfactory.OpSet,
})

// Execute all operations in a single transaction
err := batch.Execute()
if err != nil {
    log.Fatal(err)
}

// Or execute concurrently for better performance
err = batch.ExecuteConcurrent()
if err != nil {
    log.Fatal(err)
}
```

## API Reference

### BoltDatabase
- `NewBoltDatabase(dbPath string) *BoltDatabase` - Creates a new database
- `Close() error` - Closes the database connection
- `Set(bucketName, key string, value []byte) error` - Stores a key-value pair
- `Get(bucketName, key string) ([]byte, error)` - Retrieves a value
- `Delete(bucketName, key string) error` - Deletes a key-value pair
- `List(bucketName string) (map[string][]byte, error)` - Lists all pairs
- `ForEach(bucketName string, fn func(key, value []byte) error) error` - Iterates over all pairs
- `Buckets() []string` - Returns all bucket names
- `NewBatch() *BoltBatch` - Creates a new write batch

### BoltFactory
- `NewBoltFactory(name, defaultPath string) (*BoltFactory, error)` - Creates factory
- `Open(name, path string) (*BoltDatabase, error)` - Opens a new database
- `Get(name string) (*BoltDatabase, error)` - Retrieves a database
- `Close(name string) error` - Closes a specific database
- `CloseAll() error` - Closes all databases
- `GetDatabases() ([]string, error)` - Lists all database names

### BoltBatch
- `NewBoltBatch(db *BoltDatabase) *BoltBatch` - Creates a new batch
- `Add(op *WriteOperation)` - Adds an operation to the batch
- `Execute() error` - Executes all operations sequentially
- `ExecuteConcurrent() error` - Executes operations concurrently
- `SetDB(db *BoltDatabase)` - Sets the target database

### WriteOperation
- `Bucket []byte` - The bucket name
- `Key []byte` - The key to operate on
- `Value *[]byte` - The value (nil for delete operations)
- `Op WriteOp` - The operation type (OpSet or OpDelete)

### BoltDBWrapper
- `NewBoltDBWrapper(db *BoltDatabase, bucketName string) *BoltDBWrapper` - Creates wrapper
- `Get(key string) ([]byte, error)` - Retrieves a value from the bucket
- `Set(key string, value []byte) error` - Stores a key-value pair in the bucket
- `Delete(key string) error` - Deletes a key from the bucket
- `List() (map[string][]byte, error)` - Lists all pairs in the bucket
- `ForEach(fn func(key, value []byte) error) error` - Iterates over all pairs in the bucket
- `NewBatch() *BoltBatch` - Creates a new write batch

## Environment Variables
- `BOLT_DB_DEFAULT_PATH`: Path for the default database (defaults to `"./bolt.db"`)

## Thread Safety
All factory operations are thread-safe and protected by read-write locks. Batch operations are also thread-safe with mutex protection.

## Performance Tips
- Use batch operations when performing multiple write operations
- Use `ExecuteConcurrent()` for better performance with many operations
- Group operations by bucket for optimal performance
- Consider using bucket wrappers for repeated operations on the same bucket

## Dependencies
- `github.com/boltdb/bolt` - Core Bolt database functionality
- `github.com/andrew-solarstorm/dicontainer-go` - Dependency injection support
- `github.com/andrew-solarstorm/go-packages` - Common utilities 