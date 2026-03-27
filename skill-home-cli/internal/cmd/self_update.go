package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/skill-home/cli/internal/selfupdate"
)

func newSelfUpdateCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "self-update [version]",
		Short: "更新 skill-home CLI 自身",
		Long: `更新当前 skill-home 可执行文件。

示例:
  skill-home self-update         # 更新到最新版本
  skill-home self-update v0.2.3  # 更新到指定版本`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetVersion := ""
			if len(args) > 0 {
				targetVersion = args[0]
			}

			fmt.Printf("当前版本: %s\n", version)
			if targetVersion == "" {
				fmt.Println("目标版本: latest")
			} else {
				fmt.Printf("目标版本: %s\n", targetVersion)
			}

			updater := selfupdate.Updater{
				CurrentVersion:        version,
				HostedReleasesBaseURL: selfupdate.ResolveHostedReleasesBaseURL(registryEndpoint()),
			}

			resolvedVersion, err := updater.Update(targetVersion)
			if err != nil {
				return err
			}

			fmt.Printf("已更新到 %s\n", resolvedVersion)
			fmt.Println("请重新运行 skill-home 以使用新版本。")
			return nil
		},
	}

	return cmd
}
