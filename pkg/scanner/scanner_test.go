package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFileWalker_Walk(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	testFiles := []string{
		"file1.txt",
		"file2.txt",
		".hidden_file",
		"subdir/file3.txt",
		".hidden_dir/.hidden_file2",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	walker := NewFileWalker()
	visitedFiles := []string{}

	err := walker.Walk(tempDir, func(path string, info os.FileInfo) error {
		relPath, _ := filepath.Rel(tempDir, path)
		visitedFiles = append(visitedFiles, relPath)
		return nil
	})

	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	if len(visitedFiles) != len(testFiles) {
		t.Errorf("Expected %d files, got %d", len(testFiles), len(visitedFiles))
	}

	for _, expectedFile := range testFiles {
		found := false
		for _, visitedFile := range visitedFiles {
			if visitedFile == expectedFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("File %s not found in visited files", expectedFile)
		}
	}
}

func TestFileWalker_CountFiles(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	testDirs := []string{"dir1", "dir2"}
	filesPerDir := 5

	for _, dir := range testDirs {
		dirPath := filepath.Join(tempDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		for i := 0; i < filesPerDir; i++ {
			filePath := filepath.Join(dirPath, fmt.Sprintf("file%d.txt", i))
			if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}
	}

	walker := NewFileWalker()
	dirs := []string{
		filepath.Join(tempDir, "dir1"),
		filepath.Join(tempDir, "dir2"),
	}

	count, err := walker.CountFiles(dirs)
	if err != nil {
		t.Fatalf("CountFiles() error = %v", err)
	}

	expectedCount := len(testDirs) * filesPerDir
	if count != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, count)
	}
}

func TestFileWalker_CountFiles_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	walker := NewFileWalker()
	count, err := walker.CountFiles([]string{tempDir})

	if err != nil {
		t.Fatalf("CountFiles() error = %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 files, got %d", count)
	}
}

func TestFileWalker_CountFiles_NonExistentDir(t *testing.T) {
	walker := NewFileWalker()
	_, err := walker.CountFiles([]string{"/non/existent/directory"})

	if err != nil {
		t.Errorf("Expected no error for non-existent directory, got %v", err)
	}
}

func TestFileWalker_Walk_WithSymlinks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping symlink test in short mode")
	}

	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	linkPath := filepath.Join(tempDir, "link.txt")
	if err := os.Symlink(filePath, linkPath); err != nil {
		t.Skipf("Skipping symlink test: %v", err)
	}

	walker := NewFileWalker()
	count := 0
	err := walker.Walk(tempDir, func(path string, info os.FileInfo) error {
		if !info.IsDir() {
			count++
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Walk() error = %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 files (original + symlink), got %d", count)
	}
}
