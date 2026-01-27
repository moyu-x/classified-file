package cmd

import (
	"fmt"

	"github.com/moyu-x/classified-file/classifier"
	"github.com/moyu-x/classified-file/config"
	"github.com/moyu-x/classified-file/logger"
	"github.com/spf13/cobra"
)

var classifyCmd = &cobra.Command{
	Use:   "classify <directories...> <destination>",
	Short: "按文件类型分类文件",
	Long: `遍历指定目录中的所有文件，使用 filetype 判断文件类型，并将文件归类写入目标目录。
每种文件类型一个文件夹，每个类型中每 500 个文件作为一个目录。
文件名重复时自动重命名（添加自增序列）。`,
	Args: cobra.MinimumNArgs(2),
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

		sourceDirs := args[:len(args)-1]
		destDir := args[len(args)-1]

		logger.Get().Info().Msgf("源目录数: %d", len(sourceDirs))
		for i, dir := range sourceDirs {
			logger.Get().Info().Msgf("  [%d] %s", i+1, dir)
		}
		logger.Get().Info().Msgf("目标目录: %s", destDir)

		filesPerDir, _ := cmd.Flags().GetInt("files-per-dir")
		cls := classifier.NewClassifierWithCustomFilesPerDir(filesPerDir)

		logger.Get().Info().Msgf("每目录文件数: %d", filesPerDir)

		stats, err := cls.Classify(sourceDirs, destDir)
		if err != nil {
			return fmt.Errorf("文件分类失败: %w", err)
		}

		fmt.Println(stats.String())

		return nil
	},
}

func init() {
	classifyCmd.Flags().Int("files-per-dir", 500, "每个目录的文件数（默认: 500）")

	rootCmd.AddCommand(classifyCmd)
}
