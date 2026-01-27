package cmd

import (
	"fmt"

	"github.com/moyu-x/classified-file/app"
	"github.com/moyu-x/classified-file/config"
	"github.com/moyu-x/classified-file/deduplicator"
	"github.com/moyu-x/classified-file/internal"
	"github.com/moyu-x/classified-file/logger"
	"github.com/spf13/cobra"
)

var dedupCmd = &cobra.Command{
	Use:   "dedup <directories...>",
	Short: "检测并删除/移动重复文件",
	Long: `遍历指定目录中的所有文件，使用 xxHash 计算哈希值并检测重复文件。
重复文件将被删除或移动到指定目录，哈希值存储在 SQLite 数据库中。`,
	Args: cobra.MinimumNArgs(1),
	RunE: runDedup,
}

func runDedup(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	modeStr, _ := cmd.Flags().GetString("mode")
	targetDir, _ := cmd.Flags().GetString("target-dir")
	verbose, _ := cmd.Flags().GetBool("verbose")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	resume, _ := cmd.Flags().GetBool("resume")
	reset, _ := cmd.Flags().GetBool("reset")

	opts := &app.DedupOptions{
		SourceDirs: args,
		Mode:       modeStr,
		TargetDir:  targetDir,
		Verbose:    verbose,
		DryRun:     dryRun,
		Resume:     resume,
		Reset:      reset,
		LogLevel:   cfg.Logging.Level,
		LogFile:    cfg.Logging.File,
	}

	stats, err := app.RunDedup(opts)
	if err != nil {
		return err
	}

	printFinalStats(stats, args)

	return nil
}

func init() {
	deduplicator.SetupSignalHandler()

	dedupCmd.Flags().StringP("mode", "m", "delete", "操作模式: delete 或 move")
	dedupCmd.Flags().StringP("target-dir", "t", "", "移动模式的目标目录")
	dedupCmd.Flags().String("db", "", "数据库路径")
	dedupCmd.Flags().String("log-level", "info", "日志级别")
	dedupCmd.Flags().BoolP("verbose", "v", false, "显示哈希值（默认显示文件详情）")
	dedupCmd.Flags().Bool("dry-run", false, "预览模式，不实际修改文件")
	dedupCmd.Flags().BoolP("resume", "r", false, "恢复模式：跳过已扫描的文件")
	dedupCmd.Flags().BoolP("reset", "R", false, "重置模式：清除进度文件，重新扫描")

	rootCmd.AddCommand(dedupCmd)
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
