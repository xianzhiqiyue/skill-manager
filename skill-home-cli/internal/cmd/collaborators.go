package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
)

type collaboratorsOptions struct {
	format string
	role   string
	yes    bool
}

func newCollaboratorsCmd() *cobra.Command {
	opts := &collaboratorsOptions{}

	cmd := &cobra.Command{
		Use:   "collaborators <skill-ref>",
		Short: "管理远程 skill 协作者",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListCollaborators(args[0], opts)
		},
	}
	cmd.Flags().StringVarP(&opts.format, "format", "f", "table", "输出格式 (table/json)")

	addCmd := &cobra.Command{
		Use:   "add <skill-ref> <username>",
		Short: "新增或更新远程 skill 协作者",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddCollaborator(args[0], args[1], opts)
		},
	}
	addCmd.Flags().StringVar(&opts.role, "role", "maintainer", "协作者角色 (maintainer/viewer)")

	removeCmd := &cobra.Command{
		Use:     "remove <skill-ref> <username>",
		Aliases: []string{"rm", "delete"},
		Short:   "移除远程 skill 协作者",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemoveCollaborator(args[0], args[1], opts)
		},
	}
	removeCmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "跳过确认提示")

	cmd.AddCommand(addCmd)
	cmd.AddCommand(removeCmd)
	return cmd
}

func runListCollaborators(skillRef string, opts *collaboratorsOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}

	namespace, name, version, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}
	if strings.TrimSpace(version) != "" {
		return fmt.Errorf("协作者作用于整个 skill，请使用 '@%s/%s'，不要指定版本", strings.TrimPrefix(namespace, "@"), name)
	}

	collaborators, err := newRegistryClient().ListCollaborators(namespace, name)
	if err != nil {
		return fmt.Errorf("获取协作者失败: %w", err)
	}
	if opts.format == "json" {
		return printJSON(collaborators)
	}
	if len(collaborators) == 0 {
		fmt.Printf("@%s/%s 暂无协作者\n", strings.TrimPrefix(namespace, "@"), name)
		return nil
	}

	fmt.Printf("@%s/%s 协作者\n\n", strings.TrimPrefix(namespace, "@"), name)
	for _, collaborator := range collaborators {
		display := collaborator.Username
		if collaborator.DisplayNameZh != "" {
			display = fmt.Sprintf("%s / @%s", collaborator.DisplayNameZh, collaborator.Username)
		} else {
			display = "@" + collaborator.Username
		}
		fmt.Printf("%s  %-10s  %s\n", color.GreenString("•"), collaborator.Role, display)
	}
	return nil
}

func runAddCollaborator(skillRef, username string, opts *collaboratorsOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}
	role, err := normalizeCollaboratorRoleFlag(opts.role)
	if err != nil {
		return err
	}

	namespace, name, version, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}
	if strings.TrimSpace(version) != "" {
		return fmt.Errorf("协作者作用于整个 skill，请使用 '@%s/%s'，不要指定版本", strings.TrimPrefix(namespace, "@"), name)
	}

	collaborator, err := newRegistryClient().UpsertCollaborator(namespace, name, &registry.UpsertCollaboratorRequest{
		Username: strings.TrimPrefix(strings.TrimSpace(username), "@"),
		Role:     role,
	})
	if err != nil {
		return fmt.Errorf("保存协作者失败: %w", err)
	}

	fmt.Printf("%s 已设置 @%s/%s 的协作者 @%s 为 %s\n",
		color.GreenString("✓"),
		strings.TrimPrefix(namespace, "@"),
		name,
		collaborator.Username,
		collaborator.Role,
	)
	return nil
}

func runRemoveCollaborator(skillRef, username string, opts *collaboratorsOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}

	namespace, name, version, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}
	if strings.TrimSpace(version) != "" {
		return fmt.Errorf("协作者作用于整个 skill，请使用 '@%s/%s'，不要指定版本", strings.TrimPrefix(namespace, "@"), name)
	}

	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if username == "" {
		return fmt.Errorf("username is required")
	}

	fullName := fmt.Sprintf("@%s/%s", strings.TrimPrefix(namespace, "@"), name)
	if !opts.yes {
		ok, err := confirmRegistryDeletion(fmt.Sprintf("确认从 %s 移除协作者 @%s? [y/N]: ", fullName, username))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println(color.YellowString("已取消"))
			return nil
		}
	}

	if err := newRegistryClient().DeleteCollaborator(namespace, name, username); err != nil {
		return fmt.Errorf("移除协作者失败: %w", err)
	}

	fmt.Printf("%s 已从 %s 移除协作者 @%s\n", color.GreenString("✓"), fullName, username)
	return nil
}

func normalizeCollaboratorRoleFlag(role string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return "maintainer", nil
	}
	switch role {
	case "maintainer", "viewer":
		return role, nil
	default:
		return "", fmt.Errorf("role must be maintainer or viewer")
	}
}
