package cmd

import (
	"fmt"
	"time"

	"github.com/moyu-x/classified-file/internal/database"
	"github.com/moyu-x/classified-file/internal/fileprocessor"
	"github.com/moyu-x/classified-file/internal/logger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	// 命令行参数
	dbPath        string // 数据库路径
	sourceDir     string // 源目录
	targetDir     string // 目标目录
	duplicatesDir string // 重复文件目录
	debugMode     bool   // 调试模式
)

// runCmd 代表 run 命令
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "处理源目录中的文件并整理到目标目录",
	Long: `处理源目录中的文件并整理到目标目录:
1. 读取源目录中的所有文件
2. 计算每个文件的XXHash哈希值
3. 与已有记录对比哈希值，将重复文件移动到独立目录
4. 按文件类型分类并存储到目标目录`,
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化日志
		logger.Init(debugMode)

		// 验证必要的参数
		if sourceDir == "" || targetDir == "" || duplicatesDir == "" {
			logger.Fatal().Msg("错误: 必须提供源目录、目标目录和重复文件目录")
			return
		}

		startTime := time.Now()

		// 确保目标目录存在
		fs := afero.NewOsFs()
		if err := fs.MkdirAll(targetDir, 0755); err != nil {
			logger.Fatal().Err(err).Str("path", targetDir).Msg("创建目标目录失败")
			return
		}

		// 确保重复文件目录存在
		if err := fs.MkdirAll(duplicatesDir, 0755); err != nil {
			logger.Fatal().Err(err).Str("path", duplicatesDir).Msg("创建重复文件目录失败")
			return
		}

		// 初始化数据库
		logger.Info().Msg("正在初始化数据库...")
		db, err := database.New(dbPath)
		if err != nil {
			logger.Fatal().Err(err).Str("db_path", dbPath).Msg("初始化数据库失败")
			return
		}
		defer db.Close()

		// 创建文件处理器
		logger.Info().Msg("正在从数据库加载文件哈希...")
		processor, err := fileprocessor.New(sourceDir, targetDir, db, duplicatesDir)
		if err != nil {
			logger.Fatal().Err(err).Msg("创建文件处理器失败")
			return
		}

		logger.Info().Int("count", len(processor.FileHashes)).Msg("已加载文件哈希")

		// 统计源目录中的文件总数
		logger.Info().Msg("正在扫描源目录...")
		if err := processor.CountTotalFiles(); err != nil {
			logger.Fatal().Err(err).Str("source_dir", sourceDir).Msg("统计文件数量失败")
			return
		}
		logger.Info().Int("count", processor.Stats.TotalFiles).Msg("找到待处理文件")

		// 处理文件
		logger.Info().Msg("开始处理文件...")
		if err := processor.ProcessFiles(); err != nil {
			logger.Fatal().Err(err).Msg("处理文件失败")
			return
		}

		// 报告最终统计信息
		duration := time.Since(startTime).Round(time.Second)
		logger.Info().
			Dur("duration", duration).
			Int("total_files", processor.Stats.TotalFiles).
			Int("processed", processor.Stats.ProcessedFiles).
			Int("duplicates", processor.Stats.Duplicates).
			Int("errors", processor.Stats.Errors).
			Msg("处理完成")

		// 提示重复文件位置
		if processor.Stats.Duplicates > 0 {
			logger.Info().
				Int("duplicates", processor.Stats.Duplicates).
				Str("duplicates_dir", processor.DuplicatesDir).
				Msg("重复文件已移动到独立目录，可以手动检查和删除")
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// 添加命令行参数
	runCmd.Flags().StringVarP(&sourceDir, "source", "s", "", "源目录路径 (必需)")
	runCmd.Flags().StringVarP(&targetDir, "target", "t", "", "目标目录路径 (必需)")
	runCmd.Flags().StringVarP(&duplicatesDir, "duplicates", "p", "", "重复文件存放目录 (必需)")
	runCmd.Flags().StringVarP(&dbPath, "db", "d", "./file_hashes.db", "SQLite数据库文件路径")
	runCmd.Flags().BoolVarP(&debugMode, "debug", "v", false, "启用调试模式")

	// 标记必需的参数
	if err := runCmd.MarkFlagRequired("source"); err != nil {
		fmt.Println("源文件夹目录需要给出")
		return
	}

	if err := runCmd.MarkFlagRequired("target"); err != nil {
		fmt.Println("目标文件夹目录需要给出")
		return
	}

	if err := runCmd.MarkFlagRequired("duplicates"); err != nil {
		fmt.Println("重复文件存放目录需要给出")
		return
	}
}
