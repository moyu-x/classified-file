package cmd

import (
	"os"

	"github.com/moyu-x/classified-file/config"
	"github.com/moyu-x/classified-file/database"
	"github.com/moyu-x/classified-file/logger"
	"github.com/moyu-x/classified-file/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "classified-file",
	Short: "文件分类去重工具",
	Long: `一个高效的文件分类和去重命令行工具。
支持使用 xxHash 计算文件哈希，并将结果存储在 SQLite 数据库中。
提供交互式 TUI 界面，支持删除或移动重复文件。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if err := logger.Init(cfg.Logging.Level, cfg.Logging.File); err != nil {
			return err
		}

		logger.Get().Info().Msg("加载配置完成")
		logger.Get().Info().Msgf("数据库路径: %s", cfg.Database.Path)
		logger.Get().Info().Msgf("日志级别: %s", cfg.Logging.Level)
		if cfg.Logging.File != "" {
			logger.Get().Info().Msgf("日志文件: %s", cfg.Logging.File)
		}

		db, err := database.NewDatabase(cfg.Database.Path)
		if err != nil {
			return err
		}
		defer db.Close()

		tuiConfig := &tui.Config{
			DatabasePath: cfg.Database.Path,
		}
		return tui.Run(tuiConfig)
	},
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "启动 TUI 界面",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if err := logger.Init(cfg.Logging.Level, cfg.Logging.File); err != nil {
			return err
		}

		logger.Get().Info().Msg("加载配置完成")
		logger.Get().Info().Msgf("数据库路径: %s", cfg.Database.Path)
		logger.Get().Info().Msgf("日志级别: %s", cfg.Logging.Level)
		if cfg.Logging.File != "" {
			logger.Get().Info().Msgf("日志文件: %s", cfg.Logging.File)
		}

		db, err := database.NewDatabase(cfg.Database.Path)
		if err != nil {
			return err
		}
		defer db.Close()

		tuiConfig := &tui.Config{
			DatabasePath: cfg.Database.Path,
		}
		return tui.Run(tuiConfig)
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

performance:
  workers: 6

ui:
  default_mode: "delete"
  default_target_dir: ""

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
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(initCmd)
}
