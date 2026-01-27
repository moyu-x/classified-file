package scanner

import (
	"os"
	"path/filepath"

	"github.com/moyu-x/classified-file/pkg/logger"
)

type FileWalker struct {
	IncludeHidden bool
}

func NewFileWalker() *FileWalker {
	return &FileWalker{
		IncludeHidden: true,
	}
}

func (w *FileWalker) Walk(root string, callback func(path string, info os.FileInfo) error) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		return callback(path, info)
	})
}

func (w *FileWalker) CountFiles(dirs []string) (int, error) {
	logger.Get().Info().Msgf("开始统计文件数量，共 %d 个目录", len(dirs))

	count := 0
	for _, dir := range dirs {
		logger.Get().Debug().Msgf("扫描目录: %s", dir)
		err := w.Walk(dir, func(path string, info os.FileInfo) error {
			count++
			return nil
		})
		if err != nil {
			logger.Get().Error().Err(err).Msgf("扫描目录失败: %s", dir)
			return 0, err
		}
	}

	logger.Get().Info().Msgf("文件统计完成，共找到 %d 个文件", count)
	return count, nil
}
