package deduplicator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/moyu-x/classified-file/database"
	"github.com/moyu-x/classified-file/hasher"
	"github.com/moyu-x/classified-file/internal"
)

func TestNewDeduplicator(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	d := NewDeduplicator(db, internal.ModeDelete, "", 0, false)

	if d == nil {
		t.Error("Expected deduplicator to be created")
	}

	if d.db != db {
		t.Error("Expected database to be set")
	}

	if d.mode != internal.ModeDelete {
		t.Error("Expected mode to be ModeDelete")
	}

	if d.targetDir != "" {
		t.Error("Expected targetDir to be empty")
	}

	if d.progressChan == nil {
		t.Error("Expected progressChan to be initialized")
	}
}

func TestNewDeduplicator_WithTargetDir(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	targetDir := filepath.Join(tempDir, "duplicates")
	d := NewDeduplicator(db, internal.ModeMove, targetDir, 0, false)

	if d.targetDir != targetDir {
		t.Errorf("Expected targetDir to be %s, got %s", targetDir, d.targetDir)
	}

	if d.mode != internal.ModeMove {
		t.Error("Expected mode to be ModeMove")
	}
}

func TestDeduplicator_moveFile(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	sourceFile := filepath.Join(sourceDir, "file.txt")
	content := []byte("test content")
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	db, err := database.NewDatabase(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	d := NewDeduplicator(db, internal.ModeMove, targetDir, 0, false)

	hash := "aabbccdd11223344"

	err = d.moveFile(sourceFile, hash)
	if err != nil {
		t.Fatalf("moveFile() error = %v", err)
	}

	if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
		t.Error("Expected source file to be moved (no longer exist)")
	}

	expectedDest := filepath.Join(targetDir, "aabbccdd_11223344.txt")
	if _, err := os.Stat(expectedDest); os.IsNotExist(err) {
		t.Error("Expected destination file to exist")
	}

	destContent, err := os.ReadFile(expectedDest)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(destContent) != string(content) {
		t.Error("Expected destination file content to match source")
	}
}

