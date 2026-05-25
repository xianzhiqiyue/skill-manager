package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/skill-home/cli/internal/registry"
)

type fakeRegistryClient struct {
	getCatalogVersionResp *registry.CatalogVersionResponse
	getCatalogVersionErr  error

	getSkillResp *registry.Skill
	getSkillErr  error

	listVersionsResp []registry.SkillVersion
	listVersionsErr  error

	downloadErr error

	deleteSkillCalls []deleteSkillCall
	deleteSkillErr   error

	deleteVersionCalls []deleteVersionCall
	deleteVersionErr   error

	searchResp *registry.SearchResult
	searchErr  error

	listSkillsResp *registry.SearchResult
	listSkillsErr  error

	getCurrentUserResp *registry.User
	getCurrentUserErr  error

	getUserSkillsResp []registry.Skill
	getUserSkillsErr  error

	listAuditLogsResp *registry.AuditLogList
	listAuditLogsErr  error

	rateSkillResp *registry.RateSkillResponse
	rateSkillErr  error

	installEventCalls []installEventCall
	installEventResp  *registry.InstallEventResponse
	installEventErr   error

	publishReq  *registry.PublishRequest
	publishPath string
	publishResp *registry.PublishResponse
	publishErr  error

	updateSkillCalls []updateSkillCall
	updateSkillResp  *registry.Skill
	updateSkillErr   error

	healthCheckErr error
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

type updateSkillCall struct {
	namespace string
	name      string
	req       *registry.UpdateSkillRequest
}

type installEventCall struct {
	namespace string
	name      string
	req       *registry.InstallEventRequest
}

func swapRegistryClientFactory(factory func() registryClient) func() {
	previous := registryClientFactory
	registryClientFactory = factory
	return func() {
		registryClientFactory = previous
	}
}

func (f *fakeRegistryClient) GetCatalogVersion() (*registry.CatalogVersionResponse, error) {
	if f.getCatalogVersionErr != nil || f.getCatalogVersionResp != nil {
		return f.getCatalogVersionResp, f.getCatalogVersionErr
	}
	return &registry.CatalogVersionResponse{}, nil
}

func newTestRemoteCatalogCache(t *testing.T, endpoint string) *remoteCatalogCache {
	t.Helper()
	return newRemoteCatalogCache(filepath.Join(t.TempDir(), "remote-catalog"), endpoint)
}

func captureStdStreams(t *testing.T, fn func()) (stdout string, stderr string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdout returned error: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr returned error: %v", err)
	}

	stdoutDone := make(chan string, 1)
	stderrDone := make(chan string, 1)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stdoutR)
		_ = stdoutR.Close()
		stdoutDone <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, stderrR)
		_ = stderrR.Close()
		stderrDone <- buf.String()
	}()

	os.Stdout = stdoutW
	os.Stderr = stderrW
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		_ = stdoutW.Close()
		_ = stderrW.Close()
		stdout = <-stdoutDone
		stderr = <-stderrDone
	}()

	fn()
	return stdout, stderr
}

func (f *fakeRegistryClient) Search(query, namespace string, tags []string, page, perPage int) (*registry.SearchResult, error) {
	if f.searchResp != nil || f.searchErr != nil {
		return f.searchResp, f.searchErr
	}
	return nil, nil
}

func (f *fakeRegistryClient) ListSkills(opts registry.ListSkillsOptions) (*registry.SearchResult, error) {
	if f.listSkillsResp != nil || f.listSkillsErr != nil {
		return f.listSkillsResp, f.listSkillsErr
	}
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
	return f.getCurrentUserResp, f.getCurrentUserErr
}

func (f *fakeRegistryClient) GetUserSkills() ([]registry.Skill, error) {
	return f.getUserSkillsResp, f.getUserSkillsErr
}

func (f *fakeRegistryClient) ListAuditLogs(page, perPage int, action string) (*registry.AuditLogList, error) {
	return f.listAuditLogsResp, f.listAuditLogsErr
}

func (f *fakeRegistryClient) RateSkill(namespace, name string, req *registry.RateSkillRequest) (*registry.RateSkillResponse, error) {
	return f.rateSkillResp, f.rateSkillErr
}

func (f *fakeRegistryClient) RecordInstallEvent(namespace, name string, req *registry.InstallEventRequest) (*registry.InstallEventResponse, error) {
	call := installEventCall{namespace: namespace, name: name}
	if req != nil {
		copyReq := *req
		call.req = &copyReq
	}
	f.installEventCalls = append(f.installEventCalls, call)
	return f.installEventResp, f.installEventErr
}

func (f *fakeRegistryClient) Publish(skillPath string, req *registry.PublishRequest) (*registry.PublishResponse, error) {
	f.publishPath = skillPath
	if req != nil {
		copyReq := *req
		if req.Tags != nil {
			copyReq.Tags = append([]string{}, req.Tags...)
		}
		f.publishReq = &copyReq
	}
	return f.publishResp, f.publishErr
}

func (f *fakeRegistryClient) UpdateSkill(namespace, name string, req *registry.UpdateSkillRequest) (*registry.Skill, error) {
	call := updateSkillCall{namespace: namespace, name: name}
	if req != nil {
		copyReq := *req
		if req.Tags != nil {
			copyReq.Tags = append([]string{}, req.Tags...)
		}
		call.req = &copyReq
	}
	f.updateSkillCalls = append(f.updateSkillCalls, call)
	return f.updateSkillResp, f.updateSkillErr
}

func (f *fakeRegistryClient) HealthCheck() error {
	return f.healthCheckErr
}
