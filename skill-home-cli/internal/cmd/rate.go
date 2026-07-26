package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type rateOptions struct {
	score   int
	comment string
}

func newRateCmd() *cobra.Command {
	opts := &rateOptions{}

	cmd := &cobra.Command{
		Use:        "rate <skill-ref>",
		Short:      "已弃用：请使用 feedback 提交使用反馈",
		Deprecated: "五星评分已下线，请使用 'skill-home feedback <skill-ref> --type useful|issue|suggestion --message <内容>'",
		Args:       cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRate(args[0], opts)
		},
	}

	cmd.Flags().IntVarP(&opts.score, "score", "s", 0, "评分 (1-5)")
	cmd.Flags().StringVarP(&opts.comment, "comment", "m", "", "评论")
	return cmd
}

func runRate(skillRef string, opts *rateOptions) error {
	return fmt.Errorf("五星评分已下线，请改用 'skill-home feedback %s --type useful|issue|suggestion --message <内容>'", skillRef)
}
