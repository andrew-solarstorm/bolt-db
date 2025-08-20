package boltfactory

import (
	"fmt"
	"sync"
)

// BoltFactory manages multiple Bolt database instances with thread-safe operations.
// It provides a centralized way to create, access, and manage multiple databases
// with different names and file paths. All operations are protected by read-write locks
// to ensure thread safety in concurrent environments.
type BoltFactory struct {
	lck       sync.RWMutex             // Read-write lock for thread-safe operations
	databases map[string]*BoltDatabase // Map of database names to database instances
}

// NewBoltFactory creates a new factory instance with an initial database.
// The factory will manage the lifecycle of all databases created through it.
//
// Parameters:
//   - name: The name identifier for the initial database
//   - defaultPath: The file path for the initial database
//
// Returns:
//   - *BoltFactory: A new factory instance
//   - error: An error if the initial database cannot be created
func NewBoltFactory(name, defaultPath string) (*BoltFactory, error) {
	databases := make(map[string]*BoltDatabase)
	databases[name] = NewBoltDatabase(defaultPath)

	if err := databases[name]; err != nil {
		return nil, fmt.Errorf("could not open database %s: %v", name, err)
	}
	return &BoltFactory{databases: databases}, nil
}

// GetDatabases returns a list of all database names currently managed by the factory.
// This operation is thread-safe and uses a read lock.
//
// Returns:
//   - []string: A slice of database names
//   - error: Any error that occurred during the operation
func (f *BoltFactory) GetDatabases() ([]string, error) {
	f.lck.RLock()
	defer f.lck.RUnlock()

	databases := make([]string, 0, len(f.databases))
	for name := range f.databases {
		databases = append(databases, name)
	}
	return databases, nil
}

// Open creates a new database instance and adds it to the factory's management.
// If a database with the same name already exists, it will be replaced.
// This operation is thread-safe and uses a write lock.
//
// Parameters:
//   - name: The name identifier for the database
//   - path: The file path for the database
//
// Returns:
//   - *BoltDatabase: The newly created database instance
//   - error: Any error that occurred during creation
func (f *BoltFactory) Open(name, path string) (*BoltDatabase, error) {
	f.lck.Lock()
	defer f.lck.Unlock()
	f.databases[name] = NewBoltDatabase(path)
	return f.databases[name], nil
}

// Close closes a specific database and removes it from the factory's management.
// This operation is thread-safe and uses a write lock.
//
// Parameters:
//   - name: The name of the database to close
//
// Returns:
//   - error: An error if the database doesn't exist or closing fails
func (f *BoltFactory) Close(name string) error {
	f.lck.Lock()
	defer f.lck.Unlock()

	db, ok := f.databases[name]
	if !ok {
		return fmt.Errorf("database %s not found", name)
	}

	if err := db.Close(); err != nil {
		return err
	}

	delete(f.databases, name)
	return nil
}

// CloseAll closes all databases managed by the factory and clears the internal map.
// This operation is thread-safe and uses a write lock.
//
// Returns:
//   - error: Any error that occurred during the closing process
func (f *BoltFactory) CloseAll() error {
	f.lck.Lock()
	defer f.lck.Unlock()

	for name := range f.databases {
		if err := f.Close(name); err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves a database instance by name.
// This operation is thread-safe and uses a read lock.
//
// Parameters:
//   - name: The name of the database to retrieve
//
// Returns:
//   - *BoltDatabase: The database instance, or nil if not found
//   - error: An error if the database doesn't exist
func (f *BoltFactory) Get(name string) (*BoltDatabase, error) {
	f.lck.RLock()
	defer f.lck.RUnlock()

	db, ok := f.databases[name]
	if !ok {
		return nil, fmt.Errorf("database %s not found", name)
	}

	return db, nil
}
