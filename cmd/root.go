package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "classified-file",
	Short: "文件分类去重工具",
	Long: `一个高效的文件分类和去重命令行工具。
支持使用 xxHash 计算文件哈希，并将结果存储在 SQLite 数据库中。
支持删除或移动重复文件。`,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化配置文件",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
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
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
}
