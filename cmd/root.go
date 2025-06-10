package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "classified-file",
	Short: "一个用于去重和按类型整理文件的工具",
	Long: `Classified File 是一个命令行工具，用于高效处理、去重和按类型整理文件。

主要功能:
- 读取源目录中的所有文件
- 高效计算每个文件的MD5哈希值
- 将哈希存储在SQLite数据库和内存缓存中
- 基于内容哈希检测和删除重复文件
- 按MIME类型对文件进行分类
- 以结构化的目录层次组织文件`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.classified-file.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "帮助消息")
}
