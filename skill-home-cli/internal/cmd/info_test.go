package cmd

import (
	"encoding/json"
	"testing"

	"github.com/skill-home/cli/internal/registry"
)

func TestRunInfoJSONIncludesCredentials(t *testing.T) {
	restoreFactory := swapRegistryClientFactory(func() registryClient {
		return &fakeRegistryClient{
			getSkillResp: &registry.Skill{
				Namespace:     "user",
				Name:          "credential-descriptor-canary",
				Description:   "用于验证 skill-home 凭证描述符链路的最小 canary skill",
				LatestVersion: "0.1.0",
				Credentials: []registry.SkillCredentialDescriptor{
					{
						ID:          "openai_api_key",
						Env:         "OPENAI_API_KEY",
						Label:       "OpenAI API Key",
						Description: "用于验证 credentials 描述符会被 CLI 正确输出",
						Secret:      true,
						Required:    true,
						Input:       "password",
						HelpURL:     "https://platform.openai.com/api-keys",
					},
				},
			},
		}
	})
	defer restoreFactory()

	var err error
	stdout, stderr := captureStdStreams(t, func() {
		err = runInfo("@user/credential-descriptor-canary", &infoOptions{format: "json"})
	})
	if err != nil {
		t.Fatalf("runInfo returned error: %v", err)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var payload struct {
		Credentials []registry.SkillCredentialDescriptor `json:"credentials"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal(stdout) returned error: %v\nstdout=%q", err, stdout)
	}
	if len(payload.Credentials) != 1 {
		t.Fatalf("len(credentials) = %d, want 1; stdout=%q", len(payload.Credentials), stdout)
	}
	if payload.Credentials[0].Env != "OPENAI_API_KEY" {
		t.Fatalf("credential env = %q, want OPENAI_API_KEY", payload.Credentials[0].Env)
	}
}
