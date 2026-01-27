package classifier

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/h2non/filetype"
	"github.com/h2non/filetype/types"
	"github.com/moyu-x/classified-file/pkg/logger"
	"github.com/moyu-x/classified-file/pkg/scanner"
)

const (
	FilesPerDir = 500
	BufferSize  = 8192
)

type Classifier struct {
	walker         *scanner.FileWalker
	filesPerDir    int
	fileCounters   map[string]int
	fileCountersMu sync.Mutex
}

type ClassifierStats struct {
	TotalProcessed int
	Processed      int
	Failed         int
	UnknownType    int
}

func NewClassifier() *Classifier {
	return &Classifier{
		walker:       scanner.NewFileWalker(),
		filesPerDir:  FilesPerDir,
		fileCounters: make(map[string]int),
	}
}

func NewClassifierWithCustomFilesPerDir(filesPerDir int) *Classifier {
	return &Classifier{
		walker:       scanner.NewFileWalker(),
		filesPerDir:  filesPerDir,
		fileCounters: make(map[string]int),
	}
}

func (c *Classifier) Classify(sourceDirs []string, destDir string) (*ClassifierStats, error) {
	logger.Get().Info().Msgf("开始分类文件，共 %d 个源目录", len(sourceDirs))
	logger.Get().Info().Msgf("目标目录: %s", destDir)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		logger.Get().Error().Err(err).Msg("创建目标目录失败")
		return nil, fmt.Errorf("创建目标目录: %w", err)
	}

	stats := &ClassifierStats{}

	for _, sourceDir := range sourceDirs {
		if err := c.classifyDirectory(sourceDir, destDir, stats); err != nil {
			return stats, err
		}
	}

	logger.Get().Info().Msg("文件分类完成")
	return stats, nil
}

func (c *Classifier) classifyDirectory(sourceDir, destDir string, stats *ClassifierStats) error {
	logger.Get().Info().Msgf("处理目录: %s", sourceDir)

	err := c.walker.Walk(sourceDir, func(filePath string, info os.FileInfo) error {
		stats.TotalProcessed++

		if err := c.processFile(filePath, destDir, stats); err != nil {
			logger.Get().Error().Err(err).Msgf("处理文件失败: %s", filePath)
			stats.Failed++
		}

		return nil
	})

	if err != nil {
		logger.Get().Error().Err(err).Msgf("遍历目录失败: %s", sourceDir)
		return err
	}

	return nil
}

func (c *Classifier) processFile(filePath, destDir string, stats *ClassifierStats) error {
	fileType, err := c.detectFileType(filePath)
	if err != nil {
		logger.Get().Error().Err(err).Msgf("检测文件类型失败: %s", filePath)
		return err
	}

	if fileType == types.Unknown {
		logger.Get().Debug().Msgf("未知文件类型: %s", filePath)
		stats.UnknownType++
		return nil
	}

	category := c.getFileCategory(fileType)
	categoryDir := filepath.Join(destDir, category)
	if err := os.MkdirAll(categoryDir, 0755); err != nil {
		return fmt.Errorf("创建类型目录: %w", err)
	}

	subDirIndex, err := c.getSubDirIndex(categoryDir)
	if err != nil {
		return err
	}

	subDirName := fmt.Sprintf("part_%04d", subDirIndex)
	targetSubDir := filepath.Join(categoryDir, subDirName)

	if err := os.MkdirAll(targetSubDir, 0755); err != nil {
		return fmt.Errorf("创建子目录: %w", err)
	}

	fileName := filepath.Base(filePath)
	targetPath := filepath.Join(targetSubDir, fileName)

	targetPath, err = c.handleDuplicate(targetPath)
	if err != nil {
		return err
	}

	if err := c.copyFile(filePath, targetPath); err != nil {
		return fmt.Errorf("复制文件: %w", err)
	}

	stats.Processed++
	logger.Get().Debug().Msgf("已处理: %s -> %s (%s)", filePath, targetPath, category)

	return nil
}

func (c *Classifier) getFileCategory(fileType types.Type) string {
	mime := fileType.MIME.Value

	// Check MIME type prefix
	if len(mime) >= 5 {
		prefix := mime[:5]
		switch prefix {
		case "image":
			return "image"
		case "video":
			return "video"
		case "audio":
			return "audio"
		}
	}

	// Check for documents and other types based on extension
	ext := fileType.Extension
	switch ext {
	case "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "txt", "rtf", "odt", "ods", "odp":
		return "document"
	case "zip", "tar", "gz", "bz2", "rar", "7z", "xz":
		return "archive"
	}

	return "other"
}

