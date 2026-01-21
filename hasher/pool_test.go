package hasher

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHashPool_Start(t *testing.T) {
	pool := NewHashPool(2)
	pool.Start()

	time.Sleep(10 * time.Millisecond)

	pool.Close()
}

func TestHashPool_AddTask(t *testing.T) {
	pool := NewHashPool(2)
	pool.Start()
	defer pool.Close()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pool.AddTask(HashTask{Path: testFile, Size: 4})

	time.Sleep(100 * time.Millisecond)
}

func TestHashPool_Results(t *testing.T) {
	pool := NewHashPool(2)
	pool.Start()
	defer pool.Close()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := []byte("test content")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pool.AddTask(HashTask{Path: testFile, Size: int64(len(testContent))})

	resultReceived := false
	go func() {
		for result := range pool.Results() {
			if result.Error == nil && result.Path == testFile && result.Hash != 0 {
				resultReceived = true
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)

	if !resultReceived {
		t.Error("Expected to receive result from Results() channel")
	}
}

func TestHashPool_MultipleTasks(t *testing.T) {
	pool := NewHashPool(4)
	pool.Start()
	defer pool.Close()

	tempDir := t.TempDir()
	const numFiles = 10

	for i := 0; i < numFiles; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		content := []byte(fmt.Sprintf("content%d", i))
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		pool.AddTask(HashTask{Path: filePath, Size: int64(len(content))})
	}

	results := 0
	timeout := time.After(5 * time.Second)

resultLoop:
	for {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for results, got %d/%d", results, numFiles)
			break resultLoop
		case result, ok := <-pool.Results():
			if !ok {
				break resultLoop
			}
			if result.Error == nil {
				results++
				if results == numFiles {
					break resultLoop
				}
			}
		}
	}

	if results != numFiles {
		t.Errorf("Expected %d results, got %d", numFiles, results)
	}
}

func TestHashPool_ErrorHandling(t *testing.T) {
	pool := NewHashPool(2)
	pool.Start()
	defer pool.Close()

	pool.AddTask(HashTask{Path: "/non/existent/file", Size: 0})

	resultReceived := false
	timeout := time.After(1 * time.Second)

	for {
		select {
		case <-timeout:
			if !resultReceived {
				t.Error("Timeout waiting for result")
			}
			return
		case result, ok := <-pool.Results():
			if !ok {
				return
			}
			if result.Error != nil {
				resultReceived = true
				return
			}
		}
	}
}

func TestHashPool_Close(t *testing.T) {
	pool := NewHashPool(2)
	pool.Start()

	pool.Close()

	select {
	case _, ok := <-pool.Results():
		if ok {
			t.Error("Results channel should be closed after Close()")
		}
	default:
	}
}
