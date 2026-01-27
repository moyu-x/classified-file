package classifier

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifier_Classify(t *testing.T) {
	tempDir := t.TempDir()

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	testFiles := map[string]string{
		"test.jpg":      "\xff\xd8\xff\xe0\x00\x10JFIF",
		"test.png":      "\x89PNG\r\n\x1a\n",
		"test.pdf":      "%PDF-1.4",
		"test.mp3":      "ID3\x04\x00\x00\x00\x00\x00\x00",
		"test.zip":      "PK\x03\x04",
		"test.unknown":  "random content",
		"sub/test2.jpg": "\xff\xd8\xff\xe0\x00\x10JFIF",
	}

	for filename, content := range testFiles {
		fullPath := filepath.Join(sourceDir, filename)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("创建子目录失败: %v", err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	cls := NewClassifierWithCustomFilesPerDir(2)

	stats, err := cls.Classify([]string{sourceDir}, destDir)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	if stats.Processed != 6 {
		t.Errorf("Expected 6 processed files, got %d", stats.Processed)
	}

	if stats.UnknownType != 1 {
		t.Errorf("Expected 1 unknown type file, got %d", stats.UnknownType)
	}

	if stats.TotalProcessed != 7 {
		t.Errorf("Expected 7 total files, got %d", stats.TotalProcessed)
	}

	expectedDirs := []string{"image", "audio", "archive"}
	for _, dir := range expectedDirs {
		typeDir := filepath.Join(destDir, dir)
		if _, err := os.Stat(typeDir); os.IsNotExist(err) {
			t.Errorf("Expected type directory %s to exist", dir)
		}
	}

	imageDir := filepath.Join(destDir, "image")
	part0Dir := filepath.Join(imageDir, "part_0000")
	if _, err := os.Stat(part0Dir); os.IsNotExist(err) {
		t.Error("Expected image/part_0000 directory to exist")
	}

	part0Files, err := os.ReadDir(part0Dir)
	if err != nil {
		t.Fatalf("读取 image/part_0000 目录失败: %v", err)
	}

	if len(part0Files) != 2 {
		t.Errorf("Expected 2 files in image/part_0000, got %d", len(part0Files))
	}

	archiveDir := filepath.Join(destDir, "archive")
	archivePart0Dir := filepath.Join(archiveDir, "part_0000")
	if _, err := os.Stat(archivePart0Dir); os.IsNotExist(err) {
		t.Error("Expected archive/part_0000 directory to exist")
	}

	audioDir := filepath.Join(destDir, "audio")
	audioPart0Dir := filepath.Join(audioDir, "part_0000")
	if _, err := os.Stat(audioPart0Dir); os.IsNotExist(err) {
		t.Error("Expected audio/part_0000 directory to exist")
	}
}

func TestClassifier_HandleDuplicate(t *testing.T) {
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	cls := NewClassifier()

	newPath, err := cls.handleDuplicate(testFile)
	if err != nil {
		t.Fatalf("handleDuplicate() error = %v", err)
	}

	if !strings.HasSuffix(newPath, "_1.txt") {
		t.Errorf("Expected filename to end with _1.txt, got %s", newPath)
	}

	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Error("File should not exist (handleDuplicate only returns a path)")
	}
}

func TestClassifier_DetectFileType(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		filename    string
		content     string
		expectedExt string
	}{
		{"test.jpg", "\xff\xd8\xff\xe0\x00\x10JFIF", "jpg"},
		{"test.png", "\x89PNG\r\n\x1a\n", "png"},
		{"test.pdf", "%PDF-1.4", "pdf"},
		{"test.mp3", "ID3\x04\x00\x00\x00\x00\x00\x00", "mp3"},
		{"test.zip", "PK\x03\x04", "zip"},
	}

	cls := NewClassifier()

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tc.filename)

			if err := os.WriteFile(testFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("创建测试文件失败: %v", err)
			}

			detectedType, err := cls.detectFileType(testFile)
			if err != nil {
				t.Fatalf("detectFileType() error = %v", err)
			}

			if detectedType.Extension != tc.expectedExt {
				t.Errorf("Expected extension %s, got %s", tc.expectedExt, detectedType.Extension)
			}
		})
	}
}

func TestClassifier_IsImage(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		filename string
		content  string
		shouldBe bool
	}{
		{"test.jpg", "\xff\xd8\xff\xe0\x00\x10JFIF", true},
		{"test.png", "\x89PNG\r\n\x1a\n", true},
		{"test.pdf", "%PDF-1.4", false},
		{"test.mp3", "ID3\x04\x00\x00\x00\x00\x00\x00", false},
	}

	cls := NewClassifier()

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tc.filename)

			if err := os.WriteFile(testFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("创建测试文件失败: %v", err)
			}

			isImage := cls.IsImage(testFile)
			if isImage != tc.shouldBe {
				t.Errorf("IsImage() = %v, want %v", isImage, tc.shouldBe)
			}
		})
	}
}

