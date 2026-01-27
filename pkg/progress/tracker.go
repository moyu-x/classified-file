package progress

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"sync"

	"github.com/moyu-x/classified-file/pkg/logger"
)

const (
	ProgressFileName = ".classified-file-progress.txt"
)

type Tracker struct {
	rootDir      string
	filePath     string
	file         *os.File
	writer       *bufio.Writer
	seenFiles    map[string]bool // 内存缓存，加速查找
	mu           sync.RWMutex
	flushedCount int // 记录刷新次数
}

func NewTracker(rootDir string) (*Tracker, error) {
	filePath := filepath.Join(rootDir, ProgressFileName)

	// 打开或创建文件（追加模式）
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	tracker := &Tracker{
		rootDir:   rootDir,
		filePath:  filePath,
		file:      file,
		writer:    bufio.NewWriter(file),
		seenFiles: make(map[string]bool),
	}

	// 加载已存在的文件路径到内存
	if err := tracker.loadExistingFiles(); err != nil {
		logger.Get().Warn().Err(err).Msg("加载已扫描文件列表失败，将从零开始")
		tracker.seenFiles = make(map[string]bool)
	} else {
		logger.Get().Info().Msgf("从进度文件加载了 %d 个已扫描文件", len(tracker.seenFiles))
	}

	return tracker, nil
}

// loadExistingFiles 加载已存在的文件路径
func (t *Tracker) loadExistingFiles() error {
	// 读取文件内容
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	// 解析每行
	scanner := bufio.NewScanner(bytes.NewReader(data))
	count := 0
	for scanner.Scan() {
		path := scanner.Text()
		if path != "" {
			t.seenFiles[path] = true
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	logger.Get().Debug().Msgf("加载了 %d 个已扫描文件到内存", count)
	return nil
}

// IsProcessed 检查文件是否已处理
func (t *Tracker) IsProcessed(path string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.seenFiles[path]
}

// MarkProcessed 标记文件为已处理
func (t *Tracker) MarkProcessed(path string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 如果已经处理过，跳过
	if t.seenFiles[path] {
		return nil
	}

	// 写入文件
	if _, err := t.writer.WriteString(path + "\n"); err != nil {
		return err
	}

	// 添加到内存缓存
	t.seenFiles[path] = true

	// 每100个文件刷新一次
	t.flushedCount++
	if t.flushedCount%100 == 0 {
		if err := t.writer.Flush(); err != nil {
			logger.Get().Error().Err(err).Msg("刷新进度文件失败")
		}
	}

	return nil
}

// Flush 强制刷新到磁盘
func (t *Tracker) Flush() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.writer.Flush()
}

// GetProcessedCount 获取已处理文件数
func (t *Tracker) GetProcessedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.seenFiles)
}

// Close 关闭并删除进度文件
func (t *Tracker) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	logger.Get().Info().Msgf("扫描完成，删除进度文件: %s", t.filePath)

	// 先刷新缓冲区
	if err := t.writer.Flush(); err != nil {
		return err
	}

	// 关闭文件
	if err := t.file.Close(); err != nil {
		return err
	}

	// 删除文件
	if err := os.Remove(t.filePath); err != nil && !os.IsNotExist(err) {
		logger.Get().Error().Err(err).Msgf("删除进度文件失败: %s", t.filePath)
		return err
	}

	logger.Get().Info().Msgf("进度文件已删除: %s", t.filePath)
	return nil
}

// Clean 清理进度文件（用于重置）
func (t *Tracker) Clean() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 关闭文件
	if t.file != nil {
		t.writer.Flush()
		t.file.Close()
	}

	// 删除文件
	if err := os.Remove(t.filePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// 清空内存缓存
	t.seenFiles = make(map[string]bool)

	// 重新创建文件
	file, err := os.OpenFile(t.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	t.file = file
	t.writer = bufio.NewWriter(file)
	t.flushedCount = 0

	logger.Get().Info().Msgf("进度文件已清理: %s", t.filePath)
	return nil
}

// Exists 检查进度文件是否存在
func Exists(rootDir string) bool {
	filePath := filepath.Join(rootDir, ProgressFileName)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}
