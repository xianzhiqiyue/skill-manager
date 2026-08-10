package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/ide"
	"github.com/skill-home/cli/internal/sync"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "诊断本地环境与 registry 配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

func runDoctor() error {
	fmt.Println("skill-home doctor")
	fmt.Println()

	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		fmt.Printf("%s 配置文件: 使用内置默认值\n", color.YellowString("!"))
	} else {
		fmt.Printf("%s 配置文件: %s\n", color.GreenString("✓"), configFile)
	}

	if err := config.EnsureDir(config.C.Local.SkillsDir); err != nil {
		fmt.Printf("%s 本地缓存目录: %v\n", color.RedString("✗"), err)
	} else {
		fmt.Printf("%s 本地缓存目录: %s\n", color.GreenString("✓"), config.C.Local.SkillsDir)
	}

	client := newRegistryClient()
	if err := client.HealthCheck(); err != nil {
		fmt.Printf("%s Registry 健康检查失败: %v\n", color.RedString("✗"), err)
	} else {
		fmt.Printf("%s Registry 健康检查通过: %s\n", color.GreenString("✓"), registryEndpoint())
	}

	if viper.GetString("registry.api_key") == "" {
		fmt.Printf("%s 未配置 API Key\n", color.YellowString("!"))
	} else if user, err := client.GetCurrentUser(); err != nil {
		fmt.Printf("%s API Key 校验失败: %v\n", color.RedString("✗"), err)
	} else {
		namespace := strings.TrimPrefix(strings.TrimSpace(user.Namespace), "@")
		if namespace == "" {
			namespace = strings.TrimPrefix(strings.TrimSpace(user.Username), "@")
		}
		fmt.Printf("%s 当前用户: %s（发布作用域 @%s）\n", color.GreenString("✓"), user.Username, namespace)
	}

	resolver, err := config.NewPathResolver()
	if err != nil {
		fmt.Printf("%s 项目根目录解析失败: %v\n", color.RedString("✗"), err)
	} else {
		fmt.Printf("%s 项目根目录: %s\n", color.GreenString("✓"), resolver.GetProjectRoot())
		checkIDEPath("claude", config.C.IDE.Claude.Enabled, config.C.IDE.Claude.GlobalPath, resolver)
		checkIDEPath("codex", config.C.IDE.Codex.Enabled, config.C.IDE.Codex.GlobalPath, resolver)
		checkIDEPath("openclaw", config.C.IDE.OpenClaw.Enabled, config.C.IDE.OpenClaw.GlobalPath, resolver)
		checkIDEPath("xigua", config.C.IDE.Xigua.Enabled, config.C.IDE.Xigua.GlobalPath, resolver)
	}

	fmt.Printf("%s 符号链接支持: %t\n", color.GreenString("✓"), sync.SymlinkSupported)
	return nil
}

func checkIDEPath(ideType string, enabled bool, globalPath string, resolver *config.PathResolver) {
	label := filepath.Base(ideType)
	if !enabled {
		fmt.Printf("%s %s: 已禁用\n", color.YellowString("!"), label)
		return
	}

	projectPath, err := resolver.GetIDEProjectPath(ideType)
	if err != nil {
		fmt.Printf("%s %s 项目路径: %v\n", color.RedString("✗"), label, err)
		return
	}

	if err := os.MkdirAll(projectPath, 0755); err != nil {
		fmt.Printf("%s %s 项目路径不可写: %v\n", color.RedString("✗"), label, err)
	} else {
		adapter, _ := ide.NewAdapter(ideType, projectPath)
		fmt.Printf("%s %s 项目路径: %s (symlink=%t)\n", color.GreenString("✓"), label, projectPath, adapter.SupportsSymlink())
	}

	if globalPath == "" {
		return
	}

	resolvedGlobal, err := resolver.GetIDEGlobalPath(ideType)
	if err != nil {
		fmt.Printf("%s %s 全局路径: %v\n", color.YellowString("!"), label, err)
		return
	}
	if err := os.MkdirAll(resolvedGlobal, 0755); err != nil {
		fmt.Printf("%s %s 全局路径不可写: %v\n", color.RedString("✗"), label, err)
	} else {
		fmt.Printf("%s %s 全局路径: %s\n", color.GreenString("✓"), label, resolvedGlobal)
	}
}
