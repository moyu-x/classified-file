package deduplicator

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/moyu-x/classified-file/database"
	"github.com/moyu-x/classified-file/hasher"
	"github.com/moyu-x/classified-file/internal"
	"github.com/moyu-x/classified-file/logger"
	"github.com/moyu-x/classified-file/scanner"
)

type Deduplicator struct {
	db           *database.Database
	hasher       *hasher.HashPool
	mode         internal.OperationMode
	targetDir    string
	stats        internal.ProcessStats
	progressChan chan internal.ProgressUpdate
}

func NewDeduplicator(db *database.Database, mode internal.OperationMode, targetDir string) *Deduplicator {
	logger.Get().Info().Msgf("创建去重处理器，模式: %s", mode)
	if targetDir != "" {
		logger.Get().Info().Msgf("目标目录: %s", targetDir)
	}
	return &Deduplicator{
		db:           db,
		hasher:       hasher.NewHashPool(internal.DefaultWorkers),
		mode:         mode,
		targetDir:    targetDir,
		progressChan: make(chan internal.ProgressUpdate, 100),
	}
}

func (d *Deduplicator) Process(dirs []string) (*internal.ProcessStats, error) {
	logger.Get().Info().Msgf("开始处理文件，目录数: %d", len(dirs))

	d.hasher.Start()
	defer d.hasher.Close()

	d.stats = internal.ProcessStats{
		StartTime: time.Now(),
	}

	walker := scanner.NewFileWalker()
	d.submitTasks(walker, dirs)
	d.processResults()

	duration := time.Since(d.stats.StartTime)
	logger.Get().Info().Msgf("文件处理完成，总耗时: %v", duration)
	return &d.stats, nil
}

func (d *Deduplicator) submitTasks(walker *scanner.FileWalker, dirs []string) {
	for _, dir := range dirs {
		walker.Walk(dir, func(path string, info os.FileInfo) error {
			d.hasher.AddTask(hasher.HashTask{
				Path: path,
				Size: info.Size(),
			})
			return nil
		})
	}
}

func (d *Deduplicator) processResults() {
	defer close(d.progressChan)

	for result := range d.hasher.Results() {
		if result.Error != nil {
			logger.Get().Error().Err(result.Error).Msgf("处理文件失败: %s", result.Path)
			continue
		}

		hashStr := fmt.Sprintf("%x", result.Hash)

		exists, err := d.db.Exists(hashStr)
		if err != nil {
			logger.Get().Error().Err(err).Msgf("查询数据库失败: %s", result.Path)
			continue
		}

		if exists {
			logger.Get().Debug().Msgf("发现重复文件: %s", result.Path)
			switch d.mode {
			case internal.ModeDelete:
				if err := os.Remove(result.Path); err == nil {
					d.stats.Deleted++
					d.stats.FreedSpace += result.Size
					logger.Get().Debug().Msgf("已删除重复文件: %s (释放 %d bytes)", result.Path, result.Size)
				} else {
					logger.Get().Error().Err(err).Msgf("删除文件失败: %s", result.Path)
				}
			case internal.ModeMove:
				if err := d.moveFile(result.Path, hashStr); err == nil {
					d.stats.Moved++
					logger.Get().Debug().Msgf("已移动重复文件: %s", result.Path)
				} else {
					logger.Get().Error().Err(err).Msgf("移动文件失败: %s", result.Path)
				}
			}
		} else {
			record := &internal.FileRecord{
				Hash:      hashStr,
				FilePath:  result.Path,
				FileSize:  result.Size,
				CreatedAt: time.Now().Unix(),
			}
			if err := d.db.Insert(record); err == nil {
				d.stats.Added++
				logger.Get().Trace().Msgf("新增记录: %s", result.Path)
			}
		}

		d.stats.TotalProcessed++

		d.progressChan <- internal.ProgressUpdate{
			Processed:   d.stats.TotalProcessed,
			Added:       d.stats.Added,
			Deleted:     d.stats.Deleted,
			Moved:       d.stats.Moved,
			CurrentFile: result.Path,
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
	dstPath := filepath.Join(d.targetDir, hash[:8]+"_"+hash[8:]+ext)

	logger.Get().Debug().Msgf("移动文件: %s -> %s", srcPath, dstPath)
	return os.Rename(srcPath, dstPath)
}

func (d *Deduplicator) Progress() <-chan internal.ProgressUpdate {
	return d.progressChan
}
