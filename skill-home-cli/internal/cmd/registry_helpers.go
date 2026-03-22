package cmd

import (
	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/registry"
)

func registryEndpoint() string {
	server := viper.GetString("registry.endpoint")
	if server == "" {
		return "https://registry.skill-home.dev"
	}
	return server
}

func newRegistryClient() *registry.Client {
	return registry.NewClient(registryEndpoint(), viper.GetString("registry.api_key"))
}
