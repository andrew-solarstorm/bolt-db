package bolt_db

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestWriteLock_BasicOperation(t *testing.T) {
	dbPath1 := "/tmp/test_lock1.db"
	dbPath2 := "/tmp/test_lock2.db"
	defer os.Remove(dbPath1)
	defer os.Remove(dbPath2)

	db1 := NewDB(dbPath1)
	db2 := NewDB(dbPath2)
	defer db1.Close()
	defer db2.Close()

	wl := NewWriteLock()
	
	executed1 := false
	executed2 := false

	wl.Add(db1, func() error {
		executed1 = true
		return nil
	})

	wl.Add(db2, func() error {
		executed2 = true
		return nil
	})

	err := wl.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !executed1 || !executed2 {
		t.Error("Not all functions were executed")
	}
}

func TestWriteLock_ErrorHandling(t *testing.T) {
	dbPath := "/tmp/test_lock_error.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	wl := NewWriteLock()
	
	executed := false
	wl.Add(db, func() error {
		return os.ErrInvalid
	})

	wl.Add(db, func() error {
		executed = true
		return nil
	})

	err := wl.Execute()
	if err != os.ErrInvalid {
		t.Errorf("Expected ErrInvalid, got %v", err)
	}

	if executed {
		t.Error("Second function should not have executed after error")
	}
}

func TestWriteLock_DuplicateDB(t *testing.T) {
	dbPath := "/tmp/test_lock_dup.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	wl := NewWriteLock()
	
	// Add same DB multiple times
	wl.Add(db, func() error { return nil })
	wl.Add(db, func() error { return nil })
	wl.Add(db, func() error { return nil })

	// Should only lock once (verified by not deadlocking)
	err := wl.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify only one entry in dbs map
	if len(wl.dbs) != 1 {
		t.Errorf("Expected 1 unique database, got %d", len(wl.dbs))
	}

	// Verify all functions are tracked
	if len(wl.fns) != 3 {
		t.Errorf("Expected 3 functions, got %d", len(wl.fns))
	}
}

func TestWriteLock_EmptyLock(t *testing.T) {
	wl := NewWriteLock()
	
	// Execute with no databases or functions
	err := wl.Execute()
	if err != nil {
		t.Fatalf("Empty execute should not fail: %v", err)
	}
}

func TestWriteLock_ConcurrentAccess(t *testing.T) {
	dbPath := "/tmp/test_lock_concurrent.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	bucket := []byte("counter")
	key := []byte("count")
	
	// Initialize counter
	db.Set(bucket, key, []byte{0})

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	// Try concurrent writes without WriteLock - should work but may have race
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(val byte) {
			defer wg.Done()
			
			wl := NewWriteLock()
			wl.Add(db, func() error {
				// Simulate some work
				time.Sleep(1 * time.Millisecond)
				return db.Set(bucket, []byte{val}, []byte{val})
			})
			
			if err := wl.Execute(); err != nil {
				errors <- err
			}
		}(byte(i))
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent write error: %v", err)
	}
}

func TestWriteLock_OrderOfExecution(t *testing.T) {
	dbPath := "/tmp/test_lock_order.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	wl := NewWriteLock()
	
	order := []int{}
	
	wl.Add(db, func() error {
		order = append(order, 1)
		return nil
	})

	wl.Add(db, func() error {
		order = append(order, 2)
		return nil
	})

	wl.Add(db, func() error {
		order = append(order, 3)
		return nil
	})

	err := wl.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify functions execute in order
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("Functions executed out of order: %v", order)
	}
}

func TestWriteLock_AtomicityAcrossDBs(t *testing.T) {
	dbPath1 := "/tmp/test_lock_atomic1.db"
	dbPath2 := "/tmp/test_lock_atomic2.db"
	defer os.Remove(dbPath1)
	defer os.Remove(dbPath2)

	db1 := NewDB(dbPath1)
	db2 := NewDB(dbPath2)
	defer db1.Close()
	defer db2.Close()

	bucket := []byte("test")
	key := []byte("key")

	wl := NewWriteLock()
	
	wl.Add(db1, func() error {
		return db1.Set(bucket, key, []byte("value1"))
	})

	wl.Add(db2, func() error {
		return db2.Set(bucket, key, []byte("value2"))
	})

	err := wl.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify both writes succeeded
	val1, err := db1.Get(bucket, key)
	if err != nil || string(val1) != "value1" {
		t.Errorf("DB1 write failed")
	}

	val2, err := db2.Get(bucket, key)
	if err != nil || string(val2) != "value2" {
		t.Errorf("DB2 write failed")
	}
}
