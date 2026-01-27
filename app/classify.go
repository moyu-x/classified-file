package app

import (
	"fmt"

	"github.com/moyu-x/classified-file/classifier"
	"github.com/moyu-x/classified-file/logger"
)

type ClassifyOptions struct {
	SourceDirs  []string
	DestDir     string
	FilesPerDir int
	Verbose     bool
	LogLevel    string
	LogFile     string
}

func RunClassify(opts *ClassifyOptions) (*classifier.ClassifierStats, error) {
	logLevel := opts.LogLevel
	if opts.Verbose {
		logLevel = "debug"
	}

	if err := logger.Init(logLevel, opts.LogFile); err != nil {
		return nil, err
	}

	logger.Get().Info().Msg("加载配置完成")

	logger.Get().Info().Msgf("源目录数: %d", len(opts.SourceDirs))
	for i, dir := range opts.SourceDirs {
		logger.Get().Info().Msgf("  [%d] %s", i+1, dir)
	}
	logger.Get().Info().Msgf("目标目录: %s", opts.DestDir)

	cls := classifier.NewClassifierWithCustomFilesPerDir(opts.FilesPerDir)
	logger.Get().Info().Msgf("每目录文件数: %d", opts.FilesPerDir)

	stats, err := cls.Classify(opts.SourceDirs, opts.DestDir)
	if err != nil {
		return nil, fmt.Errorf("文件分类失败: %w", err)
	}

	return stats, nil
}
