package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
	"github.com/skill-home/cli/internal/skill"
)

type pushOptions struct {
	namespace string
	version   string
	force     bool
	message   string
}

func newPushCmd() *cobra.Command {
	opts := &pushOptions{}

	cmd := &cobra.Command{
		Use:   "push [path]",
		Short: "推送技能到注册中心",
		Long:  "将本地技能打包并推送到 skill-home 注册中心",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return runPush(path, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.namespace, "namespace", "n", "", "命名空间 (默认使用配置中的 default_namespace)")
	cmd.Flags().StringVarP(&opts.version, "version", "v", "", "指定版本号 (默认使用 SKILL.md 中的 version)")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "强制推送，忽略安全警告")
	cmd.Flags().StringVarP(&opts.message, "message", "m", "", "版本说明")

	return cmd
}

func runPush(path string, opts *pushOptions) error {
	// 检查登录状态
	apiKey := viper.GetString("registry.api_key")
	if apiKey == "" {
		return fmt.Errorf("未登录，请先运行 'skill-home login'")
	}

	// 解析技能
	s, err := skill.Parse(path)
	if err != nil {
		return fmt.Errorf("解析技能失败: %w", err)
	}

	fmt.Printf("推送技能: %s\n", color.CyanString(s.GetFullName()))
	fmt.Printf("版本: %s\n", color.CyanString(s.Manifest.Version))
	fmt.Println()

	// 确定命名空间
	namespace := opts.namespace
	if namespace == "" {
		namespace = s.Manifest.Namespace
	}
	if namespace == "" {
		namespace = config.C.Local.DefaultNamespace
	}

	// 构建临时包路径
	tmpDir := os.TempDir()
	packName := fmt.Sprintf("%s-%s.tar.gz", s.Manifest.Name, s.Manifest.Version)
	packPath := filepath.Join(tmpDir, packName)
	defer os.Remove(packPath)

	// 打包
	fmt.Println("正在打包技能...")
	if err := packSkill(path, packPath); err != nil {
		return fmt.Errorf("打包失败: %w", err)
	}
	fmt.Println(color.GreenString("✓"), "打包完成")

	// 创建客户端
	server := viper.GetString("registry.endpoint")
	client := registry.NewClient(server, apiKey)

	// 推送
	fmt.Println("正在推送到注册中心...")
	req := &registry.PublishRequest{
		Namespace: namespace,
		Force:     opts.force,
	}

	resp, err := client.Publish(packPath, req)
	if err != nil {
		// 处理特定错误
		if apiErr, ok := err.(*registry.APIError); ok {
			if apiErr.Code == "VERSION_EXISTS" {
				return fmt.Errorf("版本 %s 已存在，请更新版本号或使用 --force 覆盖", s.Manifest.Version)
			}
			if apiErr.Code == "VALIDATION_FAILED" {
				fmt.Println(color.RedString("✗"), "安全扫描未通过:")
				fmt.Println("  ", apiErr.Message)
				fmt.Println()
				fmt.Printf("使用 %s 强制推送 (不推荐)\n", color.YellowString("--force"))
				return nil
			}
		}
		return fmt.Errorf("推送失败: %w", err)
	}

	fmt.Println()
	fmt.Println(color.GreenString("✓"), "推送成功!")
	fmt.Printf("  技能: %s/%s@%s\n", color.CyanString(resp.Namespace), resp.Name, resp.Version)
	fmt.Printf("  下载: %s\n", color.CyanString(resp.DownloadURL))
	fmt.Printf("  时间: %s\n", resp.PublishedAt)

	// 自动同步（如果配置了）
	if config.C.Sync.AutoSyncOnPush {
		fmt.Println()
		fmt.Println("正在自动同步到本地 IDE...")
		// TODO: 调用同步逻辑
	}

	return nil
}

// packSkill 打包技能到指定路径
func packSkill(srcPath, dstPath string) error {
	// 复用 pack 命令的逻辑
	opts := &packOptions{
		output:   dstPath,
		compress: true,
	}
	return runPack(srcPath, opts)
}
