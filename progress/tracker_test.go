package progress

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTracker(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	tracker, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	if tracker == nil {
		t.Fatal("NewTracker() returned nil")
	}
	if tracker.GetProcessedCount() != 0 {
		t.Errorf("Expected 0 processed files, got %d", tracker.GetProcessedCount())
	}
	if err := tracker.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestMarkProcessed(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	tracker, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	filePath1 := "/path/to/file1.txt"
	filePath2 := "/path/to/file2.txt"

	err = tracker.MarkProcessed(filePath1)
	if err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}
	if !tracker.IsProcessed(filePath1) {
		t.Error("IsProcessed() should return true for marked file")
	}
	if tracker.GetProcessedCount() != 1 {
		t.Errorf("Expected 1 processed file, got %d", tracker.GetProcessedCount())
	}

	err = tracker.MarkProcessed(filePath2)
	if err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}
	if !tracker.IsProcessed(filePath2) {
		t.Error("IsProcessed() should return true for marked file")
	}
	if tracker.GetProcessedCount() != 2 {
		t.Errorf("Expected 2 processed files, got %d", tracker.GetProcessedCount())
	}

	err = tracker.MarkProcessed(filePath1)
	if err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}
	if tracker.GetProcessedCount() != 2 {
		t.Errorf("Expected 2 processed files (duplicate), got %d", tracker.GetProcessedCount())
	}
}

func TestIsProcessed(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	tracker, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	filePath := "/path/to/file.txt"
	if tracker.IsProcessed(filePath) {
		t.Error("IsProcessed() should return false for unmarked file")
	}

	err = tracker.MarkProcessed(filePath)
	if err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}
	if !tracker.IsProcessed(filePath) {
		t.Error("IsProcessed() should return true for marked file")
	}
}

func TestLoadExistingFiles(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	tracker1, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	filePaths := []string{
		"/path/to/file1.txt",
		"/path/to/file2.txt",
		"/path/to/file3.txt",
	}

	for _, path := range filePaths {
		err = tracker1.MarkProcessed(path)
		if err != nil {
			t.Fatalf("MarkProcessed() error = %v", err)
		}
	}

	err = tracker1.Flush()
	if err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	tracker2, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker2.Close()

	if tracker2.GetProcessedCount() != 3 {
		t.Errorf("Expected 3 loaded files, got %d", tracker2.GetProcessedCount())
	}
	for _, path := range filePaths {
		if !tracker2.IsProcessed(path) {
			t.Errorf("IsProcessed() should return true for loaded file: %s", path)
		}
	}

	err = tracker1.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestFlush(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	tracker, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	filePath := "/path/to/file.txt"
	err = tracker.MarkProcessed(filePath)
	if err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}

	progressFile := filepath.Join(tempDir, ProgressFileName)
	data, err := os.ReadFile(progressFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if contains(data, filePath) {
		t.Error("Progress file should not contain path before flush")
	}

	err = tracker.Flush()
	if err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	data, err = os.ReadFile(progressFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !contains(data, filePath) {
		t.Error("Progress file should contain path after flush")
	}

	err = tracker.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestClose(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	tracker, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	filePath := "/path/to/file.txt"
	err = tracker.MarkProcessed(filePath)
	if err != nil {
		t.Fatalf("MarkProcessed() error = %v", err)
	}

	err = tracker.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	progressFile := filepath.Join(tempDir, ProgressFileName)
	_, err = os.Stat(progressFile)
	if !os.IsNotExist(err) {
		t.Error("Progress file should be deleted after close")
	}
}

func TestClean(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	tracker, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}

	filePaths := []string{
		"/path/to/file1.txt",
		"/path/to/file2.txt",
	}

	for _, path := range filePaths {
		err = tracker.MarkProcessed(path)
		if err != nil {
			t.Fatalf("MarkProcessed() error = %v", err)
		}
	}

	err = tracker.Flush()
	if err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if tracker.GetProcessedCount() != 2 {
		t.Errorf("Expected 2 processed files, got %d", tracker.GetProcessedCount())
	}

	err = tracker.Clean()
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}
	if tracker.GetProcessedCount() != 0 {
		t.Errorf("Expected 0 processed files after clean, got %d", tracker.GetProcessedCount())
	}

	for _, path := range filePaths {
		if tracker.IsProcessed(path) {
			t.Errorf("IsProcessed() should return false after clean: %s", path)
		}
	}

	err = tracker.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestExists(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	if Exists(tempDir) {
		t.Error("Exists() should return false when progress file doesn't exist")
	}

	tracker, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	err = tracker.Flush()
	if err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	if !Exists(tempDir) {
		t.Error("Exists() should return true when progress file exists")
	}

	err = tracker.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestGetProcessedCount(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	tracker, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	if tracker.GetProcessedCount() != 0 {
		t.Errorf("Expected 0 processed files, got %d", tracker.GetProcessedCount())
	}

	for i := 1; i <= 10; i++ {
		filePath := filepath.Join("/path/to/file", string(rune('0'+i)), ".txt")
		err = tracker.MarkProcessed(filePath)
		if err != nil {
			t.Fatalf("MarkProcessed() error = %v", err)
		}
		if tracker.GetProcessedCount() != i {
			t.Errorf("Expected %d processed files, got %d", i, tracker.GetProcessedCount())
		}
	}
}

func TestConcurrentMarkProcessed(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	tracker, err := NewTracker(tempDir)
	if err != nil {
		t.Fatalf("NewTracker() error = %v", err)
	}
	defer tracker.Close()

	numGoroutines := 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			filePath := filepath.Join("/path/to/file", string(rune('0'+id%10)), ".txt")
			err := tracker.MarkProcessed(filePath)
			if err != nil {
				t.Errorf("MarkProcessed() error = %v", err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	if tracker.GetProcessedCount() != 10 {
		t.Errorf("Expected 10 unique processed files, got %d", tracker.GetProcessedCount())
	}
}

func contains(data []byte, substr string) bool {
	for i := 0; i <= len(data)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if data[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
