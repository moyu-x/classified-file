package hasher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateHash(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	testContent := []byte("test content for hashing")
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, err := CalculateHash(testFile)
	if err != nil {
		t.Fatalf("CalculateHash() error = %v", err)
	}

	if hash == 0 {
		t.Error("Expected non-zero hash")
	}

	hash2, err := CalculateHash(testFile)
	if err != nil {
		t.Fatalf("CalculateHash() second call error = %v", err)
	}

	if hash != hash2 {
		t.Error("Hash should be consistent for same file")
	}
}

func TestCalculateHash_DifferentContent(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")

	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash1, err := CalculateHash(file1)
	if err != nil {
		t.Fatalf("CalculateHash() error = %v", err)
	}

	hash2, err := CalculateHash(file2)
	if err != nil {
		t.Fatalf("CalculateHash() error = %v", err)
	}

	if hash1 == hash2 {
		t.Error("Different content should produce different hashes")
	}
}

func TestCalculateHash_NonExistentFile(t *testing.T) {
	_, err := CalculateHash("/non/existent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestCalculateHash_LargeFile(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	largeFile := filepath.Join(tempDir, "large.txt")
	const fileSize = 10 * 1024 * 1024

	file, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	data := make([]byte, 4096)
	for i := 0; i < fileSize/4096; i++ {
		if _, err := file.Write(data); err != nil {
			file.Close()
			t.Fatalf("Failed to write to large file: %v", err)
		}
	}
	file.Close()

	hash, err := CalculateHash(largeFile)
	if err != nil {
		t.Fatalf("CalculateHash() error = %v", err)
	}

	if hash == 0 {
		t.Error("Expected non-zero hash for large file")
	}
}
