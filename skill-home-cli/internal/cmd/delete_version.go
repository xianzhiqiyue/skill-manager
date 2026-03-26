package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
)

type deleteVersionOptions struct {
	yes bool
}

func newDeleteVersionCmd() *cobra.Command {
	opts := &deleteVersionOptions{}

	cmd := &cobra.Command{
		Use:   "delete-version <skill-ref>",
		Short: "删除已发布的远程技能版本",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteVersion(args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "跳过确认提示")
	return cmd
}

func runDeleteVersion(skillRef string, opts *deleteVersionOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}

	namespace, name, version, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}
	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("删除远程版本时必须指定版本，请使用 'skill-home delete-version @%s/%s@<version>'", namespace, name)
	}

	fullRef := fmt.Sprintf("@%s/%s@%s", strings.TrimPrefix(namespace, "@"), name, version)
	if !opts.yes {
		ok, err := confirmRegistryDeletion(fmt.Sprintf("确认删除远程版本 %s? [y/N]: ", fullRef))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println(color.YellowString("已取消"))
			return nil
		}
	}

	if err := newRegistryClient().DeleteVersion(namespace, name, version); err != nil {
		return fmt.Errorf("删除版本失败: %w", err)
	}

	fmt.Printf("%s 已删除 %s\n", color.GreenString("✓"), fullRef)
	return nil
}
