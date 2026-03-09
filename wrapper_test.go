package bolt_db

import (
	"os"
	"testing"
)

func TestDBWrapper_BasicOperations(t *testing.T) {
	dbPath := "/tmp/test_wrapper.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	bucket := []byte("testBucket")
	wrapper := NewDBWrapper(db, bucket)

	key := []byte("testKey")
	value := []byte("testValue")

	// Test Set
	err := wrapper.Set(key, value)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	retrieved, err := wrapper.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", value, retrieved)
	}

	// Test Delete
	err = wrapper.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	retrieved, err = wrapper.Get(key)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil after delete, got %v", retrieved)
	}
}

func TestDBWrapper_List(t *testing.T) {
	dbPath := "/tmp/test_wrapper_list.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	bucket := []byte("testBucket")
	wrapper := NewDBWrapper(db, bucket)

	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range items {
		err := wrapper.Set([]byte(k), []byte(v))
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	result, err := wrapper.List()
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

func TestDBWrapper_ForEach(t *testing.T) {
	dbPath := "/tmp/test_wrapper_foreach.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	bucket := []byte("testBucket")
	wrapper := NewDBWrapper(db, bucket)

	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	for k, v := range items {
		err := wrapper.Set([]byte(k), []byte(v))
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	count := 0
	err := wrapper.ForEach(func(key, value []byte) error {
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

func TestDBWrapper_NewBatch(t *testing.T) {
	dbPath := "/tmp/test_wrapper_batch.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	bucket := []byte("testBucket")
	wrapper := NewDBWrapper(db, bucket)

	batch := wrapper.NewBatch()
	if batch == nil {
		t.Fatal("NewBatch should not return nil")
	}

	// Verify batch is connected to the underlying database
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

	// Verify through wrapper
	retrieved, err := wrapper.Get([]byte("key"))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", value, retrieved)
	}
}

func TestDBWrapper_MultipleBuckets(t *testing.T) {
	dbPath := "/tmp/test_wrapper_multi.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	wrapper1 := NewDBWrapper(db, []byte("bucket1"))
	wrapper2 := NewDBWrapper(db, []byte("bucket2"))

	// Write to both wrappers
	err := wrapper1.Set([]byte("key"), []byte("value1"))
	if err != nil {
		t.Fatalf("Wrapper1 Set failed: %v", err)
	}

	err = wrapper2.Set([]byte("key"), []byte("value2"))
	if err != nil {
		t.Fatalf("Wrapper2 Set failed: %v", err)
	}

	// Verify isolation
	val1, _ := wrapper1.Get([]byte("key"))
	val2, _ := wrapper2.Get([]byte("key"))

	if string(val1) != "value1" {
		t.Errorf("Wrapper1 expected value1, got %s", val1)
	}

	if string(val2) != "value2" {
		t.Errorf("Wrapper2 expected value2, got %s", val2)
	}
}
