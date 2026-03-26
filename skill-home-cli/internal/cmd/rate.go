package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
)

type rateOptions struct {
	score   int
	comment string
}

func newRateCmd() *cobra.Command {
	opts := &rateOptions{}

	cmd := &cobra.Command{
		Use:   "rate <skill-ref>",
		Short: "为技能评分",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRate(args[0], opts)
		},
	}

	cmd.Flags().IntVarP(&opts.score, "score", "s", 0, "评分 (1-5)")
	cmd.Flags().StringVarP(&opts.comment, "comment", "m", "", "评论")
	cmd.MarkFlagRequired("score")

	return cmd
}

func runRate(skillRef string, opts *rateOptions) error {
	if err := requireRegistryLogin(); err != nil {
		return err
	}
	if opts.score < 1 || opts.score > 5 {
		return fmt.Errorf("评分必须在 1 到 5 之间")
	}

	namespace, name, _, err := config.ParseSkillRef(skillRef)
	if err != nil {
		return err
	}

	resp, err := newRegistryClient().RateSkill(namespace, name, &registry.RateSkillRequest{
		Rating:  opts.score,
		Comment: opts.comment,
	})
	if err != nil {
		return fmt.Errorf("评分失败: %w", err)
	}

	fmt.Printf("%s 已为 @%s/%s 评分 %d，当前均分 %.1f (%d)\n",
		color.GreenString("✓"),
		resp.Skill.Namespace,
		resp.Skill.Name,
		resp.UserRating.Rating,
		resp.Skill.Rating,
		resp.Skill.RatingCount,
	)
	return nil
}
