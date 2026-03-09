package bolt_db

import (
	"os"
	"testing"
)

type TestStruct struct {
	Name  string
	Value int
}

func TestGenericDB_BasicOperations(t *testing.T) {
	dbPath := "/tmp/test_generic.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	gdb := NewGenericDB[TestStruct](db, nil, nil)

	bucket := []byte("testBucket")
	key := []byte("testKey")
	data := &TestStruct{Name: "test", Value: 42}

	// Test Set
	err := gdb.Set(bucket, key, data)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	retrieved, err := gdb.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Name != data.Name || retrieved.Value != data.Value {
		t.Errorf("Expected %+v, got %+v", data, retrieved)
	}

	// Test Delete
	err = gdb.Delete(bucket, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestGenericDB_ForEach(t *testing.T) {
	dbPath := "/tmp/test_generic_foreach.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	gdb := NewGenericDB[TestStruct](db, nil, nil)

	bucket := []byte("testBucket")
	
	items := []*TestStruct{
		{Name: "item1", Value: 1},
		{Name: "item2", Value: 2},
		{Name: "item3", Value: 3},
	}

	for i, item := range items {
		key := []byte{byte(i)}
		err := gdb.Set(bucket, key, item)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	count := 0
	err := gdb.ForEach(bucket, func(key []byte, value *TestStruct) error {
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

// BUG TEST: generic_db.go:79 returns nil instead of err
func TestGenericDB_SetEncodingErrorBug(t *testing.T) {
	dbPath := "/tmp/test_generic_bug.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	// Create a GenericDB with a failing encoder
	failingEncoder := func(in any) ([]byte, error) {
		return nil, os.ErrInvalid
	}

	gdb := NewGenericDB[TestStruct](db, nil, failingEncoder)

	bucket := []byte("testBucket")
	key := []byte("testKey")
	data := &TestStruct{Name: "test", Value: 42}

	// BUG: This should return the encoding error, but returns nil
	err := gdb.Set(bucket, key, data)
	if err == nil {
		t.Error("Expected encoding error, but got nil (BUG: line 79 returns nil instead of err)")
	}
}

func TestGenericDB_DefaultEncoderDecoder(t *testing.T) {
	dbPath := "/tmp/test_generic_default.db"
	defer os.Remove(dbPath)

	db := NewDB(dbPath)
	defer db.Close()

	// Test with nil encoder/decoder to use defaults
	gdb := NewGenericDB[TestStruct](db, nil, nil)

	bucket := []byte("testBucket")
	key := []byte("testKey")
	data := &TestStruct{Name: "test", Value: 99}

	err := gdb.Set(bucket, key, data)
	if err != nil {
		t.Fatalf("Set with default encoder failed: %v", err)
	}

	retrieved, err := gdb.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get with default decoder failed: %v", err)
	}

	if retrieved.Name != data.Name || retrieved.Value != data.Value {
		t.Errorf("Expected %+v, got %+v", data, retrieved)
	}
}
