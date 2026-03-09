# Bolt DB

A Go package that provides a convenient interface for managing [Bolt](https://github.com/boltdb/bolt) databases with caching, generic types, and batch operations.

## Features

- **Multiple Database Management**: Create and manage multiple Bolt databases with factory pattern
- **Cached Database**: In-memory caching layer with configurable write-through strategies
- **Sharded Database**: Horizontal partitioning across multiple DB files for better concurrency
- **Multi-Database Transactions**: WriteLock for atomic operations across multiple databases
- **Generic Type Support**: Type-safe database operations with automatic serialization
- **Thread-Safe Operations**: All operations are protected by read-write locks
- **Bucket-Specific Wrappers**: Simplified interface for working with specific buckets
- **Batch Operations**: Efficient batch processing with cache-aware execution
- **Interface-Based Design**: `IDatabase` interface for interchangeable DB, CachedDB, and ShardDB usage

## Quick Start

### Basic Database Usage

```go
package main

import (
    "fmt"
    "log"
    "github.com/andrew-solarstorm/bolt-db"
)

func main() {
    // Create a new database instance
    db := boltdb.NewDB("./myapp.db")
    defer db.Close()

    // Store a value
    err := db.Set([]byte("users"), []byte("user1"), []byte("John Doe"))
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve a value
    value, err := db.Get([]byte("users"), []byte("user1"))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %s\n", string(value))
}
```

### Using Cached Database

```go
package main

import (
    "log"
    "time"
    "github.com/andrew-solarstorm/bolt-db"
)

func main() {
    // Create base database
    db := boltdb.NewDB("./myapp.db")
    
    // Wrap with caching layer
    // Parameters: updateThreshold (300), updateInterval (5min), deleteInterval (15min)
    cachedDB := boltdb.NewCachedDB(db, 300, 5*time.Minute, 15*time.Minute)
    defer cachedDB.Close()

    // Operations go through cache first
    err := cachedDB.Set([]byte("users"), []byte("user1"), []byte("John Doe"))
    if err != nil {
        log.Fatal(err)
    }

    // Read from cache if available
    value, err := cachedDB.Get([]byte("users"), []byte("user1"))
    if err != nil {
        log.Fatal(err)
    }

    // Manually flush cache to disk
    cachedDB.Flush()
}
```

## Using Generic Type-Safe Database

```go
package main

import (
    "log"
    "github.com/andrew-solarstorm/bolt-db"
)

type User struct {
    Name  string
    Email string
    Age   int
}

func main() {
    db := boltdb.NewDB("./app.db")
    defer db.Close()

    // Create a generic database with automatic serialization
    userDB := boltdb.NewGenericDB[User](db, nil, nil)

    // Store typed data
    user := &User{Name: "John", Email: "john@example.com", Age: 30}
    err := userDB.Set([]byte("users"), []byte("user1"), user)
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve typed data
    retrieved, err := userDB.Get([]byte("users"), []byte("user1"))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %+v\n", *retrieved)
}
```

## Using Generic Cached Database

```go
db := boltdb.NewDB("./app.db")
userDB := boltdb.NewGenericDB[User](db, nil, nil)

// Wrap with caching layer for typed data
cachedUserDB := boltdb.NewGenericCachedDB[User](userDB, 300, 5*time.Minute, 15*time.Minute)
defer cachedUserDB.Close()

user := &User{Name: "John", Email: "john@example.com", Age: 30}
cachedUserDB.Set([]byte("users"), []byte("user1"), user)
```

## Using Sharded Database

Sharding distributes buckets across multiple database files for better write throughput:

```go
package main

import (
    "log"
    "github.com/andrew-solarstorm/bolt-db"
)

func main() {
    // Create a factory
    factory, err := boltdb.NewBoltFactory("main", "./main.db")
    if err != nil {
        log.Fatal(err)
    }
    defer factory.CloseAll()

    // Create sharded database with 4 shards
    // Files will be named: myapp_shard0.db, myapp_shard1.db, etc.
    shardDB, err := boltdb.NewShardDB(factory, "myapp", 4)
    if err != nil {
        log.Fatal(err)
    }
    defer shardDB.Close()

    // Operations are automatically routed to the correct shard
    shardDB.Set([]byte("users"), []byte("user1"), []byte("John Doe"))
    shardDB.Set([]byte("orders"), []byte("order1"), []byte("Order Data"))

    // Get from correct shard automatically
    value, _ := shardDB.Get([]byte("users"), []byte("user1"))
}
```

## Using the Factory

```go
// Create a factory with an initial database
factory, err := boltdb.NewBoltFactory("main", "./main.db")
if err != nil {
    log.Fatal(err)
}
defer factory.CloseAll()

// Open additional databases
userDB, err := factory.Open("users", "./users.db")
if err != nil {
    log.Fatal(err)
}

// Retrieve and use databases
mainDB, _ := factory.Get("main")
mainDB.Set([]byte("config"), []byte("version"), []byte("1.0.0"))
userDB.Set([]byte("profiles"), []byte("user1"), []byte("John Doe"))
```

## Using Bucket Wrappers

```go
db := boltdb.NewDB("./app.db")
defer db.Close()

// Create a wrapper for a specific bucket
userWrapper := boltdb.NewDBWrapper(db, []byte("users"))

// Use the wrapper without specifying bucket name
err := userWrapper.Set([]byte("user1"), []byte("John Doe"))
value, err := userWrapper.Get([]byte("user1"))
```

## Using ForEach for Iteration

```go
db := boltdb.NewDB("./app.db")
defer db.Close()

// Store some data first
db.Set([]byte("users"), []byte("user1"), []byte("John Doe"))
db.Set([]byte("users"), []byte("user2"), []byte("Jane Smith"))

// Iterate over all key-value pairs in a bucket
err := db.ForEach([]byte("users"), func(key, value []byte) error {
    fmt.Printf("Key: %s, Value: %s\n", string(key), string(value))
    return nil
})

// Using wrapper for iteration
userWrapper := boltdb.NewDBWrapper(db, []byte("users"))
err = userWrapper.ForEach(func(key, value []byte) error {
    fmt.Printf("User %s: %s\n", string(key), string(value))
    return nil
})

// List all buckets in the database
buckets := db.Buckets()
fmt.Printf("Available buckets: %v\n", buckets)
```

## Using Batch Operations

Batch operations automatically use the appropriate strategy based on database type:
- For `DB`: Writes in optimized transactions
- For `CachedDB`: Writes to cache first, respecting cache thresholds

```go
db := boltdb.NewDB("./app.db")
defer db.Close()

// Create a new batch
batch := db.NewBatch()

// Add multiple operations to the batch
val1 := []byte("John Doe")
batch.Add(&boltdb.WriteOperation{
    Bucket: []byte("users"),
    Key:    []byte("user1"),
    Value:  &val1,
    Op:     boltdb.OpSet,
})

val2 := []byte("Jane Smith")
batch.Add(&boltdb.WriteOperation{
    Bucket: []byte("users"),
    Key:    []byte("user2"),
    Value:  &val2,
    Op:     boltdb.OpSet,
})

// Execute all operations
err := batch.Execute()
if err != nil {
    log.Fatal(err)
}
```

### Batch Operations with CachedDB

```go
cachedDB := boltdb.NewCachedDB(db, 300, 5*time.Minute, 15*time.Minute)
defer cachedDB.Close()

// Batch operations on CachedDB write to cache first
batch := cachedDB.NewBatch()
val := []byte("User Data")
batch.Add(&boltdb.WriteOperation{
    Bucket: []byte("users"),
    Key:    []byte("user1"),
    Value:  &val,
    Op:     boltdb.OpSet,
})

// All operations go through cache layer
batch.Execute()
```

## Using WriteLock for Multi-Database Atomic Operations

`WriteLock` allows you to coordinate operations across multiple databases atomically by locking all involved databases before executing functions. Databases are identified by their `Name()` method and automatically deduplicated to prevent deadlocks.

```go
package main

import (
    "log"
    "github.com/andrew-solarstorm/bolt-db"
)

func main() {
    orderDB := boltdb.NewDB("./orders.db")
    userDB := boltdb.NewDB("./users.db")
    defer orderDB.Close()
    defer userDB.Close()

    // Create a WriteLock
    wl := boltdb.NewWriteLock()

    // Add database and function pairs
    // Each Add() registers a database to lock and a function to execute
    orderData := []byte("Buy 100 shares AAPL at $150")
    wl.Add(orderDB, func() error {
        return orderDB.Set([]byte("orders"), []byte("order123"), orderData)
    })

    balanceData := []byte("Balance: $9000.00")
    wl.Add(userDB, func() error {
        return userDB.Set([]byte("balances"), []byte("user1"), balanceData)
    })

    // Execute: locks both databases (sorted by name to prevent deadlocks),
    // runs all functions in order, then unlocks all databases
    err := wl.Execute()
    if err != nil {
        log.Fatal(err)
    }
}
```

### WriteLock with ShardDB

When using `ShardDB`, get the specific shard DBs you need to lock to avoid locking the entire sharded database:

```go
factory, _ := boltdb.NewBoltFactory("main", "./main.db")
shardDB, _ := boltdb.NewShardDB(factory, "myapp", 4)

// Get the specific shards for the buckets you'll modify
orderShardDB, _ := shardDB.GetShardForBucket([]byte("orders"))
userShardDB, _ := shardDB.GetShardForBucket([]byte("users"))

wl := boltdb.NewWriteLock()

orderData := []byte("Order Data")
wl.Add(orderShardDB, func() error {
    return orderShardDB.Set([]byte("orders"), []byte("order1"), orderData)
})

userData := []byte("User Data")
wl.Add(userShardDB, func() error {
    return userShardDB.Set([]byte("users"), []byte("user1"), userData)
})

// Only locks the 2 specific shards, not the whole ShardDB
wl.Execute()
```

### WriteLock Benefits

- **Atomic Operations**: All functions execute atomically or none at all
- **Deadlock Prevention**: Databases locked in sorted name order
- **Automatic Deduplication**: Same database added multiple times only locks once
- **Clean API**: Simple Add/Execute pattern

## API Reference

### IDatabase Interface
Common interface implemented by `DB`, `CachedDB`, and `ShardDB`:
- `Get(bucket, key []byte) ([]byte, error)` - Retrieves a value
- `Set(bucket, key, value []byte) error` - Stores a key-value pair
- `Delete(bucket, key []byte) error` - Deletes a key-value pair
- `Close() error` - Closes the database connection
- `ForEach(bucket []byte, fn func(key, value []byte) error) error` - Iterates over all pairs
- `Write(fn func(tx *bolt.Tx) error) error` - Executes a write transaction
- `Read(fn func(tx *bolt.Tx) error) error` - Executes a read transaction
- `NewBatch() *Batch` - Creates a new write batch
- `List(bucket []byte) (map[string][]byte, error)` - Lists all pairs in a bucket
- `Buckets() [][]byte` - Returns all bucket names
- `DropBucket(bucketName string) error` - Deletes an entire bucket
- `Lock()` - Acquires write lock (for multi-DB transactions)
- `Unlock()` - Releases write lock
- `Name() string` - Returns unique database identifier

### DB
- `NewDB(dbPath string) *DB` - Creates a new database instance
- `Name() string` - Returns the database file path

### CachedDB
- `NewCachedDB(db *DB, updateThreshold int, updateInterval, deleteInterval time.Duration) *CachedDB` - Creates cached database
- `Flush() error` - Flushes all cached data to disk
- `Exist(bucket, key []byte) (bool, error)` - Checks if a key exists (cache-aware)
- `Name() string` - Returns "cached:" + underlying DB path

**Caching Strategy:**
- Writes go to cache first and persist to disk based on:
  - **Update threshold**: Number of updates before persisting (default: 300)
  - **Update interval**: Time since last persist (default: 5 minutes)
  - **Delete interval**: Time of inactivity before cache eviction (default: 15 minutes)
  - **Velocity check**: Updates per second must be ≤ 1.0

### GenericDB[T]
- `NewGenericDB[T](db *DB, dec, enc func) *GenericDB[T]` - Creates type-safe database
- All standard database operations with type `*T` instead of `[]byte`

### GenericCachedDB[T]
- `NewGenericCachedDB[T](db *GenericDB[T], updateThreshold int, updateInterval, deleteInterval time.Duration) *GenericCachedDB[T]`
- Type-safe cached database with same caching strategy as `CachedDB`

### ShardDB
- `NewShardDB(factory *Factory, name string, shards int) (*ShardDB, error)` - Creates sharded database with name prefix
- `GetShardForBucket(bucket []byte) (*DB, error)` - Gets the specific shard DB for a bucket
- `Name() string` - Returns the sharded database name
- Implements `IDatabase` interface
- Sharding based on bucket name (first byte + last byte)
- All keys in the same bucket are in the same shard
- All standard database operations automatically route to correct shard
- Lock operations are no-ops (lock individual shards via GetShardForBucket instead)

### BoltFactory
- `NewBoltFactory(name, defaultPath string) (*BoltFactory, error)` - Creates factory
- `Open(name, path string) (*DB, error)` - Opens a new database
- `Get(name string) (*DB, error)` - Retrieves a database
- `Close(name string) error` - Closes a specific database
- `CloseAll() error` - Closes all databases
- `GetDatabases() ([]string, error)` - Lists all database names

### Batch
- `NewBatch(db IDatabase) *Batch` - Creates a new batch
- `Add(op *WriteOperation) error` - Adds an operation to the batch
- `Execute() error` - Executes all operations (cache-aware for CachedDB)
- `SetDB(db IDatabase)` - Sets the target database

### WriteLock
- `NewWriteLock() *WriteLock` - Creates a new multi-DB transaction coordinator
- `Add(db IDatabase, fn func() error)` - Adds a database/function pair (deduplicates by db.Name())
- `Execute() error` - Locks all databases (sorted by name), executes functions, then unlocks

### WriteOperation
- `Bucket []byte` - The bucket name
- `Key []byte` - The key to operate on
- `Value *[]byte` - The value (nil for delete operations)
- `Op WriteOp` - The operation type (OpSet or OpDelete)

### DBWrapper
- `NewDBWrapper(db IDatabase, bucket []byte) *DBWrapper` - Creates wrapper
- `Get(key []byte) ([]byte, error)` - Retrieves a value from the bucket
- `Set(key, value []byte) error` - Stores a key-value pair in the bucket
- `Delete(key []byte) error` - Deletes a key from the bucket
- `List() (map[string][]byte, error)` - Lists all pairs in the bucket
- `ForEach(fn func(key, value []byte) error) error` - Iterates over all pairs in the bucket
- `NewBatch() *Batch` - Creates a new write batch

## Caching Behavior

The `CachedDB` provides intelligent write-through caching with:

1. **Update Threshold**: Persist after N updates if velocity is low (default: 300 updates)
2. **Update Interval**: Persist if time since last persist exceeds interval (default: 5 minutes)
3. **Delete Interval**: Remove from cache after period of inactivity (default: 15 minutes)
4. **Velocity Check**: Only persist if update rate ≤ 1 update/second

### When to Use CachedDB
- High-frequency writes to same keys
- Read-heavy workloads with occasional writes
- Applications that can tolerate eventual consistency
- When you want to reduce disk I/O

### When to Use Regular DB
- Strict consistency requirements
- Infrequent writes
- When immediate persistence is critical

## Sharding Behavior

The `ShardDB` distributes buckets across multiple database files:

**Sharding Strategy:**
- Shard determined by: `(bucket[0] + bucket[len(bucket)-1]) % numShards`
- All keys in the same bucket are stored in the same shard
- Each shard is an independent database file
- Operations automatically route to the correct shard

**Benefits:**
- Improved write throughput by reducing contention
- Better concurrent access patterns
- Isolates lock contention per bucket
- Can scale horizontally

### When to Use ShardDB
- High-concurrency workloads with many different buckets
- Applications with natural bucket-level partitioning
- When you need to scale write throughput
- Large datasets that benefit from distribution

### When NOT to Use ShardDB
- Single or few buckets (no benefit from sharding)
- When you need cross-bucket transactions
- Small datasets where overhead isn't justified

## Thread Safety
All operations are thread-safe:
- Factory operations use read-write locks
- Batch operations use mutex protection
- Cache operations use read-write locks per bucket
- WriteLock coordinates locks across multiple databases using sorted name order to prevent deadlocks

## Performance Tips
- Use `CachedDB` for high-frequency writes to reduce disk I/O
- Use `ShardDB` for high-concurrency workloads with many buckets
- Use batch operations when performing multiple write operations
- Batch operations on `CachedDB` automatically use cache layer
- Use `WriteLock` for atomic multi-database operations (e.g., order + balance update)
- Group operations by bucket for optimal performance
- Consider using bucket wrappers for repeated operations on the same bucket
- Call `Flush()` periodically or before critical operations to ensure persistence
- When using `ShardDB` with `WriteLock`, lock only the specific shards you need via `GetShardForBucket()`

## Dependencies
- `github.com/boltdb/bolt` - Core Bolt database functionality