func TestDeduplicator_moveFile_NoTargetDir(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	db, err := database.NewDatabase(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	d := NewDeduplicator(db, internal.ModeMove, "", 0, false)

	hash := "aabbccdd11223344"

	err = d.moveFile(sourceFile, hash)
	if err == nil {
		t.Error("Expected error when target directory is not specified")
	}
}

func TestDeduplicator_moveFile_CreatesTargetDir(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(sourceFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	nestedTargetDir := filepath.Join(tempDir, "level1", "level2", "level3")

	db, err := database.NewDatabase(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	d := NewDeduplicator(db, internal.ModeMove, nestedTargetDir, 0, false)

	hash := "aabbccdd11223344"

	err = d.moveFile(sourceFile, hash)
	if err != nil {
		t.Fatalf("moveFile() error = %v", err)
	}

	if _, err := os.Stat(nestedTargetDir); os.IsNotExist(err) {
		t.Error("Expected nested target directory to be created")
	}
}

func TestDeduplicator_Progress(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	d := NewDeduplicator(db, internal.ModeDelete, "", 0, false)

	progressChan := d.Progress()
	if progressChan == nil {
		t.Error("Expected progress channel to be returned")
	}

	if cap(progressChan) != 100 {
		t.Errorf("Expected progress channel buffer size 100, got %d", cap(progressChan))
	}
}

func TestDeduplicator_moveFile_WithConflict(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	targetDir := filepath.Join(tempDir, "target")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	db, err := database.NewDatabase(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	d := NewDeduplicator(db, internal.ModeMove, targetDir, 0, false)

	hash := "aabbccdd11223344"

	srcFile1 := filepath.Join(sourceDir, "test.txt")
	err = os.WriteFile(srcFile1, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err = d.moveFile(srcFile1, hash)
	if err != nil {
		t.Errorf("First moveFile() error = %v", err)
	}

	srcFile2 := filepath.Join(sourceDir, "test2.txt")
	err = os.WriteFile(srcFile2, []byte("content2"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err = d.moveFile(srcFile2, hash)
	if err != nil {
		t.Errorf("Second moveFile() error = %v", err)
	}

	_, err = os.Stat(filepath.Join(targetDir, "aabbccdd_11223344.txt"))
	if err != nil {
		t.Errorf("First moved file not found: %v", err)
	}

	_, err = os.Stat(filepath.Join(targetDir, "aabbccdd_11223344_1.txt"))
	if err != nil {
		t.Errorf("Second moved file not found: %v", err)
	}

	_, err = os.Stat(srcFile1)
	if !os.IsNotExist(err) {
		t.Error("Expected first source file to be moved (no longer exist)")
	}

	_, err = os.Stat(srcFile2)
	if !os.IsNotExist(err) {
		t.Error("Expected second source file to be moved (no longer exist)")
	}
}

func TestDeduplicator_Integration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	content := []byte("test")
	file1 := filepath.Join(tempDir, "file1.txt")
	if err := os.WriteFile(file1, content, 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	hash, err := hasher.CalculateHash(file1)
	if err != nil {
		t.Fatalf("CalculateHash() error = %v", err)
	}

	hashStr := fmt.Sprintf("%016x", hash)

	record := &internal.FileRecord{
		Hash:      hashStr,
		FilePath:  file1,
		FileSize:  int64(len(content)),
		CreatedAt: time.Now().Unix(),
	}

	if err := db.Insert(record); err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	exists, err := db.Exists(hashStr)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if !exists {
		t.Error("Expected hash to exist in database")
	}
}

func TestDeduplicator_Integration_DuplicateDetection(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	content := []byte("duplicate content")

	file1 := filepath.Join(tempDir, "file1.txt")
	if err := os.WriteFile(file1, content, 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	file2 := filepath.Join(tempDir, "file2.txt")
	if err := os.WriteFile(file2, content, 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	hash, err := hasher.CalculateHash(file1)
	if err != nil {
		t.Fatalf("CalculateHash() error = %v", err)
	}

	hashStr := fmt.Sprintf("%016x", hash)

	record := &internal.FileRecord{
		Hash:      hashStr,
		FilePath:  file1,
		FileSize:  int64(len(content)),
		CreatedAt: time.Now().Unix(),
	}

	if err := db.Insert(record); err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	exists, err := db.Exists(hashStr)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if !exists {
		t.Error("Expected hash to exist in database")
	}

	hash2, err := hasher.CalculateHash(file2)
	if err != nil {
		t.Fatalf("CalculateHash() error = %v", err)
	}

	hashStr2 := fmt.Sprintf("%016x", hash2)

	if hashStr2 != hashStr {
		t.Errorf("Expected duplicate files to have same hash: %s vs %s", hashStr, hashStr2)
	}

	exists2, err := db.Exists(hashStr2)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if !exists2 {
		t.Error("Expected hash to exist in database")
	}
}

func TestDeduplicator_Process_DeleteMode(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	testFilesDir := filepath.Join(tempDir, "files")

	if err := os.MkdirAll(testFilesDir, 0755); err != nil {
		t.Fatalf("Failed to create test files directory: %v", err)
	}

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	duplicateContent := []byte("duplicate content")
	uniqueContent := []byte("unique content")

	file1 := filepath.Join(testFilesDir, "file1.txt")
	if err := os.WriteFile(file1, duplicateContent, 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	file2 := filepath.Join(testFilesDir, "file2.txt")
	if err := os.WriteFile(file2, duplicateContent, 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	file3 := filepath.Join(testFilesDir, "file3.txt")
	if err := os.WriteFile(file3, uniqueContent, 0644); err != nil {
		t.Fatalf("Failed to create file3: %v", err)
	}

	hash, err := hasher.CalculateHash(file1)
	if err != nil {
		t.Fatalf("CalculateHash() error = %v", err)
	}

	hashStr := fmt.Sprintf("%016x", hash)

	record := &internal.FileRecord{
		Hash:      hashStr,
		FilePath:  "/some/other/path/file1.txt",
		FileSize:  int64(len(duplicateContent)),
		CreatedAt: time.Now().Unix(),
	}

	if err := db.Insert(record); err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	d := NewDeduplicator(db, internal.ModeDelete, "", 3, false)

	stats, err := d.Process([]string{testFilesDir})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	if stats.Added != 1 {
		t.Errorf("Expected 1 file added, got %d", stats.Added)
	}

	if stats.Deleted != 2 {
		t.Errorf("Expected 2 files deleted, got %d", stats.Deleted)
	}

	if stats.TotalProcessed != 3 {
		t.Errorf("Expected 3 files processed, got %d", stats.TotalProcessed)
	}

	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Error("Expected duplicate file1 to be deleted")
	}

	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("Expected duplicate file2 to be deleted")
	}

	if _, err := os.Stat(file3); os.IsNotExist(err) {
		t.Error("Expected unique file3 to still exist")
	}

	if stats.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}

	if stats.EndTime.IsZero() {
		t.Error("Expected EndTime to be set")
	}
}

func TestDeduplicator_Process_MoveMode(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	testFilesDir := filepath.Join(tempDir, "files")
	targetDir := filepath.Join(tempDir, "moved")

	if err := os.MkdirAll(testFilesDir, 0755); err != nil {
		t.Fatalf("Failed to create test files directory: %v", err)
	}

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	duplicateContent := []byte("duplicate content")
	uniqueContent := []byte("unique content")

	file1 := filepath.Join(testFilesDir, "file1.txt")
	if err := os.WriteFile(file1, duplicateContent, 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	file2 := filepath.Join(testFilesDir, "file2.txt")
	if err := os.WriteFile(file2, duplicateContent, 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	file3 := filepath.Join(testFilesDir, "file3.txt")
	if err := os.WriteFile(file3, uniqueContent, 0644); err != nil {
		t.Fatalf("Failed to create file3: %v", err)
	}

	hash, err := hasher.CalculateHash(file1)
	if err != nil {
		t.Fatalf("CalculateHash() error = %v", err)
	}

	hashStr := fmt.Sprintf("%016x", hash)

	record := &internal.FileRecord{
		Hash:      hashStr,
		FilePath:  "/some/other/path/file1.txt",
		FileSize:  int64(len(duplicateContent)),
		CreatedAt: time.Now().Unix(),
	}

	if err := db.Insert(record); err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	d := NewDeduplicator(db, internal.ModeMove, targetDir, 3, false)

	stats, err := d.Process([]string{testFilesDir})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	if stats.Added != 1 {
		t.Errorf("Expected 1 file added, got %d", stats.Added)
	}

	if stats.Moved != 2 {
		t.Errorf("Expected 2 files moved, got %d", stats.Moved)
	}

	if stats.TotalProcessed != 3 {
		t.Errorf("Expected 3 files processed, got %d", stats.TotalProcessed)
	}

	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Error("Expected duplicate file1 to be moved")
	}

	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("Expected duplicate file2 to be moved")
	}

	if _, err := os.Stat(file3); os.IsNotExist(err) {
		t.Error("Expected unique file3 to still exist")
	}

	expectedDst1 := filepath.Join(targetDir, hashStr[:8]+"_"+hashStr[8:]+".txt")
	if _, err := os.Stat(expectedDst1); os.IsNotExist(err) {
		t.Errorf("Expected first moved file to exist at %s", expectedDst1)
	}

	expectedDst2 := filepath.Join(targetDir, hashStr[:8]+"_"+hashStr[8:]+"_1.txt")
	if _, err := os.Stat(expectedDst2); os.IsNotExist(err) {
		t.Errorf("Expected second moved file to exist at %s", expectedDst2)
	}

	if stats.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}

	if stats.EndTime.IsZero() {
		t.Error("Expected EndTime to be set")
	}
}

func TestDeduplicator_Process_WithExistingDatabaseRecords(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	testFilesDir := filepath.Join(tempDir, "files")
	targetDir := filepath.Join(tempDir, "moved")

	if err := os.MkdirAll(testFilesDir, 0755); err != nil {
		t.Fatalf("Failed to create test files directory: %v", err)
	}

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	duplicateContent := []byte("duplicate content")

	file1 := filepath.Join(testFilesDir, "file1.txt")
	if err := os.WriteFile(file1, duplicateContent, 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	file2 := filepath.Join(testFilesDir, "file2.txt")
	if err := os.WriteFile(file2, duplicateContent, 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	file3 := filepath.Join(testFilesDir, "file3.txt")
	if err := os.WriteFile(file3, duplicateContent, 0644); err != nil {
		t.Fatalf("Failed to create file3: %v", err)
	}

	hash, err := hasher.CalculateHash(file1)
	if err != nil {
		t.Fatalf("CalculateHash() error = %v", err)
	}

	hashStr := fmt.Sprintf("%016x", hash)

	record := &internal.FileRecord{
		Hash:      hashStr,
		FilePath:  "/some/other/path/file1.txt",
		FileSize:  int64(len(duplicateContent)),
		CreatedAt: time.Now().Unix(),
	}

	if err := db.Insert(record); err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	d := NewDeduplicator(db, internal.ModeMove, targetDir, 3, false)

	stats, err := d.Process([]string{testFilesDir})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	if stats.Added != 0 {
		t.Errorf("Expected 0 file added, got %d", stats.Added)
	}

	if stats.Moved != 3 {
		t.Errorf("Expected 3 files moved, got %d", stats.Moved)
	}

	if stats.TotalProcessed != 3 {
		t.Errorf("Expected 3 files processed, got %d", stats.TotalProcessed)
	}

	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Error("Expected file1 to be moved")
	}

	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("Expected file2 to be moved")
	}

	if _, err := os.Stat(file3); !os.IsNotExist(err) {
		t.Error("Expected file3 to be moved")
	}

	if stats.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}

	if stats.EndTime.IsZero() {
		t.Error("Expected EndTime to be set")
	}
}

func TestDeduplicator_Process_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	emptyDir := filepath.Join(tempDir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	d := NewDeduplicator(db, internal.ModeDelete, "", 0, false)

	stats, err := d.Process([]string{emptyDir})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	if stats.Added != 0 {
		t.Errorf("Expected 0 files added, got %d", stats.Added)
	}

	if stats.Deleted != 0 {
		t.Errorf("Expected 0 files deleted, got %d", stats.Deleted)
	}

	if stats.Moved != 0 {
		t.Errorf("Expected 0 files moved, got %d", stats.Moved)
	}

	if stats.TotalProcessed != 0 {
		t.Errorf("Expected 0 files processed, got %d", stats.TotalProcessed)
	}
}

func TestDeduplicator_Process_Statistics(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	testFilesDir := filepath.Join(tempDir, "files")

	if err := os.MkdirAll(testFilesDir, 0755); err != nil {
		t.Fatalf("Failed to create test files directory: %v", err)
	}

	db, err := database.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	contents := [][]byte{
		[]byte("unique content 1"),
		[]byte("duplicate content"),
		[]byte("duplicate content"),
		[]byte("unique content 2"),
		[]byte("another duplicate"),
		[]byte("another duplicate"),
		[]byte("another duplicate"),
	}

	files := []string{}
	for i, content := range contents {
		file := filepath.Join(testFilesDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(file, content, 0644); err != nil {
			t.Fatalf("Failed to create file%d: %v", i, err)
		}
		files = append(files, file)
	}

	hash1, err := hasher.CalculateHash(files[0])
	if err != nil {
		t.Fatalf("CalculateHash() error for file0: %v", err)
	}
	hashStr1 := fmt.Sprintf("%016x", hash1)

	hash2, err := hasher.CalculateHash(files[1])
	if err != nil {
		t.Fatalf("CalculateHash() error for file1: %v", err)
	}
	hashStr2 := fmt.Sprintf("%016x", hash2)

	hash5, err := hasher.CalculateHash(files[4])
	if err != nil {
		t.Fatalf("CalculateHash() error for file4: %v", err)
	}
	hashStr5 := fmt.Sprintf("%016x", hash5)

	record1 := &internal.FileRecord{
		Hash:      hashStr1,
		FilePath:  "/some/other/path/file0.txt",
		FileSize:  int64(len(contents[0])),
		CreatedAt: time.Now().Unix(),
	}
	if err := db.Insert(record1); err != nil {
		t.Fatalf("Failed to insert record1: %v", err)
	}

	record2 := &internal.FileRecord{
		Hash:      hashStr2,
		FilePath:  "/some/other/path/file1.txt",
		FileSize:  int64(len(contents[1])),
		CreatedAt: time.Now().Unix(),
	}
	if err := db.Insert(record2); err != nil {
		t.Fatalf("Failed to insert record2: %v", err)
	}

	record3 := &internal.FileRecord{
		Hash:      hashStr5,
		FilePath:  "/some/other/path/file4.txt",
		FileSize:  int64(len(contents[4])),
		CreatedAt: time.Now().Unix(),
	}
	if err := db.Insert(record3); err != nil {
		t.Fatalf("Failed to insert record3: %v", err)
	}

	d := NewDeduplicator(db, internal.ModeDelete, "", len(files), false)

	stats, err := d.Process([]string{testFilesDir})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	if stats.Added != 1 {
		t.Errorf("Expected 1 file added, got %d", stats.Added)
	}

	if stats.Deleted != 6 {
		t.Errorf("Expected 6 files deleted, got %d", stats.Deleted)
	}

	if stats.TotalProcessed != 7 {
		t.Errorf("Expected 7 files processed, got %d", stats.TotalProcessed)
	}

	expectedFreedSpace := int64(len(contents[0]) + len(contents[1]) + len(contents[2]) + len(contents[4]) + len(contents[5]) + len(contents[6]))
	if stats.FreedSpace != expectedFreedSpace {
		t.Errorf("Expected freed space %d, got %d", expectedFreedSpace, stats.FreedSpace)
	}

	if stats.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}

	if stats.EndTime.IsZero() {
		t.Error("Expected EndTime to be set")
	}
}
