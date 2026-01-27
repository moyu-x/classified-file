package deduplicator

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/moyu-x/classified-file/pkg/database"
	"github.com/moyu-x/classified-file/pkg/hasher"
	"github.com/moyu-x/classified-file/internal"
	"github.com/moyu-x/classified-file/pkg/logger"
	"github.com/moyu-x/classified-file/pkg/progress"
	"github.com/moyu-x/classified-file/pkg/scanner"
)

type Deduplicator struct {
	db           *database.Database
	mode         internal.OperationMode
	targetDir    string
	stats        internal.ProcessStats
	progressChan chan internal.ProgressUpdate
	totalFiles   int
	verbose      bool

	trackers   map[string]*progress.Tracker
	resumeMode bool
	resetMode  bool
}

var globalDedup *Deduplicator

func NewDeduplicator(db *database.Database, mode internal.OperationMode, targetDir string, verbose bool) *Deduplicator {
	logger.Get().Info().Msgf("创建去重处理器，模式: %s", mode)
	if targetDir != "" {
		logger.Get().Info().Msgf("目标目录: %s", targetDir)
	}
	dedup := &Deduplicator{
		db:           db,
		mode:         mode,
		targetDir:    targetDir,
		progressChan: make(chan internal.ProgressUpdate, 100),
		verbose:      verbose,
		trackers:     make(map[string]*progress.Tracker),
	}
	globalDedup = dedup
	return dedup
}

func SetupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Get().Warn().Msgf("收到信号 %v，正在优雅关闭...", sig)
		if globalDedup != nil {
			globalDedup.HandleInterrupt()
		}
		os.Exit(0)
	}()
}

func (d *Deduplicator) Process(dirs []string, resume, reset bool) (*internal.ProcessStats, error) {
	d.resumeMode = resume
	d.resetMode = reset

	logger.Get().Info().Msgf("扫描模式: resume=%v, reset=%v", resume, reset)

	d.stats = internal.ProcessStats{
		StartTime: time.Now(),
	}

	for _, dir := range dirs {
		rootDir := getRootDir(dir)
		progressRoot := getProgressRoot(dir)

		if reset {
			if progress.Exists(progressRoot) {
				logger.Get().Info().Msgf("删除进度文件: %s", progressRoot)
				progressFile := filepath.Join(progressRoot, progress.ProgressFileName)
				if err := os.Remove(progressFile); err != nil && !os.IsNotExist(err) {
					logger.Get().Error().Err(err).Msgf("删除进度文件失败: %s", progressRoot)
				}
			}
		}

		tracker, err := progress.NewTracker(progressRoot)
		if err != nil {
			logger.Get().Error().Err(err).Msgf("创建进度跟踪器失败: %s", progressRoot)
			return nil, err
		}

		d.trackers[rootDir] = tracker

		processedCount := tracker.GetProcessedCount()
		if processedCount > 0 {
			logger.Get().Info().Msgf("发现未完成的扫描，已处理 %d 个文件: %s", processedCount, rootDir)
		}
	}

	walker := scanner.NewFileWalker()
	totalFiles, err := walker.CountFiles(dirs)
	if err != nil {
		return nil, fmt.Errorf("统计文件数量失败: %w", err)
	}
	d.totalFiles = totalFiles

	logger.Get().Info().Msgf("文件统计完成，共找到 %d 个文件", totalFiles)

	d.processFiles(walker, dirs)

	for rootDir, tracker := range d.trackers {
		if err := tracker.Close(); err != nil {
			logger.Get().Error().Err(err).Msgf("关闭进度跟踪器失败: %s", rootDir)
		}
	}

	d.stats.EndTime = time.Now()
	duration := d.stats.EndTime.Sub(d.stats.StartTime)
	logger.Get().Info().Msgf("文件处理完成，总耗时: %v", duration)
	logger.Get().Info().Msgf("统计: TotalProcessed=%d, Added=%d, Deleted=%d, Moved=%d",
		d.stats.TotalProcessed, d.stats.Added, d.stats.Deleted, d.stats.Moved)
	return &d.stats, nil
}

func (d *Deduplicator) processFiles(walker *scanner.FileWalker, dirs []string) {
	for _, dir := range dirs {
		rootDir := getRootDir(dir)
		tracker := d.trackers[rootDir]

		walker.Walk(dir, func(path string, info os.FileInfo) error {
			if tracker.IsProcessed(path) {
				if d.resumeMode {
					d.stats.TotalProcessed++
					if d.verbose {
						logger.Get().Debug().Msgf("[%d/%d] 跳过已处理文件: %s",
							d.stats.TotalProcessed, d.totalFiles, path)
					}
					return nil
				}
			}

			hash, err := hasher.CalculateHash(path)
			if err != nil {
				logger.Get().Error().Err(err).Msgf("处理文件失败: %s", path)
				return nil
			}

			hashStr := fmt.Sprintf("%016x", hash)
			logger.Get().Debug().Msgf("File hash: %s = %s", path, hashStr)

			exists, err := d.db.Exists(hashStr)
			if err != nil {
				logger.Get().Error().Err(err).Msgf("查询数据库失败: %s", path)
				return nil
			}

			if exists {
				logger.Get().Debug().Msgf("File is duplicate (hash exists): %s", path)
				d.handleDuplicate(path, info, hashStr)
			} else {
				logger.Get().Debug().Msgf("File is new (hash not in DB): %s", path)
				record := &internal.FileRecord{
					Hash:      hashStr,
					FilePath:  path,
					FileSize:  info.Size(),
					CreatedAt: time.Now().Unix(),
				}
				if err := d.db.Insert(record); err == nil {
					d.stats.Added++
					if d.verbose {
						logger.Get().Info().Msgf("[%d/%d] 新增记录: %s (%s, 哈希: %s)",
							d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), hashStr)
					} else {
						logger.Get().Info().Msgf("[%d/%d] 新增记录: %s (%s)",
							d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()))
					}
				}
			}

			if err := tracker.MarkProcessed(path); err != nil {
				logger.Get().Error().Err(err).Msgf("标记文件已处理失败: %s", path)
			}

			d.stats.TotalProcessed++
			return nil
		})
	}
}

