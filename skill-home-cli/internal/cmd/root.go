package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/logger"
)

type RootOptions struct {
	ConfigFile string
	Verbose    bool
	Debug      bool
}

func NewRootCmd(version, commit, buildDate string) *cobra.Command {
	opts := &RootOptions{}

	rootCmd := &cobra.Command{
		Use:   "skill-home",
		Short: "AI Skill 跨平台管理工具",
		Long: color.CyanString(`
   _____ _    _       _   _
  / ____| |  | |     | | | |
 | (___ | | _| | ___ | |_| |__   ___  _ __ ___
  \___ \| |/ / |/ _ \| __| '_ \ / _ \| '_ ` + "`" + ` _ \
  ____) |   <| | (_) | |_| | | | (_) | | | | | |
 |_____/|_|\_\_|\___/ \__|_| |_|\___/|_| |_| |_|

`)+ `skill-home 是一个跨平台的 AI 技能管理工具，支持 Claude Code、Cursor、
Codex 等多个 IDE，实现技能的"一次编写，到处同步"。

使用 "skill-home [command] --help" 查看具体命令的帮助信息。`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// 初始化配置
			if err := config.Init(opts.ConfigFile); err != nil {
				return fmt.Errorf("初始化配置失败: %w", err)
			}

			// 设置日志级别
			if opts.Debug {
				logger.SetDebug()
			}

			logger.Debug("Debug mode enabled")
			logger.Debug("Config file: %s", viper.ConfigFileUsed())

			return nil
		},
	}

	// 全局 flags
	rootCmd.PersistentFlags().StringVarP(&opts.ConfigFile, "config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "显示详细信息")
	rootCmd.PersistentFlags().BoolVar(&opts.Debug, "debug", false, "启用调试模式")

	// 绑定 viper
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	// 添加子命令
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newPackCmd())
	rootCmd.AddCommand(newSyncCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newVersionCmd(version, commit, buildDate))
	rootCmd.AddCommand(newScanCmd())

	// 注册中心相关命令
	rootCmd.AddCommand(newLoginCmd())
	rootCmd.AddCommand(newLogoutCmd())
	rootCmd.AddCommand(newWhoamiCmd())
	rootCmd.AddCommand(newPushCmd())
	rootCmd.AddCommand(newPullCmd())
	rootCmd.AddCommand(newSearchCmd())

	// 技能创建和导入命令
	rootCmd.AddCommand(newCreateCmd())
	rootCmd.AddCommand(newImportCmd())

	return rootCmd
}
