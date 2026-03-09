package bolt_db

import (
	"os"
	"testing"
)

func TestBatch_BasicOperations(t *testing.T) {
	dbPath := "/tmp/test_batch.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	if db == nil {
		t.Fatal("Failed to create database")
	}
	defer db.Close()

	batch := NewBatch(db)

	bucket := []byte("testBucket")
	value := []byte("value")

	// Add set operations
	for i := 0; i < 5; i++ {
		key := []byte{byte(i)}
		err := batch.Add(&WriteOperation{
			Bucket: bucket,
			Key:    key,
			Value:  &value,
			Op:     OpSet,
		})
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	// Execute batch
	err := batch.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify all items were written
	for i := 0; i < 5; i++ {
		key := []byte{byte(i)}
		retrieved, err := db.Get(bucket, key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if string(retrieved) != string(value) {
			t.Errorf("Expected %s, got %s", value, retrieved)
		}
	}
}

func TestBatch_DeleteOperations(t *testing.T) {
	dbPath := "/tmp/test_batch_delete.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	bucket := []byte("testBucket")
	key := []byte("testKey")
	value := []byte("testValue")

	// Set a value first
	err := db.Set(bucket, key, value)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Create batch with delete operation
	batch := NewBatch(db)
	err = batch.Add(&WriteOperation{
		Bucket: bucket,
		Key:    key,
		Value:  nil,
		Op:     OpDelete,
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Execute batch
	err = batch.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify deletion
	retrieved, err := db.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil after delete, got %v", retrieved)
	}
}

func TestBatch_EmptyBatch(t *testing.T) {
	dbPath := "/tmp/test_batch_empty.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	batch := NewBatch(db)
	
	// Execute empty batch should not error
	err := batch.Execute()
	if err != nil {
		t.Fatalf("Empty batch execute should not fail: %v", err)
	}
}

func TestBatch_NilValueInSetOperation(t *testing.T) {
	dbPath := "/tmp/test_batch_nil.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	batch := NewBatch(db)
	err := batch.Add(&WriteOperation{
		Bucket: []byte("bucket"),
		Key:    []byte("key"),
		Value:  nil,
		Op:     OpSet,
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Execute should fail with nil value for set
	err = batch.Execute()
	if err == nil {
		t.Error("Expected error for nil value in set operation")
	}
}

func TestBatch_MultipleBuckets(t *testing.T) {
	dbPath := "/tmp/test_batch_multi.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	batch := NewBatch(db)

	buckets := [][]byte{
		[]byte("bucket1"),
		[]byte("bucket2"),
		[]byte("bucket3"),
	}

	value := []byte("value")
	for _, bucket := range buckets {
		err := batch.Add(&WriteOperation{
			Bucket: bucket,
			Key:    []byte("key"),
			Value:  &value,
			Op:     OpSet,
		})
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	err := batch.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify all buckets have the data
	for _, bucket := range buckets {
		retrieved, err := db.Get(bucket, []byte("key"))
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if string(retrieved) != string(value) {
			t.Errorf("Expected %s, got %s", value, retrieved)
		}
	}
}

func TestBatch_SetDB(t *testing.T) {
	dbPath1 := "/tmp/test_batch_db1.db"
	dbPath2 := "/tmp/test_batch_db2.db"
	defer os.Remove(dbPath1)
	defer os.Remove(dbPath2)

	db1 := NewDB(dbPath1)
	db2 := NewDB(dbPath2)
	defer db1.Close()
	defer db2.Close()

	batch := NewBatch(db1)
	
	// Change target database
	batch.SetDB(db2)

	bucket := []byte("bucket")
	value := []byte("value")
	err := batch.Add(&WriteOperation{
		Bucket: bucket,
		Key:    []byte("key"),
		Value:  &value,
		Op:     OpSet,
	})
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	err = batch.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify data is in db2, not db1
	retrieved, err := db2.Get(bucket, []byte("key"))
	if err != nil {
		t.Fatalf("Get from db2 failed: %v", err)
	}
	if retrieved == nil {
		t.Error("Expected data in db2")
	}

	retrieved, err = db1.Get(bucket, []byte("key"))
	if err != nil {
		t.Fatalf("Get from db1 failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected no data in db1")
	}
}

// BUG TEST: batching.go:62 checks len(b.ops) instead of total operation count
func TestBatch_MaxOperationsCountBug(t *testing.T) {
	dbPath := "/tmp/test_batch_max.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	batch := NewBatch(db)

	// This test exposes the bug: we can add more than MAX_SEQUENTIAL_OPERATIONS
	// if we distribute them across buckets, because the check only looks at map length
	value := []byte("value")
	
	// Try to add MAX_SEQUENTIAL_OPERATIONS + 1 operations
	for i := 0; i <= MAX_SEQUENTIAL_OPERATIONS; i++ {
		err := batch.Add(&WriteOperation{
			Bucket: []byte("bucket"),
			Key:    []byte{byte(i)},
			Value:  &value,
			Op:     OpSet,
		})
		
		if i == MAX_SEQUENTIAL_OPERATIONS && err == nil {
			t.Error("Expected error when exceeding MAX_SEQUENTIAL_OPERATIONS")
		}
	}
}
