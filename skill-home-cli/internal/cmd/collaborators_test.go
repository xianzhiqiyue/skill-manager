package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/registry"
)

func TestRunListCollaboratorsRequiresLogin(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	err := runListCollaborators("@team/github", &collaboratorsOptions{})
	if err == nil {
		t.Fatal("expected login error")
	}
	if err.Error() != "未登录，请先运行 'skill-home login'" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunAddCollaboratorCallsRegistry(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")

	fake := &fakeRegistryClient{
		upsertCollaboratorResp: &registry.SkillCollaborator{
			Username: "teammate",
			Role:     "viewer",
		},
	}
	restore := swapRegistryClientFactory(func() registryClient {
		return fake
	})
	defer restore()

	if err := runAddCollaborator("@team/github", "@teammate", &collaboratorsOptions{role: "viewer"}); err != nil {
		t.Fatalf("runAddCollaborator returned error: %v", err)
	}
	if len(fake.upsertCollaboratorCalls) != 1 {
		t.Fatalf("expected 1 upsert call, got %d", len(fake.upsertCollaboratorCalls))
	}
	call := fake.upsertCollaboratorCalls[0]
	if call.namespace != "team" || call.name != "github" || call.req == nil || call.req.Username != "teammate" || call.req.Role != "viewer" {
		t.Fatalf("unexpected upsert call: %+v", call)
	}
}

func TestRunAddCollaboratorRejectsInvalidRole(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")

	err := runAddCollaborator("@team/github", "teammate", &collaboratorsOptions{role: "owner"})
	if err == nil {
		t.Fatal("expected invalid role error")
	}
	if !strings.Contains(err.Error(), "maintainer or viewer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRemoveCollaboratorCallsRegistry(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")

	fake := &fakeRegistryClient{}
	restore := swapRegistryClientFactory(func() registryClient {
		return fake
	})
	defer restore()

	if err := runRemoveCollaborator("@team/github", "@teammate", &collaboratorsOptions{yes: true}); err != nil {
		t.Fatalf("runRemoveCollaborator returned error: %v", err)
	}
	if len(fake.deleteCollaboratorCalls) != 1 {
		t.Fatalf("expected 1 delete collaborator call, got %d", len(fake.deleteCollaboratorCalls))
	}
	call := fake.deleteCollaboratorCalls[0]
	if call.namespace != "team" || call.name != "github" || call.username != "teammate" {
		t.Fatalf("unexpected delete collaborator call: %+v", call)
	}
}
