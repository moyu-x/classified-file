package app

import (
	"fmt"

	"github.com/moyu-x/classified-file/pkg/config"
	"github.com/moyu-x/classified-file/pkg/database"
	"github.com/moyu-x/classified-file/pkg/deduplicator"
	"github.com/moyu-x/classified-file/internal"
	"github.com/moyu-x/classified-file/pkg/logger"
)

type DedupOptions struct {
	SourceDirs []string
	Mode       string
	TargetDir  string
	DBPath     string
	LogLevel   string
	LogFile    string
	Verbose    bool
	DryRun     bool
	Resume     bool
	Reset      bool
}

func RunDedup(opts *DedupOptions) (*internal.ProcessStats, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logLevel := cfg.Logging.Level
	if opts.Verbose {
		logLevel = "debug"
	}

	if err := logger.Init(logLevel, cfg.Logging.File); err != nil {
		return nil, err
	}

	logger.Get().Info().Msg("加载配置完成")
	logger.Get().Info().Msgf("数据库路径: %s", cfg.Database.Path)

	db, err := database.NewDatabase(cfg.Database.Path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if opts.Mode == "move" && opts.TargetDir == "" {
		return nil, fmt.Errorf("使用 move 模式时必须指定 --target-dir")
	}

	logger.Get().Info().Msgf("操作模式: %s", opts.Mode)
	if opts.TargetDir != "" {
		logger.Get().Info().Msgf("目标目录: %s", opts.TargetDir)
	}
	logger.Get().Info().Msgf("恢复模式: %v", opts.Resume)
	logger.Get().Info().Msgf("重置模式: %v", opts.Reset)

	if opts.DryRun {
		logger.Get().Info().Msg("=== 预览模式，不会实际修改文件 ===")
	}

	dedup := deduplicator.NewDeduplicator(db, internal.OperationMode(opts.Mode), opts.TargetDir, opts.Verbose)

	stats, err := dedup.Process(opts.SourceDirs, opts.Resume, opts.Reset)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
