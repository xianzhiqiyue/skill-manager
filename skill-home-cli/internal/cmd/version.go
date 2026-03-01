package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newVersionCmd(version, commit, buildDate string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(color.CyanString("skill-home"))
			fmt.Printf("  Version:   %s\n", version)
			fmt.Printf("  Commit:    %s\n", commit)
			fmt.Printf("  BuildDate: %s\n", buildDate)
		},
	}
}