func TestClassifier_IsVideo(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		filename string
		content  string
		shouldBe bool
	}{
		{"test.mp4", "\x00\x00\x00\x18ftypmp42", true},
		{"test.jpg", "\xff\xd8\xff\xe0\x00\x10JFIF", false},
	}

	cls := NewClassifier()

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tc.filename)

			if err := os.WriteFile(testFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("创建测试文件失败: %v", err)
			}

			isVideo := cls.IsVideo(testFile)
			if isVideo != tc.shouldBe {
				t.Errorf("IsVideo() = %v, want %v", isVideo, tc.shouldBe)
			}
		})
	}
}

func TestClassifier_CopyFile(t *testing.T) {
	tempDir := t.TempDir()

	srcFile := filepath.Join(tempDir, "src.txt")
	dstFile := filepath.Join(tempDir, "dst.txt")

	content := "test content for copy"
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("创建源文件失败: %v", err)
	}

	cls := NewClassifier()

	if err := cls.copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Error("目标文件未被创建")
	}

	copiedContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("读取目标文件失败: %v", err)
	}

	if string(copiedContent) != content {
		t.Errorf("复制内容不匹配，期望 %s，得到 %s", content, string(copiedContent))
	}
}

func TestNewClassifier(t *testing.T) {
	cls := NewClassifier()

	if cls.filesPerDir != FilesPerDir {
		t.Errorf("Expected filesPerDir %d, got %d", FilesPerDir, cls.filesPerDir)
	}

	if cls.fileCounters == nil {
		t.Error("Expected fileCounters to be initialized")
	}
}

func TestNewClassifierWithCustomFilesPerDir(t *testing.T) {
	customFilesPerDir := 100
	cls := NewClassifierWithCustomFilesPerDir(customFilesPerDir)

	if cls.filesPerDir != customFilesPerDir {
		t.Errorf("Expected filesPerDir %d, got %d", customFilesPerDir, cls.filesPerDir)
	}
}

func TestParseFilesPerDir(t *testing.T) {
	testCases := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"500", 500, false},
		{"100", 100, false},
		{"1", 1, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := ParseFilesPerDir(tc.input)

			if tc.hasError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tc.expected {
					t.Errorf("Expected %d, got %d", tc.expected, result)
				}
			}
		})
	}
}

func TestClassifierStats_String(t *testing.T) {
	stats := &ClassifierStats{
		TotalProcessed: 100,
		Processed:      90,
		Failed:         5,
		UnknownType:    5,
	}

	output := stats.String()

	if !strings.Contains(output, "总文件数: 100") {
		t.Error("输出应包含总文件数统计")
	}

	if !strings.Contains(output, "已处理: 90") {
		t.Error("输出应包含已处理统计")
	}

	if !strings.Contains(output, "失败: 5") {
		t.Error("输出应包含失败统计")
	}

	if !strings.Contains(output, "未知类型: 5") {
		t.Error("输出应包含未知类型统计")
	}
}

func TestClassifier_Classify_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	sourceDir := filepath.Join(tempDir, "source")
	destDir := filepath.Join(tempDir, "dest")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("创建源目录失败: %v", err)
	}

	cls := NewClassifier()

	stats, err := cls.Classify([]string{sourceDir}, destDir)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	if stats.TotalProcessed != 0 {
		t.Errorf("Expected 0 total files, got %d", stats.TotalProcessed)
	}

	if stats.Processed != 0 {
		t.Errorf("Expected 0 processed files, got %d", stats.Processed)
	}
}

func TestClassifier_Classify_MultipleSourceDirs(t *testing.T) {
	tempDir := t.TempDir()

	sourceDir1 := filepath.Join(tempDir, "source1")
	sourceDir2 := filepath.Join(tempDir, "source2")
	destDir := filepath.Join(tempDir, "dest")

	for _, dir := range []string{sourceDir1, sourceDir2} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("创建源目录失败: %v", err)
		}
	}

	testFile1 := filepath.Join(sourceDir1, "test1.jpg")
	testFile2 := filepath.Join(sourceDir2, "test2.png")

	if err := os.WriteFile(testFile1, []byte("\xff\xd8\xff\xe0\x00\x10JFIF"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	if err := os.WriteFile(testFile2, []byte("\x89PNG\r\n\x1a\n"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	cls := NewClassifier()

	stats, err := cls.Classify([]string{sourceDir1, sourceDir2}, destDir)
	if err != nil {
		t.Fatalf("Classify() error = %v", err)
	}

	if stats.TotalProcessed != 2 {
		t.Errorf("Expected 2 total files, got %d", stats.TotalProcessed)
	}

	if stats.Processed != 2 {
		t.Errorf("Expected 2 processed files, got %d", stats.Processed)
	}
}
