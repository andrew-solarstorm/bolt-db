package bolt_db

// Lockable represents a database that can be locked for multi-DB transactions.
type Lockable interface {
	Lock()
	Unlock()
}

// WriteLock provides a mechanism to execute multiple functions atomically
// across multiple databases by acquiring locks on all involved databases.
//
// The map-based structure prevents the same database from being added twice,
// which would cause deadlocks. Databases are identified by their Name() method.
//
// Usage pattern:
//  1. Create a new WriteLock
//  2. Add database and function pairs
//  3. Call Execute() to lock all databases and run all functions
//  4. All databases are automatically unlocked after execution
//
// Example:
//
//	wl := NewWriteLock()
//	wl.Add(orderDB, func() error {
//	    return orderDB.Set([]byte("orders"), []byte("order1"), orderData)
//	})
//	wl.Add(userDB, func() error {
//	    return userDB.Set([]byte("users"), []byte("user1"), userData)
//	})
//	err := wl.Execute() // Locks both DBs, executes both functions, then unlocks
type WriteLock struct {
	dbs map[string]IDatabase // Map of database names to prevent duplicates
	fns []func() error       // Functions to execute while databases are locked
}

// NewWriteLock creates a new WriteLock instance for coordinating multi-database operations.
//
// Returns:
//   - *WriteLock: A new WriteLock instance
func NewWriteLock() *WriteLock {
	return &WriteLock{
		dbs: make(map[string]IDatabase),
		fns: make([]func() error, 0),
	}
}

// Add registers a database and function pair to be executed atomically.
// The database is identified by its Name() method (typically the db path).
// If the same database is added multiple times, only one lock is acquired.
//
// Parameters:
//   - db: The database to lock during execution
//   - fn: The function to execute while the database is locked
func (wl *WriteLock) Add(db IDatabase, fn func() error) {
	wl.dbs[db.Name()] = db
	wl.fns = append(wl.fns, fn)
}

// Execute locks all registered databases, executes all functions in order,
// then unlocks all databases. Locks are acquired in a consistent order (sorted by name)
// to prevent deadlocks.
//
// If any function returns an error, execution stops immediately, all locks
// are released, and the error is returned.
//
// Returns:
//   - error: The first error encountered during execution, or nil if all succeed
func (wl *WriteLock) Execute() error {
	if len(wl.dbs) == 0 || len(wl.fns) == 0 {
		return nil
	}

	// Lock all databases in sorted name order
	for _, db := range wl.dbs {
		db.Lock()
	}

	// Ensure all databases are unlocked on return
	defer func() {
		for _, db := range wl.dbs {
			db.Unlock()
		}
	}()

	// Execute all functions in order
	for _, fn := range wl.fns {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}
