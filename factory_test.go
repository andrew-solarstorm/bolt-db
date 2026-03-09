package bolt_db

import (
	"os"
	"testing"
)

func TestFactory_Creation(t *testing.T) {
	dbPath := "/tmp/test_factory.db"
	defer os.Remove(dbPath)

	factory, err := NewBoltFactory("main", dbPath)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()

	if factory == nil {
		t.Fatal("Factory should not be nil")
	}
}

func TestFactory_GetDatabases(t *testing.T) {
	dbPath := "/tmp/test_factory_get.db"
	defer os.Remove(dbPath)

	factory, err := NewBoltFactory("main", dbPath)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()

	dbs, err := factory.GetDatabases()
	if err != nil {
		t.Fatalf("GetDatabases failed: %v", err)
	}

	if len(dbs) != 1 {
		t.Errorf("Expected 1 database, got %d", len(dbs))
	}

	if dbs[0] != "main" {
		t.Errorf("Expected database name 'main', got %s", dbs[0])
	}
}

func TestFactory_Open(t *testing.T) {
	dbPath1 := "/tmp/test_factory_open1.db"
	dbPath2 := "/tmp/test_factory_open2.db"
	defer os.Remove(dbPath1)
	defer os.Remove(dbPath2)

	factory, err := NewBoltFactory("db1", dbPath1)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()

	db2, err := factory.Open("db2", dbPath2)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if db2 == nil {
		t.Error("Opened database should not be nil")
	}

	dbs, _ := factory.GetDatabases()
	if len(dbs) != 2 {
		t.Errorf("Expected 2 databases, got %d", len(dbs))
	}
}

func TestFactory_Get(t *testing.T) {
	dbPath := "/tmp/test_factory_get_db.db"
	defer os.Remove(dbPath)

	factory, err := NewBoltFactory("main", dbPath)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()

	db, err := factory.Get("main")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if db == nil {
		t.Error("Retrieved database should not be nil")
	}

	// Test non-existent database
	_, err = factory.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent database")
	}
}

func TestFactory_Close(t *testing.T) {
	dbPath1 := "/tmp/test_factory_close1.db"
	dbPath2 := "/tmp/test_factory_close2.db"
	defer os.Remove(dbPath1)
	defer os.Remove(dbPath2)

	factory, err := NewBoltFactory("db1", dbPath1)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()

	factory.Open("db2", dbPath2)

	err = factory.Close("db2")
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	dbs, _ := factory.GetDatabases()
	if len(dbs) != 1 {
		t.Errorf("Expected 1 database after close, got %d", len(dbs))
	}

	// Try to close non-existent database
	err = factory.Close("nonexistent")
	if err == nil {
		t.Error("Expected error when closing non-existent database")
	}
}

func TestFactory_CloseAll(t *testing.T) {
	dbPath1 := "/tmp/test_factory_closeall1.db"
	dbPath2 := "/tmp/test_factory_closeall2.db"
	dbPath3 := "/tmp/test_factory_closeall3.db"
	defer os.Remove(dbPath1)
	defer os.Remove(dbPath2)
	defer os.Remove(dbPath3)

	factory, err := NewBoltFactory("db1", dbPath1)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	factory.Open("db2", dbPath2)
	factory.Open("db3", dbPath3)

	err = factory.CloseAll()
	if err != nil {
		t.Fatalf("CloseAll failed: %v", err)
	}

	dbs, _ := factory.GetDatabases()
	if len(dbs) != 0 {
		t.Errorf("Expected 0 databases after CloseAll, got %d", len(dbs))
	}
}

func TestFactory_ReplaceDatabase(t *testing.T) {
	dbPath1 := "/tmp/test_factory_replace1.db"
	dbPath2 := "/tmp/test_factory_replace2.db"
	defer os.Remove(dbPath1)
	defer os.Remove(dbPath2)

	factory, err := NewBoltFactory("main", dbPath1)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.CloseAll()

	// Open with same name but different path should replace
	db2, err := factory.Open("main", dbPath2)
	if err != nil {
		t.Fatalf("Replace failed: %v", err)
	}

	if db2 == nil {
		t.Error("Replaced database should not be nil")
	}

	dbs, _ := factory.GetDatabases()
	if len(dbs) != 1 {
		t.Errorf("Expected 1 database after replace, got %d", len(dbs))
	}
	
	// Verify the new database is the one at dbPath2
	if db2.Name() != dbPath2 {
		t.Errorf("Expected database path %s, got %s", dbPath2, db2.Name())
	}
}
