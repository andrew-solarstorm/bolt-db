package bolt_db

import (
	"os"
	"testing"
	"time"
)

func TestCachedDB_BasicOperations(t *testing.T) {
	dbPath := "/tmp/test_cached_db.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	if db == nil {
		t.Fatal("Failed to create database")
	}

	cached := NewCachedDB(db, 100, 1*time.Second, 5*time.Second)
	defer cached.Close()

	bucket := []byte("testBucket")
	key := []byte("testKey")
	value := []byte("testValue")

	// Test Set
	err := cached.Set(bucket, key, value)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get from cache
	retrieved, err := cached.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", value, retrieved)
	}

	// Test Delete
	err = cached.Delete(bucket, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	retrieved, err = cached.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil after delete, got %v", retrieved)
	}
}

func TestCachedDB_Exist(t *testing.T) {
	dbPath := "/tmp/test_cached_exist.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	cached := NewCachedDB(db, 100, 1*time.Second, 5*time.Second)
	defer cached.Close()

	bucket := []byte("testBucket")
	key := []byte("testKey")

	// Test non-existent key
	exists, err := cached.Exist(bucket, key)
	if err != nil {
		t.Fatalf("Exist failed: %v", err)
	}
	if exists {
		t.Error("Expected false for non-existent key")
	}

	// Add key
	err = cached.Set(bucket, key, []byte("value"))
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test existent key
	exists, err = cached.Exist(bucket, key)
	if err != nil {
		t.Fatalf("Exist failed: %v", err)
	}
	if !exists {
		t.Error("Expected true for existent key")
	}
}

func TestCachedDB_Flush(t *testing.T) {
	dbPath := "/tmp/test_cached_flush.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	cached := NewCachedDB(db, 1000, 1*time.Hour, 1*time.Hour)
	defer cached.Close()

	bucket := []byte("testBucket")
	key := []byte("testKey")
	value := []byte("testValue")

	// Set multiple times to trigger update counter
	for i := 0; i < 5; i++ {
		err := cached.Set(bucket, key, value)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	// Manually flush
	err := cached.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Verify data is persisted by reading directly from DB
	retrieved, err := db.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get from underlying DB failed: %v", err)
	}
	if retrieved == nil {
		t.Error("Expected data to be persisted after flush")
	}
	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", value, retrieved)
	}
}

func TestCachedDB_CopyProtection(t *testing.T) {
	dbPath := "/tmp/test_cached_copy.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	cached := NewCachedDB(db, 100, 1*time.Second, 5*time.Second)
	defer cached.Close()

	bucket := []byte("testBucket")
	key := []byte("testKey")
	value := []byte("original")

	err := cached.Set(bucket, key, value)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get and modify
	retrieved, err := cached.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Modify retrieved value
	retrieved[0] = 'X'

	// Get again and verify original is unchanged
	retrieved2, err := cached.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved2) != "original" {
		t.Errorf("Cache was mutated: expected 'original', got %s", retrieved2)
	}
}

func TestCachedDB_List(t *testing.T) {
	dbPath := "/tmp/test_cached_list.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	cached := NewCachedDB(db, 100, 1*time.Second, 5*time.Second)
	defer cached.Close()

	bucket := []byte("testBucket")
	
	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	for k, v := range items {
		err := cached.Set(bucket, []byte(k), []byte(v))
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	result, err := cached.List(bucket)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(result))
	}
}

func TestCachedDB_ForEach(t *testing.T) {
	dbPath := "/tmp/test_cached_foreach.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	cached := NewCachedDB(db, 100, 1*time.Second, 5*time.Second)
	defer cached.Close()

	bucket := []byte("testBucket")
	
	// Add some items
	for i := 0; i < 5; i++ {
		key := []byte{byte(i)}
		value := []byte{byte(i + 100)}
		err := cached.Set(bucket, key, value)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	count := 0
	err := cached.ForEach(bucket, func(key, value []byte) error {
		count++
		return nil
	})

	if err != nil {
		t.Fatalf("ForEach failed: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 iterations, got %d", count)
	}
}

func TestCachedDB_DefaultParameters(t *testing.T) {
	dbPath := "/tmp/test_cached_defaults.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	
	// Test with zero/negative values to trigger defaults
	cached := NewCachedDB(db, 0, 0, 0)
	defer cached.Close()

	if cached.updateThreshold != 300 {
		t.Errorf("Expected default updateThreshold 300, got %d", cached.updateThreshold)
	}
	if cached.updateInterval != 5*time.Minute {
		t.Errorf("Expected default updateInterval 5m, got %v", cached.updateInterval)
	}
	if cached.deleteInterval != 15*time.Minute {
		t.Errorf("Expected default deleteInterval 15m, got %v", cached.deleteInterval)
	}
}

func TestCachedDB_Name(t *testing.T) {
	dbPath := "/tmp/test_cached_name.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	cached := NewCachedDB(db, 100, 1*time.Second, 5*time.Second)
	defer cached.Close()

	name := cached.Name()
	expected := "cached:" + dbPath
	if name != expected {
		t.Errorf("Expected name %s, got %s", expected, name)
	}
}

func TestCachedDB_DropBucket(t *testing.T) {
	dbPath := "/tmp/test_cached_drop.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	cached := NewCachedDB(db, 100, 1*time.Second, 5*time.Second)
	defer cached.Close()

	bucket := []byte("testBucket")
	err := cached.Set(bucket, []byte("key"), []byte("value"))
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Flush to persist the bucket to disk
	err = cached.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	err = cached.DropBucket(string(bucket))
	if err != nil {
		t.Fatalf("DropBucket failed: %v", err)
	}

	// Verify bucket is dropped from cache
	value, err := cached.Get(bucket, []byte("key"))
	if err != nil {
		t.Fatalf("Get after drop failed: %v", err)
	}
	if value != nil {
		t.Errorf("Expected nil after bucket drop, got %v", value)
	}
}
