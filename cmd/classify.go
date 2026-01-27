package cmd

import (
	"fmt"

	"github.com/moyu-x/classified-file/app"
	"github.com/moyu-x/classified-file/config"
	"github.com/spf13/cobra"
)

var classifyCmd = &cobra.Command{
	Use:   "classify <directories...> <destination>",
	Short: "按文件类型分类文件",
	Long: `遍历指定目录中的所有文件，使用 filetype 判断文件类型，并将文件归类写入目标目录。
每种文件类型一个文件夹，每个类型中每 500 个文件作为一个目录。
文件名重复时自动重命名（添加自增序列）。`,
	Args: cobra.MinimumNArgs(2),
	RunE: runClassify,
}

func runClassify(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	sourceDirs := args[:len(args)-1]
	destDir := args[len(args)-1]

	filesPerDir, _ := cmd.Flags().GetInt("files-per-dir")
	verbose, _ := cmd.Flags().GetBool("verbose")

	opts := &app.ClassifyOptions{
		SourceDirs:  sourceDirs,
		DestDir:     destDir,
		FilesPerDir: filesPerDir,
		Verbose:     verbose,
		LogLevel:    cfg.Logging.Level,
		LogFile:     cfg.Logging.File,
	}

	stats, err := app.RunClassify(opts)
	if err != nil {
		return err
	}

	fmt.Println(stats.String())

	return nil
}

func init() {
	classifyCmd.Flags().Int("files-per-dir", 500, "每个目录的文件数（默认: 500）")
	classifyCmd.Flags().Bool("verbose", false, "显示详细日志")

	rootCmd.AddCommand(classifyCmd)
}
