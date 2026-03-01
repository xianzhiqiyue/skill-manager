package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
)

type loginOptions struct {
	apiKey string
	server string
}

func newLoginCmd() *cobra.Command {
	opts := &loginOptions{}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "登录到注册中心",
		Long:  "使用 API Key 登录到 skill-home 注册中心",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.apiKey, "api-key", "k", "", "API Key")
	cmd.Flags().StringVarP(&opts.server, "server", "s", "", "注册中心地址")

	return cmd
}

func runLogin(opts *loginOptions) error {
	// 交互式输入 API Key
	apiKey := opts.apiKey
	if apiKey == "" {
		fmt.Print("请输入 API Key: ")
		fmt.Scanln(&apiKey)
		if apiKey == "" {
			return fmt.Errorf("API Key 不能为空")
		}
	}

	// 获取服务器地址
	server := opts.server
	if server == "" {
		server = viper.GetString("registry.endpoint")
	}
	if server == "" {
		server = "https://registry.skill-home.dev"
	}

	fmt.Printf("正在连接到 %s...\n", color.CyanString(server))

	// 创建客户端并验证
	client := registry.NewClient(server, apiKey)
	user, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}

	// 保存配置
	viper.Set("registry.endpoint", server)
	viper.Set("registry.api_key", apiKey)

	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Println()
	fmt.Println(color.GreenString("✓"), "登录成功!")
	fmt.Printf("  用户名: %s\n", color.CyanString(user.Username))
	fmt.Printf("  邮箱: %s\n", color.CyanString(user.Email))

	return nil
}

// newLogoutCmd 登出命令
func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "登出注册中心",
		RunE: func(cmd *cobra.Command, args []string) error {
			viper.Set("registry.api_key", "")
			if err := config.Save(); err != nil {
				return fmt.Errorf("保存配置失败: %w", err)
			}
			fmt.Println(color.GreenString("✓"), "已登出")
			return nil
		},
	}
}

// newWhoamiCmd 显示当前用户命令
func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "显示当前登录用户",
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey := viper.GetString("registry.api_key")
			if apiKey == "" {
				return fmt.Errorf("未登录，请运行 'skill-home login'")
			}

			server := viper.GetString("registry.endpoint")
			client := registry.NewClient(server, apiKey)

			user, err := client.GetCurrentUser()
			if err != nil {
				return fmt.Errorf("获取用户信息失败: %w", err)
			}

			fmt.Printf("已登录用户:\n")
			fmt.Printf("  用户名: %s\n", color.CyanString(user.Username))
			fmt.Printf("  邮箱: %s\n", color.CyanString(user.Email))
			fmt.Printf("  注册时间: %s\n", user.CreatedAt.Format("2006-01-02"))

			return nil
		},
	}
}
