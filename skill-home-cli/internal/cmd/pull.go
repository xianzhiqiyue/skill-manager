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
	"github.com/skill-home/cli/pkg/archive"
)

type pullOptions struct {
	outputDir string
	extract   bool
	force     bool
}

func newPullCmd() *cobra.Command {
	opts := &pullOptions{}

	cmd := &cobra.Command{
		Use:   "pull <skill-ref>",
		Short: "从注册中心拉取技能",
		Long: `从注册中心拉取技能到本地。

技能引用格式:
  skill-home pull my-skill              # 拉取最新版本
  skill-home pull @user/my-skill        # 指定命名空间
  skill-home pull my-skill@1.0.0        # 指定版本`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.outputDir, "output", "o", "", "输出目录 (默认使用缓存目录)")
	cmd.Flags().BoolVarP(&opts.extract, "extract", "x", true, "自动解压")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "覆盖已存在目录并重新下载")

	return cmd
}

func runPull(skillRef string, opts *pullOptions) error {
	// 检查登录状态（可选，因为下载可以是公开的）
	apiKey := viper.GetString("registry.api_key")
	server := viper.GetString("registry.endpoint")
	if server == "" {
		server = "https://registry.skill-home.dev"
	}

	// 解析技能引用
	namespace, name, version, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}

	fmt.Printf("拉取技能: %s/%s\n", color.CyanString("@"+namespace), name)
	if version != "" {
		fmt.Printf("版本: %s\n", color.CyanString(version))
	} else {
		fmt.Println("版本: 最新版")
	}
	fmt.Println()

	// 创建客户端
	client := registry.NewClient(server, apiKey)

	// 如果没有指定版本，获取最新版本
	if version == "" {
		skill, err := client.GetSkill(namespace, name)
		if err != nil {
			return fmt.Errorf("获取技能信息失败: %w", err)
		}
		version = skill.LatestVersion
		if version == "" {
			return fmt.Errorf("技能没有可用版本")
		}
		fmt.Printf("最新版本: %s\n", color.CyanString(version))
	}

	// 确定输出目录
	outputDir := opts.outputDir
	if outputDir == "" {
		outputDir = config.GetSkillCacheDir(namespace, name, version)
	}

	// 检查是否已存在
	if _, err := os.Stat(outputDir); err == nil {
		if opts.force {
			if err := removeExistingOutput(outputDir); err != nil {
				return err
			}
		} else {
			fmt.Printf("技能已存在于: %s\n", color.YellowString(outputDir))
			fmt.Println("使用 --force 可以重新下载")
			return nil
		}
	}

	// 创建临时下载文件
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s-%s.tar.gz", namespace, name, version))
	defer os.Remove(tmpFile)

	// 下载
	fmt.Println("正在下载...")
	if err := client.Download(namespace, name, version, tmpFile); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	fmt.Println(color.GreenString("✓"), "下载完成")

	// 解压
	if opts.extract {
		fmt.Println("正在解压...")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}

		if err := archive.ExtractTarGz(tmpFile, outputDir); err != nil {
			return fmt.Errorf("解压失败: %w", err)
		}
		fmt.Println(color.GreenString("✓"), "解压完成")
	}

	fmt.Println()
	fmt.Println(color.GreenString("✓"), "拉取成功!")
	fmt.Printf("  位置: %s\n", color.CyanString(outputDir))

	// 提示同步
	fmt.Println()
	fmt.Printf("运行 '%s' 同步到 IDE\n", color.YellowString(fmt.Sprintf("skill-home sync %s", outputDir)))

	return nil
}

func removeExistingOutput(path string) error {
	cleanPath := filepath.Clean(path)
	rootPath := filepath.VolumeName(cleanPath) + string(filepath.Separator)
	if cleanPath == "" || cleanPath == "." || cleanPath == string(filepath.Separator) || cleanPath == rootPath {
		return fmt.Errorf("拒绝删除不安全路径: %s", path)
	}
	if err := os.RemoveAll(cleanPath); err != nil {
		return fmt.Errorf("清理已存在目录失败: %w", err)
	}
	return nil
}
