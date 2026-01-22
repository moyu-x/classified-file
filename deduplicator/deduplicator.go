package deduplicator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/moyu-x/classified-file/database"
	"github.com/moyu-x/classified-file/hasher"
	"github.com/moyu-x/classified-file/internal"
	"github.com/moyu-x/classified-file/logger"
	"github.com/moyu-x/classified-file/scanner"
)

type Deduplicator struct {
	db           *database.Database
	mode         internal.OperationMode
	targetDir    string
	stats        internal.ProcessStats
	progressChan chan internal.ProgressUpdate
	totalFiles   int
	verbose      bool
}

func NewDeduplicator(db *database.Database, mode internal.OperationMode, targetDir string, totalFiles int, verbose bool) *Deduplicator {
	logger.Get().Info().Msgf("创建去重处理器，模式: %s", mode)
	if targetDir != "" {
		logger.Get().Info().Msgf("目标目录: %s", targetDir)
	}
	return &Deduplicator{
		db:           db,
		mode:         mode,
		targetDir:    targetDir,
		progressChan: make(chan internal.ProgressUpdate, 100),
		totalFiles:   totalFiles,
		verbose:      verbose,
	}
}

func (d *Deduplicator) Process(dirs []string) (*internal.ProcessStats, error) {
	logger.Get().Info().Msgf("开始处理文件，目录数: %d", len(dirs))

	d.stats = internal.ProcessStats{
		StartTime: time.Now(),
	}

	walker := scanner.NewFileWalker()
	d.processFiles(walker, dirs)

	duration := time.Since(d.stats.StartTime)
	logger.Get().Info().Msgf("文件处理完成，总耗时: %v", duration)
	return &d.stats, nil
}

func (d *Deduplicator) processFiles(walker *scanner.FileWalker, dirs []string) {
	defer close(d.progressChan)

	for _, dir := range dirs {
		walker.Walk(dir, func(path string, info os.FileInfo) error {
			hash, err := hasher.CalculateHash(path)
			if err != nil {
				logger.Get().Error().Err(err).Msgf("处理文件失败: %s", path)
				return nil
			}

			hashStr := fmt.Sprintf("%x", hash)

			exists, err := d.db.Exists(hashStr)
			if err != nil {
				logger.Get().Error().Err(err).Msgf("查询数据库失败: %s", path)
				return nil
			}

			if exists {
				switch d.mode {
				case internal.ModeDelete:
					if err := os.Remove(path); err == nil {
						d.stats.Deleted++
						d.stats.FreedSpace += info.Size()
						if d.verbose {
							logger.Get().Info().Msgf("[%d/%d] 发现重复: %s (%s, 已删除, 哈希: %s)",
								d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), hashStr[:16]+"...")
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
									d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), dstPath, hashStr[:16]+"...")
							} else {
								logger.Get().Info().Msgf("[%d/%d] 发现重复: %s (%s, 已移动到 %s [重命名])",
									d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), dstPath)
							}
						} else {
							if d.verbose {
								logger.Get().Info().Msgf("[%d/%d] 发现重复: %s (%s, 已移动到 %s, 哈希: %s)",
									d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), dstPath, hashStr[:16]+"...")
							} else {
								logger.Get().Info().Msgf("[%d/%d] 发现重复: %s (%s, 已移动到 %s)",
									d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), dstPath)
							}
						}
					} else {
						logger.Get().Error().Err(err).Msgf("移动文件失败: %s", path)
					}
				}
			} else {
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
							d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()), hashStr[:16]+"...")
					} else {
						logger.Get().Info().Msgf("[%d/%d] 新增记录: %s (%s)",
							d.stats.TotalProcessed+1, d.totalFiles, path, formatBytes(info.Size()))
					}
				}
			}

			d.stats.TotalProcessed++

			d.progressChan <- internal.ProgressUpdate{
				Processed:   d.stats.TotalProcessed,
				Added:       d.stats.Added,
				Deleted:     d.stats.Deleted,
				Moved:       d.stats.Moved,
				CurrentFile: path,
			}

			return nil
		})
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