func (d *Deduplicator) handleDuplicate(path string, info os.FileInfo, hashStr string) {
	switch d.mode {
	case internal.ModeDelete:
		if err := os.Remove(path); err == nil {
			d.stats.Deleted++
			d.stats.FreedSpace += info.Size()
			if d.verbose {
				logger.Get().Info().Msgf("[%d/%d] 发现重复: %s (%s, 已删除, 哈希: %s)",
					d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), hashStr)
			} else {
				logger.Get().Info().Msgf("[%d/%d] 发现重复: %s (%s, 已删除)",
					d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()))
			}
		} else {
			logger.Get().Error().Err(err).Msgf("删除文件失败: %s", path)
		}
	case internal.ModeMove:
		if err := d.moveFile(path, hashStr); err == nil {
			d.stats.Moved++
			dstPath := d.buildDstPath(path, hashStr)
			if strings.Contains(filepath.Base(dstPath), "_") && !strings.HasPrefix(filepath.Base(dstPath), hashStr[:8]+"_"+hashStr[8:]) {
				if d.verbose {
					logger.Get().Info().Msgf("[%d/%d] 发现重复: %s (%s, 已移动到 %s [重命名], 哈希: %s)",
						d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), dstPath, hashStr)
				} else {
					logger.Get().Info().Msgf("[%d/%d] 发现重复: %s (%s, 已移动到 %s [重命名])",
						d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), dstPath)
				}
			} else {
				if d.verbose {
					logger.Get().Info().Msgf("[%d/%d] 发现重复: %s (%s, 已移动到 %s, 哈希: %s)",
						d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), dstPath, hashStr)
				} else {
					logger.Get().Info().Msgf("[%d/%d] 发现重复: %s (%s, 已移动到 %s)",
						d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), dstPath)
				}
			}
		} else {
			logger.Get().Error().Err(err).Msgf("移动文件失败: %s", path)
		}
	}
}

func (d *Deduplicator) moveFile(srcPath, hash string) error {
	if d.targetDir == "" {
		return fmt.Errorf("target directory not specified")
	}

	if err := os.MkdirAll(d.targetDir, 0755); err != nil {
		return err
	}

	filename := filepath.Base(srcPath)
	ext := filepath.Ext(filename)

	baseName := hash[:8] + "_" + hash[8:]
	dstPath := filepath.Join(d.targetDir, baseName+ext)

	conflictCounter := 0
	for {
		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			break
		} else if err != nil {
			return fmt.Errorf("检查目标文件失败: %w", err)
		}

		conflictCounter++
		newBaseName := fmt.Sprintf("%s_%d", baseName, conflictCounter)
		dstPath = filepath.Join(d.targetDir, newBaseName+ext)

		if conflictCounter == 1 {
			logger.Get().Warn().Msgf("目标文件已存在，尝试重命名: %s", dstPath)
		}

		if conflictCounter >= 100 {
			return fmt.Errorf("无法生成唯一文件名，已尝试 %d 次", conflictCounter)
		}
	}

	logger.Get().Debug().Msgf("移动文件: %s -> %s", srcPath, dstPath)
	return os.Rename(srcPath, dstPath)
}

func (d *Deduplicator) buildDstPath(srcPath, hash string) string {
	filename := filepath.Base(srcPath)
	ext := filepath.Ext(filename)
	baseName := hash[:8] + "_" + hash[8:]
	dstPath := filepath.Join(d.targetDir, baseName+ext)

	conflictCounter := 0
	for {
		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			break
		}
		conflictCounter++
		newBaseName := fmt.Sprintf("%s_%d", baseName, conflictCounter)
		dstPath = filepath.Join(d.targetDir, newBaseName+ext)
		if conflictCounter >= 100 {
			break
		}
	}

	return dstPath
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (d *Deduplicator) Progress() <-chan internal.ProgressUpdate {
	return d.progressChan
}

func (d *Deduplicator) HandleInterrupt() {
	logger.Get().Warn().Msg("收到中断信号，正在保存状态...")

	for rootDir, tracker := range d.trackers {
		if err := tracker.Flush(); err != nil {
			logger.Get().Error().Err(err).Msgf("刷新进度文件失败: %s", rootDir)
		} else {
			logger.Get().Info().Msgf("进度文件已保存: %s (已处理 %d 个文件)",
				rootDir, tracker.GetProcessedCount())
		}
	}

	logger.Get().Warn().Msgf("中断处理完成，已处理: %d/%d 个文件", d.stats.TotalProcessed, d.totalFiles)
}

func getRootDir(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}

	absPath, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}

	return absPath
}

func getProgressRoot(dir string) string {
	absPath := filepath.Dir(getRootDir(dir))
	return absPath
}
