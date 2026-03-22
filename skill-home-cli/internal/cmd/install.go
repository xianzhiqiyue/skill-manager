package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/skill"
)

type installOptions struct {
	ide      string
	mode     string
	global   bool
	force    bool
	skipScan bool
}

func newInstallCmd() *cobra.Command {
	opts := &installOptions{}

	cmd := &cobra.Command{
		Use:   "install <skill-ref>",
		Short: "安装技能到本地并同步到 IDE",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.ide, "ide", "", "指定 IDE (claude/copilot/cursor/codex)")
	cmd.Flags().StringVar(&opts.mode, "mode", "", "同步模式 (auto/symlink/mirror)")
	cmd.Flags().BoolVar(&opts.global, "global", false, "同步到全局配置而非项目配置")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "重新下载并在扫描失败时继续安装")
	cmd.Flags().BoolVar(&opts.skipScan, "skip-scan", false, "跳过安装前安全扫描")

	return cmd
}

func runInstall(skillRef string, opts *installOptions) error {
	pulled, err := pullSkillRef(skillRef, &pullOptions{
		extract: true,
		force:   opts.force,
	})
	if err != nil {
		return err
	}

	if !opts.skipScan && config.C.Security.ScanOnInstall {
		if err := scanInstallTarget(pulled.OutputDir, opts.force); err != nil {
			return err
		}
	}

	s, err := skill.Parse(pulled.OutputDir)
	if err != nil {
		return fmt.Errorf("解析已下载技能失败: %w", err)
	}

	fmt.Println()
	fmt.Println("正在安装到 IDE...")
	if err := syncParsedSkill(s, &syncOptions{
		ide:    opts.ide,
		mode:   opts.mode,
		global: opts.global,
	}); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("%s 已安装 %s/%s@%s\n", color.GreenString("✓"), "@"+pulled.Namespace, pulled.Name, pulled.Version)
	return nil
}

func scanInstallTarget(path string, force bool) error {
	result, err := scanSkillPath(path)
	if err != nil {
		return err
	}
	if len(result.Issues) > 0 {
		fmt.Printf("安装前安全扫描: %s\n", result.Summary)
	}
	if err := evaluateScanResult(result, false, true); err != nil {
		if force {
			fmt.Printf("%s 检测到高风险问题，但由于指定了 --force，继续安装\n", color.YellowString("!"))
			return nil
		}
		return err
	}
	return nil
}
