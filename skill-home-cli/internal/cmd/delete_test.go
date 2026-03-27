package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/registry"
)

func TestRunDeleteRequiresLogin(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	err := runDelete("@team/github", &deleteOptions{yes: true})
	if err == nil {
		t.Fatal("expected login error")
	}
	if err.Error() != "未登录，请先运行 'skill-home login'" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDeleteRejectsVersionedRef(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")

	err := runDelete("@team/github@1.0.0", &deleteOptions{yes: true})
	if err == nil {
		t.Fatal("expected versioned ref error")
	}
	if !strings.Contains(err.Error(), "delete-version") {
		t.Fatalf("expected delete-version hint, got %v", err)
	}
}

func TestRunDeleteCallsRegistryDeleteSkill(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")

	fake := &fakeRegistryClient{}
	restore := swapRegistryClientFactory(func() registryClient {
		return fake
	})
	defer restore()

	if err := runDelete("@team/github", &deleteOptions{yes: true}); err != nil {
		t.Fatalf("runDelete returned error: %v", err)
	}

	if len(fake.deleteSkillCalls) != 1 {
		t.Fatalf("expected 1 delete skill call, got %d", len(fake.deleteSkillCalls))
	}
	call := fake.deleteSkillCalls[0]
	if call.namespace != "team" || call.name != "github" {
		t.Fatalf("unexpected delete skill call: %+v", call)
	}
}

func TestRunDeleteVersionRequiresLogin(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	err := runDeleteVersion("@team/github@1.0.0", &deleteVersionOptions{yes: true})
	if err == nil {
		t.Fatal("expected login error")
	}
	if err.Error() != "未登录，请先运行 'skill-home login'" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDeleteVersionRejectsMissingVersion(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")

	err := runDeleteVersion("@team/github", &deleteVersionOptions{yes: true})
	if err == nil {
		t.Fatal("expected missing version error")
	}
	if !strings.Contains(err.Error(), "delete-version") {
		t.Fatalf("expected delete-version usage hint, got %v", err)
	}
}

func TestRunDeleteVersionCallsRegistryDeleteVersion(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")

	fake := &fakeRegistryClient{}
	restore := swapRegistryClientFactory(func() registryClient {
		return fake
	})
	defer restore()

	if err := runDeleteVersion("@team/github@1.2.3", &deleteVersionOptions{yes: true}); err != nil {
		t.Fatalf("runDeleteVersion returned error: %v", err)
	}

	if len(fake.deleteVersionCalls) != 1 {
		t.Fatalf("expected 1 delete version call, got %d", len(fake.deleteVersionCalls))
	}
	call := fake.deleteVersionCalls[0]
	if call.namespace != "team" || call.name != "github" || call.version != "1.2.3" {
		t.Fatalf("unexpected delete version call: %+v", call)
	}
}

func swapRegistryClientFactory(factory func() registryClient) func() {
	previous := registryClientFactory
	registryClientFactory = factory
	return func() {
		registryClientFactory = previous
	}
}

type fakeRegistryClient struct {
	getSkillResp *registry.Skill
	getSkillErr  error

	listVersionsResp []registry.SkillVersion
	listVersionsErr  error

	downloadErr error

	deleteSkillCalls []deleteSkillCall
	deleteSkillErr   error

	deleteVersionCalls []deleteVersionCall
	deleteVersionErr   error
}

type deleteSkillCall struct {
	namespace string
	name      string
}

type deleteVersionCall struct {
	namespace string
	name      string
	version   string
}

func (f *fakeRegistryClient) Search(query, namespace string, tags []string, page, perPage int) (*registry.SearchResult, error) {
	return nil, nil
}

func (f *fakeRegistryClient) ListSkills(opts registry.ListSkillsOptions) (*registry.SearchResult, error) {
	return nil, nil
}

func (f *fakeRegistryClient) GetSkill(namespace, name string) (*registry.Skill, error) {
	return f.getSkillResp, f.getSkillErr
}

func (f *fakeRegistryClient) ListVersions(namespace, name string) ([]registry.SkillVersion, error) {
	return f.listVersionsResp, f.listVersionsErr
}

func (f *fakeRegistryClient) Download(namespace, name, version, outputPath string) error {
	return f.downloadErr
}

func (f *fakeRegistryClient) DeleteSkill(namespace, name string) error {
	f.deleteSkillCalls = append(f.deleteSkillCalls, deleteSkillCall{namespace: namespace, name: name})
	return f.deleteSkillErr
}

func (f *fakeRegistryClient) DeleteVersion(namespace, name, version string) error {
	f.deleteVersionCalls = append(f.deleteVersionCalls, deleteVersionCall{namespace: namespace, name: name, version: version})
	return f.deleteVersionErr
}

func (f *fakeRegistryClient) GetCurrentUser() (*registry.User, error) {
	return nil, nil
}

func (f *fakeRegistryClient) GetUserSkills() ([]registry.Skill, error) {
	return nil, nil
}

func (f *fakeRegistryClient) ListAuditLogs(page, perPage int, action string) (*registry.AuditLogList, error) {
	return nil, nil
}

func (f *fakeRegistryClient) RateSkill(namespace, name string, req *registry.RateSkillRequest) (*registry.RateSkillResponse, error) {
	return nil, nil
}

func (f *fakeRegistryClient) HealthCheck() error {
	return nil
}
