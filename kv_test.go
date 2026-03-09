package bolt_db

import (
	"os"
	"sync"
	"testing"
)

func TestInMemKV_BasicOperations(t *testing.T) {
	kv := NewInmemMap[string, int]()

	// Test Set and Get
	kv.Set("key1", 100)
	val := kv.Get("key1")
	if val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}

	// Test non-existent key
	val = kv.Get("nonexistent")
	if val != 0 {
		t.Errorf("Expected 0 for non-existent key, got %d", val)
	}
}

func TestInMemKV_Delete(t *testing.T) {
	kv := NewInmemMap[string, string]()

	kv.Set("key1", "value1")
	kv.Delete("key1")

	val := kv.Get("key1")
	if val != "" {
		t.Errorf("Expected empty string after delete, got %s", val)
	}
}

func TestInMemKV_Exists(t *testing.T) {
	kv := NewInmemMap[string, bool]()

	if kv.Exists("key1") {
		t.Error("Key should not exist")
	}

	kv.Set("key1", true)
	if !kv.Exists("key1") {
		t.Error("Key should exist")
	}

	kv.Delete("key1")
	if kv.Exists("key1") {
		t.Error("Key should not exist after delete")
	}
}

func TestInMemKV_Len(t *testing.T) {
	kv := NewInmemMap[int, string]()

	if kv.Len() != 0 {
		t.Errorf("Expected length 0, got %d", kv.Len())
	}

	kv.Set(1, "one")
	kv.Set(2, "two")
	kv.Set(3, "three")

	if kv.Len() != 3 {
		t.Errorf("Expected length 3, got %d", kv.Len())
	}

	kv.Delete(2)

	if kv.Len() != 2 {
		t.Errorf("Expected length 2 after delete, got %d", kv.Len())
	}
}

func TestInMemKV_Clear(t *testing.T) {
	kv := NewInmemMap[string, int]()

	kv.Set("key1", 1)
	kv.Set("key2", 2)
	kv.Set("key3", 3)

	kv.Clear()

	if kv.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", kv.Len())
	}

	if kv.Exists("key1") {
		t.Error("Keys should not exist after clear")
	}
}

func TestInMemKV_ForEach(t *testing.T) {
	kv := NewInmemMap[string, int]()

	items := map[string]int{
		"key1": 100,
		"key2": 200,
		"key3": 300,
	}

	for k, v := range items {
		kv.Set(k, v)
	}

	count := 0
	sum := 0
	err := kv.ForEach(func(k string, v int) error {
		count++
		sum += v
		return nil
	})

	if err != nil {
		t.Fatalf("ForEach failed: %v", err)
	}

	if count != len(items) {
		t.Errorf("Expected %d iterations, got %d", len(items), count)
	}

	if sum != 600 {
		t.Errorf("Expected sum 600, got %d", sum)
	}
}

func TestInMemKV_ConcurrentAccess(t *testing.T) {
	kv := NewInmemMap[int, int]()

	var wg sync.WaitGroup
	
	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			kv.Set(val, val*2)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			_ = kv.Get(val)
		}(i)
	}

	wg.Wait()

	if kv.Len() != 100 {
		t.Errorf("Expected 100 items after concurrent writes, got %d", kv.Len())
	}
}

func TestInMemKV_NestedMap(t *testing.T) {
	// Test nested InMemKV structure like used in CachedDB
	outerKV := NewInmemMap[string, *InMemKV[string, int]]()

	innerKV1 := NewInmemMap[string, int]()
	innerKV1.Set("key1", 100)
	innerKV1.Set("key2", 200)

	outerKV.Set("bucket1", innerKV1)

	retrieved := outerKV.Get("bucket1")
	if retrieved == nil {
		t.Fatal("Expected nested map, got nil")
	}

	val := retrieved.Get("key1")
	if val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}
}

func TestInMemKV_ForEachWithError(t *testing.T) {
	kv := NewInmemMap[string, int]()

	kv.Set("key1", 1)
	kv.Set("key2", 2)
	kv.Set("key3", 3)

	testErr := os.ErrInvalid
	err := kv.ForEach(func(k string, v int) error {
		if v == 2 {
			return testErr
		}
		return nil
	})

	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}
}
