package bolt_db

import (
	"fmt"
	"os"
	"testing"
)

func TestShardDB_Creation(t *testing.T) {
	basePath := "/tmp/test_shard"
	factory, err := NewBoltFactory("main", basePath+"_main.db")
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()
	defer cleanupShards(basePath, 3)

	shardDB, err := NewShardDB(factory, basePath, 3, false)
	if err != nil {
		t.Fatalf("Failed to create ShardDB: %v", err)
	}
	defer shardDB.Close()

	if shardDB.shards != 3 {
		t.Errorf("Expected 3 shards, got %d", shardDB.shards)
	}
}

func TestShardDB_BasicOperations(t *testing.T) {
	basePath := "/tmp/test_shard_basic"
	factory, err := NewBoltFactory("main", basePath+"_main.db")
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()
	defer cleanupShards(basePath, 2)

	shardDB, err := NewShardDB(factory, basePath, 2, false)
	if err != nil {
		t.Fatalf("Failed to create ShardDB: %v", err)
	}
	defer shardDB.Close()

	bucket := []byte("testBucket")
	key := []byte("testKey")
	value := []byte("testValue")

	// Test Set
	err = shardDB.Set(bucket, key, value)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	retrieved, err := shardDB.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", value, retrieved)
	}

	// Test Delete
	err = shardDB.Delete(bucket, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	retrieved, err = shardDB.Get(bucket, key)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if retrieved != nil {
		t.Errorf("Expected nil after delete, got %v", retrieved)
	}
}

func TestShardDB_ShardDistribution(t *testing.T) {
	basePath := "/tmp/test_shard_dist"
	factory, err := NewBoltFactory("main", basePath+"_main.db")
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()
	defer cleanupShards(basePath, 3)

	shardDB, err := NewShardDB(factory, basePath, 3, false)
	if err != nil {
		t.Fatalf("Failed to create ShardDB: %v", err)
	}
	defer shardDB.Close()

	// Test that same bucket always goes to same shard
	bucket := []byte("consistentBucket")
	shard1 := shardDB.getShard(bucket)
	shard2 := shardDB.getShard(bucket)

	if shard1 != shard2 {
		t.Errorf("Same bucket should map to same shard: got %d and %d", shard1, shard2)
	}

	// Test empty bucket
	emptyShard := shardDB.getShard([]byte(""))
	if emptyShard != 0 {
		t.Errorf("Empty bucket should map to shard 0, got %d", emptyShard)
	}

	// Test single byte bucket
	singleShard := shardDB.getShard([]byte("a"))
	if singleShard < 0 || singleShard >= 3 {
		t.Errorf("Single byte bucket gave invalid shard: %d", singleShard)
	}
}

func TestShardDB_List(t *testing.T) {
	basePath := "/tmp/test_shard_list"
	factory, err := NewBoltFactory("main", basePath+"_main.db")
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()
	defer cleanupShards(basePath, 2)

	shardDB, err := NewShardDB(factory, basePath, 2, false)
	if err != nil {
		t.Fatalf("Failed to create ShardDB: %v", err)
	}
	defer shardDB.Close()

	bucket := []byte("testBucket")

	items := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for k, v := range items {
		err := shardDB.Set(bucket, []byte(k), []byte(v))
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	result, err := shardDB.List(bucket)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(result))
	}
}

func TestShardDB_Buckets(t *testing.T) {
	basePath := "/tmp/test_shard_buckets"
	factory, err := NewBoltFactory("main", basePath+"_main.db")
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()
	defer cleanupShards(basePath, 2)

	shardDB, err := NewShardDB(factory, basePath, 2, false)
	if err != nil {
		t.Fatalf("Failed to create ShardDB: %v", err)
	}
	defer shardDB.Close()

	// Create buckets that will distribute across shards
	buckets := [][]byte{
		[]byte("bucket1"),
		[]byte("bucket2"),
		[]byte("bucket3"),
	}

	for _, bucket := range buckets {
		err := shardDB.Set(bucket, []byte("key"), []byte("value"))
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
	}

	result := shardDB.Buckets()
	if len(result) != len(buckets) {
		t.Errorf("Expected %d buckets, got %d", len(buckets), len(result))
	}
}

func TestShardDB_DropBucket(t *testing.T) {
	basePath := "/tmp/test_shard_drop"
	factory, err := NewBoltFactory("main", basePath+"_main.db")
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()
	defer cleanupShards(basePath, 2)

	shardDB, err := NewShardDB(factory, basePath, 2, false)
	if err != nil {
		t.Fatalf("Failed to create ShardDB: %v", err)
	}
	defer shardDB.Close()

	bucket := []byte("testBucket")
	err = shardDB.Set(bucket, []byte("key"), []byte("value"))
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	err = shardDB.DropBucket(string(bucket))
	if err != nil {
		t.Fatalf("DropBucket failed: %v", err)
	}

	value, err := shardDB.Get(bucket, []byte("key"))
	if err != nil {
		t.Fatalf("Get after drop failed: %v", err)
	}
	if value != nil {
		t.Errorf("Expected nil after bucket drop, got %v", value)
	}
}

func TestShardDB_InvalidShardCount(t *testing.T) {
	basePath := "/tmp/test_shard_invalid"
	factory, err := NewBoltFactory("main", basePath+"_main.db")
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()

	_, err = NewShardDB(factory, basePath, 0, false)
	if err == nil {
		t.Error("Expected error for invalid shard count")
	}

	_, err = NewShardDB(factory, basePath, -1, false)
	if err == nil {
		t.Error("Expected error for negative shard count")
	}
}

// Helper function to cleanup shard files
func cleanupShards(basePath string, count int) {
	for i := 0; i < count; i++ {
		os.Remove(fmt.Sprintf("%s_shard%d.db", basePath, i))
	}
	os.Remove(basePath + "_main.db")
}
