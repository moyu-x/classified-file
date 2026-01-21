package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/moyu-x/classified-file/internal"
)

func TestNewDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	if db.db == nil {
		t.Error("Expected database connection")
	}

	if db.cache == nil {
		t.Error("Expected cache map")
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Expected database file to be created")
	}
}

func TestDatabase_Exists(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	hash := "test_hash_1234567890"

	exists, err := db.Exists(hash)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if exists {
		t.Error("Expected hash to not exist initially")
	}

	record := &internal.FileRecord{
		Hash:      hash,
		FilePath:  "/test/file.txt",
		FileSize:  1024,
		CreatedAt: time.Now().Unix(),
	}

	if err := db.Insert(record); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	exists, err = db.Exists(hash)
	if err != nil {
		t.Fatalf("Exists() after insert error = %v", err)
	}

	if !exists {
		t.Error("Expected hash to exist after insert")
	}
}

func TestDatabase_Insert(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	record := &internal.FileRecord{
		Hash:      "hash1",
		FilePath:  "/test/file1.txt",
		FileSize:  1024,
		CreatedAt: time.Now().Unix(),
	}

	if err := db.Insert(record); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	exists, err := db.Exists("hash1")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if !exists {
		t.Error("Expected hash to exist after insert")
	}
}

func TestDatabase_Insert_Duplicate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	record := &internal.FileRecord{
		Hash:      "duplicate_hash",
		FilePath:  "/test/file.txt",
		FileSize:  1024,
		CreatedAt: time.Now().Unix(),
	}

	if err := db.Insert(record); err != nil {
		t.Fatalf("First Insert() error = %v", err)
	}

	record2 := &internal.FileRecord{
		Hash:      "duplicate_hash",
		FilePath:  "/test/file2.txt",
		FileSize:  2048,
		CreatedAt: time.Now().Unix(),
	}

	err = db.Insert(record2)
	if err == nil {
		t.Error("Expected error when inserting duplicate hash")
	}
}

func TestDatabase_MultipleRecords(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	const numRecords = 100

	for i := 0; i < numRecords; i++ {
		record := &internal.FileRecord{
			Hash:      fmt.Sprintf("hash%d", i),
			FilePath:  fmt.Sprintf("/test/file%d.txt", i),
			FileSize:  int64(1024 * (i + 1)),
			CreatedAt: time.Now().Unix(),
		}

		if err := db.Insert(record); err != nil {
			t.Fatalf("Insert() error = %v", err)
		}
	}

	for i := 0; i < numRecords; i++ {
		hash := fmt.Sprintf("hash%d", i)
		exists, err := db.Exists(hash)
		if err != nil {
			t.Fatalf("Exists() error for hash %s = %v", hash, err)
		}
		if !exists {
			t.Errorf("Expected hash %s to exist", hash)
		}
	}
}

func TestDatabase_Cache(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	hash := "cached_hash"

	exists, err := db.Exists(hash)
	if err != nil {
		t.Fatalf("Exists() first call error = %v", err)
	}
	if exists {
		t.Error("Expected hash to not exist initially")
	}

	record := &internal.FileRecord{
		Hash:      hash,
		FilePath:  "/test/file.txt",
		FileSize:  1024,
		CreatedAt: time.Now().Unix(),
	}

	if err := db.Insert(record); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	exists, err = db.Exists(hash)
	if err != nil {
		t.Fatalf("Exists() second call error = %v", err)
	}
	if !exists {
		t.Error("Expected hash to exist after insert")
	}
}

func TestDatabase_Close(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	_ = db.Close()
}

func TestDatabase_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db1, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("First NewDatabase() error = %v", err)
	}

	record := &internal.FileRecord{
		Hash:      "persistent_hash",
		FilePath:  "/test/file.txt",
		FileSize:  1024,
		CreatedAt: time.Now().Unix(),
	}

	if err := db1.Insert(record); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}

	if err := db1.Close(); err != nil {
		t.Fatalf("First Close() error = %v", err)
	}

	db2, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Second NewDatabase() error = %v", err)
	}
	defer db2.Close()

	exists, err := db2.Exists("persistent_hash")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if !exists {
		t.Error("Expected hash to persist across database reopen")
	}
}
