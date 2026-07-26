package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
)

type feedbackOptions struct {
	feedbackType string
	message      string
}

func newFeedbackCmd() *cobra.Command {
	opts := &feedbackOptions{}
	cmd := &cobra.Command{
		Use:   "feedback <skill-ref>",
		Short: "提交 Skill 使用反馈",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFeedback(args[0], opts)
		},
	}
	cmd.Flags().StringVarP(&opts.feedbackType, "type", "t", "", "反馈类型 (useful/issue/suggestion)")
	cmd.Flags().StringVarP(&opts.message, "message", "m", "", "一句话反馈内容")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("message")
	return cmd
}

func runFeedback(skillRef string, opts *feedbackOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}
	feedbackType := strings.ToLower(strings.TrimSpace(opts.feedbackType))
	if feedbackType != "useful" && feedbackType != "issue" && feedbackType != "suggestion" {
		return fmt.Errorf("反馈类型必须是 useful、issue 或 suggestion")
	}
	message := strings.TrimSpace(opts.message)
	if message == "" {
		return fmt.Errorf("反馈内容不能为空")
	}
	if len([]rune(message)) > 1000 {
		return fmt.Errorf("反馈内容不能超过 1000 个字符")
	}

	namespace, name, _, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}
	response, err := newRegistryClient().CreateSkillFeedback(namespace, name, &registry.CreateSkillFeedbackRequest{
		FeedbackType: feedbackType,
		Content:      message,
	})
	if err != nil {
		return fmt.Errorf("提交反馈失败: %w", err)
	}

	fmt.Printf("%s 已向 @%s/%s 提交 %s 反馈（状态：%s）\n",
		color.GreenString("✓"), namespace, name, response.FeedbackType, response.Status)
	return nil
}