func (c *Classifier) detectFileType(filePath string) (types.Type, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return types.Unknown, err
	}
	defer file.Close()

	buffer := make([]byte, BufferSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return types.Unknown, err
	}

	return filetype.Match(buffer[:n])
}

func (c *Classifier) getSubDirIndex(categoryDir string) (int, error) {
	c.fileCountersMu.Lock()
	defer c.fileCountersMu.Unlock()

	key := filepath.Base(categoryDir)
	currentCount, exists := c.fileCounters[key]

	if !exists {
		files, err := c.countFilesInDir(categoryDir)
		if err != nil {
			return 0, err
		}
		currentCount = files
		c.fileCounters[key] = currentCount
	}

	subDirIndex := currentCount / c.filesPerDir
	c.fileCounters[key] = currentCount + 1

	return subDirIndex, nil
}

func (c *Classifier) countFilesInDir(dirPath string) (int, error) {
	count := 0

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() {
			count++
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return count, nil
}

func (c *Classifier) handleDuplicate(targetPath string) (string, error) {
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return targetPath, nil
	}

	ext := filepath.Ext(targetPath)
	baseName := targetPath[:len(targetPath)-len(ext)]

	for i := 1; ; i++ {
		newPath := fmt.Sprintf("%s_%d%s", baseName, i, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath, nil
		}
	}
}

func (c *Classifier) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, _, err = copyBuffer(destFile, sourceFile); err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

func copyBuffer(dst io.Writer, src io.Reader) (int64, int64, error) {
	buf := make([]byte, BufferSize)
	var written, total int64

	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			return total, written, err
		}
		if n == 0 {
			break
		}

		nw, err := dst.Write(buf[:n])
		if err != nil {
			return total, written, err
		}

		written += int64(nw)
		total += int64(n)
	}

	return total, written, nil
}

func (c *Classifier) detectFileTypeFromContent(content []byte) (types.Type, error) {
	return filetype.Match(content)
}

func (c *Classifier) IsImage(filePath string) bool {
	buffer, err := c.readFileBuffer(filePath)
	if err != nil {
		return false
	}
	return filetype.IsImage(buffer)
}

func (c *Classifier) IsVideo(filePath string) bool {
	buffer, err := c.readFileBuffer(filePath)
	if err != nil {
		return false
	}
	return filetype.IsVideo(buffer)
}

func (c *Classifier) IsAudio(filePath string) bool {
	buffer, err := c.readFileBuffer(filePath)
	if err != nil {
		return false
	}
	return filetype.IsAudio(buffer)
}

func (c *Classifier) IsDocument(filePath string) bool {
	buffer, err := c.readFileBuffer(filePath)
	if err != nil {
		return false
	}
	return filetype.IsDocument(buffer)
}

func (c *Classifier) IsArchive(filePath string) bool {
	buffer, err := c.readFileBuffer(filePath)
	if err != nil {
		return false
	}
	return filetype.IsArchive(buffer)
}

func (c *Classifier) readFileBuffer(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := make([]byte, BufferSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buffer[:n], nil
}

func (c *Classifier) GetFileType(filePath string) (string, error) {
	fileType, err := c.detectFileType(filePath)
	if err != nil {
		return "", err
	}

	if fileType == types.Unknown {
		return "unknown", nil
	}

	return fileType.Extension, nil
}

func (s *ClassifierStats) String() string {
	var buf bytes.Buffer

	buf.WriteString("========== 分类统计 ==========\n")
	buf.WriteString(fmt.Sprintf("总文件数: %d\n", s.TotalProcessed))
	buf.WriteString(fmt.Sprintf("已处理: %d\n", s.Processed))
	buf.WriteString(fmt.Sprintf("失败: %d\n", s.Failed))
	buf.WriteString(fmt.Sprintf("未知类型: %d\n", s.UnknownType))

	if s.TotalProcessed > 0 {
		successRate := float64(s.Processed) / float64(s.TotalProcessed) * 100
		buf.WriteString(fmt.Sprintf("成功率: %.2f%%\n", successRate))
	}

	buf.WriteString("============================")

	return buf.String()
}

func ParseFilesPerDir(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("无效的数字: %w", err)
	}

	if n <= 0 {
		return 0, fmt.Errorf("每目录文件数必须大于 0")
	}

	return n, nil
}
