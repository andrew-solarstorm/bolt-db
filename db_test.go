package bolt_db

import (
	"os"
	"testing"
)

func TestDB_BasicOperations(t *testing.T) {
	dbPath := "/tmp/test_db.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	if db == nil {
		t.Fatal("Failed to create database")
	}
	defer db.Close()

	bucket := []byte("testBucket")
	key := []byte("testKey")
	value := []byte("testValue")

	// Test Set
	err := db.Set(bucket, key, value)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	retrieved, err := db.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", value, retrieved)
	}

	// Test Delete
	err = db.Delete(bucket, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	retrieved, err = db.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil after delete, got %v", retrieved)
	}
}

func TestDB_GetNonExistentBucket(t *testing.T) {
	dbPath := "/tmp/test_db_nonexistent.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	if db == nil {
		t.Fatal("Failed to create database")
	}
	defer db.Close()

	value, err := db.Get([]byte("nonexistent"), []byte("key"))
	if err != nil {
		t.Fatalf("Get on nonexistent bucket should not error: %v", err)
	}
	if value != nil {
		t.Errorf("Expected nil for nonexistent bucket, got %v", value)
	}
}



func TestDB_List(t *testing.T) {
	dbPath := "/tmp/test_db_list.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	if db == nil {
		t.Fatal("Failed to create database")
	}
	defer db.Close()

	bucket := []byte("testBucket")

	// Add multiple items
	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range items {
		err := db.Set(bucket, []byte(k), []byte(v))
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// List all items
	result, err := db.List(bucket)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(result))
	}

	for k, v := range items {
		if string(result[k]) != v {
			t.Errorf("Expected %s for key %s, got %s", v, k, result[k])
		}
	}
}

func TestDB_ForEach(t *testing.T) {
	dbPath := "/tmp/test_db_foreach.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	if db == nil {
		t.Fatal("Failed to create database")
	}
	defer db.Close()

	bucket := []byte("testBucket")

	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	for k, v := range items {
		err := db.Set(bucket, []byte(k), []byte(v))
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	count := 0
	err := db.ForEach(bucket, func(key, value []byte) error {
		count++
		return nil
	})

	if err != nil {
		t.Fatalf("ForEach failed: %v", err)
	}

	if count != len(items) {
		t.Errorf("Expected %d iterations, got %d", len(items), count)
	}
}

func TestDB_Buckets(t *testing.T) {
	dbPath := "/tmp/test_db_buckets.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	if db == nil {
		t.Fatal("Failed to create database")
	}
	defer db.Close()

	buckets := [][]byte{
		[]byte("bucket1"),
		[]byte("bucket2"),
		[]byte("bucket3"),
	}

	for _, bucket := range buckets {
		err := db.Set(bucket, []byte("key"), []byte("value"))
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	result := db.Buckets()
	if len(result) != len(buckets) {
		t.Errorf("Expected %d buckets, got %d", len(buckets), len(result))
	}
}

func TestDB_DropBucket(t *testing.T) {
	dbPath := "/tmp/test_db_drop.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	if db == nil {
		t.Fatal("Failed to create database")
	}
	defer db.Close()

	bucket := []byte("testBucket")
	err := db.Set(bucket, []byte("key"), []byte("value"))
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err = db.DropBucket(string(bucket))
	if err != nil {
		t.Fatalf("DropBucket failed: %v", err)
	}

	// Verify bucket is dropped
	value, err := db.Get(bucket, []byte("key"))
	if err != nil {
		t.Fatalf("Get after drop failed: %v", err)
	}
	if value != nil {
		t.Errorf("Expected nil after bucket drop, got %v", value)
	}
}

func TestDB_Name(t *testing.T) {
	dbPath := "/tmp/test_db_name.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	if db == nil {
		t.Fatal("Failed to create database")
	}
	defer db.Close()

	if db.Name() != dbPath {
		t.Errorf("Expected name %s, got %s", dbPath, db.Name())
	}
}

func TestDB_LockUnlock(t *testing.T) {
	dbPath := "/tmp/test_db_lock.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	if db == nil {
		t.Fatal("Failed to create database")
	}
	defer db.Close()

	// Test that Lock/Unlock don't panic
	db.Lock()
	db.Unlock()
}
