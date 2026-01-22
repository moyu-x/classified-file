package cmd

import (
	"fmt"
	"os"

	"github.com/moyu-x/classified-file/config"
	"github.com/moyu-x/classified-file/database"
	"github.com/moyu-x/classified-file/deduplicator"
	"github.com/moyu-x/classified-file/internal"
	"github.com/moyu-x/classified-file/logger"
	"github.com/moyu-x/classified-file/scanner"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	dryRun  bool
)

var rootCmd = &cobra.Command{
	Use:   "classified-file <directories...>",
	Short: "文件分类去重工具",
	Long: `一个高效的文件分类和去重命令行工具。
支持使用 xxHash 计算文件哈希，并将结果存储在 SQLite 数据库中。
支持删除或移动重复文件。`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		logLevel := cfg.Logging.Level
		if verbose {
			logLevel = "debug"
		}
		if err := logger.Init(logLevel, cfg.Logging.File); err != nil {
			return err
		}

		logger.Get().Info().Msg("加载配置完成")
		logger.Get().Info().Msgf("数据库路径: %s", cfg.Database.Path)

		db, err := database.NewDatabase(cfg.Database.Path)
		if err != nil {
			return err
		}
		defer db.Close()

		modeStr, _ := cmd.Flags().GetString("mode")
		targetDir, _ := cmd.Flags().GetString("target-dir")

		if modeStr == "move" && targetDir == "" {
			return fmt.Errorf("使用 move 模式时必须指定 --target-dir")
		}

		logger.Get().Info().Msgf("操作模式: %s", modeStr)
		if targetDir != "" {
			logger.Get().Info().Msgf("目标目录: %s", targetDir)
		}

		walker := scanner.NewFileWalker()
		totalFiles, err := walker.CountFiles(args)
		if err != nil {
			return fmt.Errorf("统计文件数量失败: %w", err)
		}

		logger.Get().Info().Msgf("文件统计完成，共找到 %d 个文件", totalFiles)

		if dryRun {
			logger.Get().Info().Msg("=== 预览模式，不会实际修改文件 ===")
		}

		dedup := deduplicator.NewDeduplicator(db, internal.OperationMode(modeStr), targetDir, totalFiles, verbose)

		stats, err := dedup.Process(args)
		if err != nil {
			return err
		}

		printFinalStats(stats, args)

		return nil
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化配置文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		configDir := homeDir + "/.classified-file"
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}

		configPath := configDir + "/config.yaml"
		defaultConfig := `# 文件分类去重工具配置文件

database:
  path: "~/.classified-file/hashes.db"

scanner:
  follow_symlinks: false

logging:
  level: "info"
  file: ""
`
		return os.WriteFile(configPath, []byte(defaultConfig), 0644)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringP("mode", "m", "delete", "操作模式: delete 或 move")
	rootCmd.Flags().StringP("target-dir", "t", "", "移动模式的目标目录")
	rootCmd.Flags().String("db", "", "数据库路径")
	rootCmd.Flags().String("log-level", "info", "日志级别")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "显示哈希值（默认显示文件详情）")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "预览模式，不实际修改文件")

	rootCmd.AddCommand(initCmd)
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

func printFinalStats(stats *internal.ProcessStats, dirs []string) {
	elapsed := stats.EndTime.Sub(stats.StartTime)

	logger.Get().Info().Msg("========== 处理完成 ==========")
	logger.Get().Info().Msgf("扫描目录数: %d", len(dirs))
	for i, dir := range dirs {
		logger.Get().Info().Msgf("  [%d] %s", i+1, dir)
	}
	logger.Get().Info().Msgf("总文件数: %d", stats.TotalProcessed)
	logger.Get().Info().Msgf("新增记录: %d 个文件", stats.Added)
	logger.Get().Info().Msgf("重复文件: %d 个文件", stats.Deleted+stats.Moved)
	logger.Get().Info().Msgf("  - 已删除: %d 个", stats.Deleted)
	logger.Get().Info().Msgf("  - 已移动: %d 个", stats.Moved)
	logger.Get().Info().Msgf("释放空间: %s", formatBytes(stats.FreedSpace))
	logger.Get().Info().Msgf("总耗时: %v", elapsed)
	logger.Get().Info().Msg("============================")
}
