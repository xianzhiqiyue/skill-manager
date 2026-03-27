package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
)

type deleteOptions struct {
	yes bool
}

func newDeleteCmd() *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <skill-ref>",
		Short: "删除已发布的远程技能",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "跳过确认提示")
	return cmd
}

func runDelete(skillRef string, opts *deleteOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}

	namespace, name, version, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}
	if version != "" {
		return fmt.Errorf("删除整个技能时不能指定版本，请改用 'skill-home delete-version @%s/%s@%s'", namespace, name, version)
	}

	fullName := fmt.Sprintf("@%s/%s", strings.TrimPrefix(namespace, "@"), name)
	if !opts.yes {
		ok, err := confirmRegistryDeletion(fmt.Sprintf("确认删除远程技能 %s? [y/N]: ", fullName))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println(color.YellowString("已取消"))
			return nil
		}
	}

	if err := newRegistryClient().DeleteSkill(namespace, name); err != nil {
		return fmt.Errorf("删除技能失败: %w", err)
	}

	fmt.Printf("%s 已删除 %s\n", color.GreenString("✓"), fullName)
	return nil
}

func confirmRegistryDeletion(prompt string) (bool, error) {
	fmt.Print(prompt)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	answer := strings.ToLower(strings.TrimSpace(input))
	return answer == "y" || answer == "yes", nil
}
