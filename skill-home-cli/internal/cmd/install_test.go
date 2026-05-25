package cmd

import (
	"errors"
	"strings"
	"testing"
)

func TestRecordInstallEventSendsInstallMetadata(t *testing.T) {
	fake := &fakeRegistryClient{}
	restoreFactory := swapRegistryClientFactory(func() registryClient {
		return fake
	})
	defer restoreFactory()

	previousVersion := cliVersion
	cliVersion = "v9.9.9"
	defer func() {
		cliVersion = previousVersion
	}()

	recordInstallEvent(&pulledSkill{
		Namespace: "team",
		Name:      "github",
		Version:   "1.2.3",
	}, &installOptions{
		ide:  "codex",
		mode: "mirror",
	})

	if len(fake.installEventCalls) != 1 {
		t.Fatalf("install event calls = %d, want 1", len(fake.installEventCalls))
	}

	call := fake.installEventCalls[0]
	if call.namespace != "team" || call.name != "github" {
		t.Fatalf("install event target = %s/%s, want team/github", call.namespace, call.name)
	}
	if call.req == nil {
		t.Fatal("install event request is nil")
	}
	if call.req.Version != "1.2.3" || call.req.Target != "codex" || call.req.InstallMode != "mirror" || call.req.ClientVersion != "v9.9.9" {
		t.Fatalf("unexpected install event request: %+v", call.req)
	}
}

func TestRecordInstallEventWarnsOnFailure(t *testing.T) {
	fake := &fakeRegistryClient{installEventErr: errors.New("registry unavailable")}
	restoreFactory := swapRegistryClientFactory(func() registryClient {
		return fake
	})
	defer restoreFactory()

	stdout, _ := captureStdStreams(t, func() {
		recordInstallEvent(&pulledSkill{
			Namespace: "team",
			Name:      "github",
			Version:   "1.2.3",
		}, nil)
	})

	if !strings.Contains(stdout, "安装统计上报失败") {
		t.Fatalf("stdout = %q, want install event warning", stdout)
	}
}
