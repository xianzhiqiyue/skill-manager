package cmd

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
)

type registryClient interface {
	HealthCheck() error
	GetCatalogVersion() (*registry.CatalogVersionResponse, error)
	Search(query, namespace string, tags []string, page, perPage int) (*registry.SearchResult, error)
	ListSkills(opts registry.ListSkillsOptions) (*registry.SearchResult, error)
	GetSkill(namespace, name string) (*registry.Skill, error)
	ListVersions(namespace, name string) ([]registry.SkillVersion, error)
	Download(namespace, name, version, outputPath string) error
	DeleteSkill(namespace, name string) error
	DeleteVersion(namespace, name, version string) error
	GetCurrentUser() (*registry.User, error)
	GetUserSkills() ([]registry.Skill, error)
	ListAuditLogs(page, perPage int, action string) (*registry.AuditLogList, error)
	RateSkill(namespace, name string, req *registry.RateSkillRequest) (*registry.RateSkillResponse, error)
	RecordInstallEvent(namespace, name string, req *registry.InstallEventRequest) (*registry.InstallEventResponse, error)
	Publish(skillPath string, req *registry.PublishRequest) (*registry.PublishResponse, error)
	UpdateSkill(namespace, name string, req *registry.UpdateSkillRequest) (*registry.Skill, error)
}

var registryClientFactory = func() registryClient {
	return registry.NewClient(registryEndpoint(), viper.GetString("registry.api_key"))
}

func registryEndpoint() string {
	server := viper.GetString("registry.endpoint")
	if server == "" {
		return config.DefaultRegistryEndpoint
	}
	return server
}

func newRegistryClient() registryClient {
	return registryClientFactory()
}

func requireRegistryLogin() error {
	if viper.GetString("registry.api_key") == "" {
		return fmt.Errorf("未登录，请先运行 'skill-home login'")
	}
	return nil
}

func wrapRegistryReadError(err error) error {
	if err == nil {
		return nil
	}

	apiErr, ok := err.(*registry.APIError)
	if !ok {
		return err
	}
	if apiErr.Code != "FORBIDDEN" {
		return err
	}

	return fmt.Errorf("访问被拒绝：该 skill 可能是私有的，请先运行 'skill-home login' 并确认你有权限")
}
