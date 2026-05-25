package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
)

type loginOptions struct {
	apiKey     string
	server     string
	email      string
	password   string
	apiKeyName string
}

func newLoginCmd() *cobra.Command {
	opts := &loginOptions{}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "登录到注册中心",
		Long:  "使用邮箱/密码登录到 skill-home 服务并自动创建 CLI API Key，或直接使用现成 API Key 登录",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.apiKey, "api-key", "k", "", "API Key")
	cmd.Flags().StringVarP(&opts.server, "server", "s", "", "注册中心地址")
	cmd.Flags().StringVarP(&opts.email, "email", "e", "", "登录邮箱")
	cmd.Flags().StringVarP(&opts.password, "password", "p", "", "登录密码")
	cmd.Flags().StringVar(&opts.apiKeyName, "api-key-name", "", "自动创建的 CLI API Key 名称")

	return cmd
}

func runLogin(opts *loginOptions) error {
	server := strings.TrimSpace(opts.server)
	if server == "" {
		server = registryEndpoint()
	}

	fmt.Printf("正在连接到 %s...\n", color.CyanString(server))

	apiKey := strings.TrimSpace(opts.apiKey)
	if apiKey != "" {
		return loginWithAPIKey(server, apiKey)
	}

	email := strings.TrimSpace(opts.email)
	password := opts.password
	apiKeyName := strings.TrimSpace(opts.apiKeyName)

	if email != "" || password != "" {
		if email == "" {
			return fmt.Errorf("邮箱不能为空")
		}
		if password == "" {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf("密码不能为空")
			}
			var err error
			password, err = promptPassword("请输入密码: ")
			if err != nil {
				return err
			}
		}
		return loginWithAccount(server, email, password, apiKeyName)
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("未提供登录凭据，请使用 --api-key 或 --email/--password")
	}

	method, err := promptLoginMethod()
	if err != nil {
		return err
	}

	if method == "api-key" {
		apiKey, err = promptLine("请输入 API Key: ")
		if err != nil {
			return err
		}
		if strings.TrimSpace(apiKey) == "" {
			return fmt.Errorf("API Key 不能为空")
		}
		return loginWithAPIKey(server, apiKey)
	}

	email, err = promptLine("请输入邮箱: ")
	if err != nil {
		return err
	}
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	password, err = promptPassword("请输入密码: ")
	if err != nil {
		return err
	}
	if password == "" {
		return fmt.Errorf("密码不能为空")
	}

	return loginWithAccount(server, email, password, apiKeyName)
}

func loginWithAPIKey(server, apiKey string) error {
	client := registry.NewClient(server, apiKey)
	user, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}

	return saveLoginSession(server, apiKey, user, "")
}

func loginWithAccount(server, email, password, apiKeyName string) error {
	client := registry.NewClient(server, "")
	authResp, err := client.Login(email, password)
	if err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}

	if apiKeyName == "" {
		apiKeyName = defaultCLIAPIKeyName()
	}

	apiKeyResp, err := registry.NewClient(server, authResp.Token).CreateAPIKey(&registry.CreateAPIKeyRequest{
		Name: apiKeyName,
	})
	if err != nil {
		return fmt.Errorf("创建 CLI API Key 失败: %w", err)
	}

	return saveLoginSession(server, apiKeyResp.Key, &authResp.User, apiKeyName)
}

func saveLoginSession(server, apiKey string, user *registry.User, apiKeyName string) error {
	viper.Set("registry.endpoint", server)
	viper.Set("registry.api_key", apiKey)
	username := ""
	email := ""
	if user != nil {
		username = strings.TrimSpace(user.Username)
		email = strings.TrimSpace(user.Email)
	}
	if username != "" {
		viper.Set("local.default_namespace", "@"+strings.TrimPrefix(username, "@"))
	}

	if err := config.Save(); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Println()
	fmt.Println(color.GreenString("✓"), "登录成功!")
	fmt.Printf("  用户名: %s\n", color.CyanString(username))
	fmt.Printf("  邮箱: %s\n", color.CyanString(email))
	if username != "" {
		fmt.Printf("  默认命名空间: %s\n", color.CyanString(viper.GetString("local.default_namespace")))
	}
	if apiKeyName != "" {
		fmt.Printf("  CLI API Key: %s\n", color.CyanString(apiKeyName))
	}

	return nil
}

func promptLoginMethod() (string, error) {
	fmt.Println("请选择登录方式:")
	fmt.Println("  1. 邮箱 + 密码（自动生成并保存 CLI API Key）")
	fmt.Println("  2. API Key")

	answer, err := promptLine("输入选项 [1/2，默认 1]: ")
	if err != nil {
		return "", err
	}

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "", "1", "email", "password":
		return "account", nil
	case "2", "api", "api-key", "apikey":
		return "api-key", nil
	default:
		return "", fmt.Errorf("不支持的登录方式: %s", answer)
	}
}

func promptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil && len(text) == 0 {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func promptPassword(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return promptLine(prompt)
	}

	fmt.Print(prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(password)), nil
}

func defaultCLIAPIKeyName() string {
	hostname, err := os.Hostname()
	if err == nil && strings.TrimSpace(hostname) != "" {
		return fmt.Sprintf("skill-home-cli@%s", hostname)
	}
	return "skill-home-cli"
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
			if err := requireRegistryLogin(); err != nil {
				return err
			}
			apiKey := viper.GetString("registry.api_key")

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
