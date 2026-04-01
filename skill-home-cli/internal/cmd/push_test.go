package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/config"
	"github.com/skill-home/cli/internal/registry"
)

func TestPushHelpDoesNotPanicFromFlagConflicts(t *testing.T) {
	root := NewRootCmd("test", "test", "test")
	root.SetArgs([]string{"push", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}

func TestRunPushPublishesCategoryAndTags(t *testing.T) {
	t.Cleanup(func() {
		viper.Reset()
		config.C = nil
	})
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")
	config.C = &config.Config{
		Local: config.Local{DefaultNamespace: "@team"},
	}

	client := &fakeRegistryClient{
		publishResp: &registry.PublishResponse{
			Namespace:   "team",
			Name:        "deploy-buddy",
			Version:     "1.2.3",
			DownloadURL: "/api/v1/download/team/deploy-buddy/1.2.3",
			PublishedAt: "2026-04-01T12:00:00Z",
		},
	}
	restore := swapRegistryClientFactory(func() registryClient {
		return client
	})
	defer restore()

	skillPath := writePushSkill(t, `---
name: deploy-buddy
version: 1.2.3
description: 部署助手
category: ops
tags:
  - deployment
  - ci-cd
license: MIT
---

body
`)

	_, _ = captureStdStreams(t, func() {
		if err := runPush(skillPath, &pushOptions{}); err != nil {
			t.Fatalf("runPush returned error: %v", err)
		}
	})

	if client.publishReq == nil {
		t.Fatal("expected publish request to be captured")
	}
	if client.publishReq.Category != "ops" {
		t.Fatalf("unexpected category: %#v", client.publishReq)
	}
	if len(client.publishReq.Tags) != 2 || client.publishReq.Tags[0] != "deployment" || client.publishReq.Tags[1] != "ci-cd" {
		t.Fatalf("unexpected tags: %#v", client.publishReq.Tags)
	}
}

func TestRunPushRejectsMissingCategoryBeforePublishInNonInteractiveMode(t *testing.T) {
	t.Cleanup(func() {
		viper.Reset()
		config.C = nil
	})
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")
	config.C = &config.Config{
		Local: config.Local{DefaultNamespace: "@team"},
	}

	client := &fakeRegistryClient{}
	restore := swapRegistryClientFactory(func() registryClient {
		return client
	})
	defer restore()

	skillPath := writePushSkill(t, `---
name: deploy-buddy
version: 1.2.3
description: 部署助手
tags:
  - deployment
license: MIT
---

body
`)

	_, _ = captureStdStreams(t, func() {})
	err := runPush(skillPath, &pushOptions{})
	if err == nil {
		t.Fatal("expected runPush to fail")
	}
	if !strings.Contains(err.Error(), "category") {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.publishReq != nil {
		t.Fatalf("publish should not be called: %#v", client.publishReq)
	}
}

func TestRunPushPromptsForMissingMetadataInInteractiveMode(t *testing.T) {
	t.Cleanup(func() {
		viper.Reset()
		config.C = nil
	})
	viper.Reset()
	viper.Set("registry.api_key", "sk_test")
	config.C = &config.Config{
		Local: config.Local{DefaultNamespace: "@team"},
	}

	client := &fakeRegistryClient{
		publishResp: &registry.PublishResponse{
			Namespace:   "team",
			Name:        "deploy-buddy",
			Version:     "1.2.3",
			DownloadURL: "/api/v1/download/team/deploy-buddy/1.2.3",
			PublishedAt: "2026-04-01T12:00:00Z",
		},
	}
	restoreClient := swapRegistryClientFactory(func() registryClient {
		return client
	})
	defer restoreClient()

	previousInteractive := pushTerminalChecker
	pushTerminalChecker = func() bool { return true }
	defer func() {
		pushTerminalChecker = previousInteractive
	}()

	previousPrompter := pushMetadataPrompter
	pushMetadataPrompter = func(category string, tags []string) (string, []string, error) {
		if category != "" {
			t.Fatalf("expected empty category before prompt, got %q", category)
		}
		if len(tags) != 0 {
			t.Fatalf("expected empty tags before prompt, got %#v", tags)
		}
		return "ops", []string{"deploy", "ci"}, nil
	}
	defer func() {
		pushMetadataPrompter = previousPrompter
	}()

	skillPath := writePushSkill(t, `---
name: deploy-buddy
version: 1.2.3
description: 部署助手
license: MIT
---

body
`)

	_, _ = captureStdStreams(t, func() {
		if err := runPush(skillPath, &pushOptions{}); err != nil {
			t.Fatalf("runPush returned error: %v", err)
		}
	})

	if client.publishReq == nil {
		t.Fatal("expected publish request to be captured")
	}
	if client.publishReq.Category != "ops" {
		t.Fatalf("unexpected category: %#v", client.publishReq)
	}
	if len(client.publishReq.Tags) != 2 || client.publishReq.Tags[0] != "deployment" || client.publishReq.Tags[1] != "ci-cd" {
		t.Fatalf("unexpected tags: %#v", client.publishReq.Tags)
	}

	content, err := os.ReadFile(filepath.Join(skillPath, "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(content), "category: ops") {
		t.Fatalf("expected persisted category, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "- deployment") || !strings.Contains(string(content), "- ci-cd") {
		t.Fatalf("expected persisted normalized tags, got:\n%s", string(content))
	}
}

func writePushSkill(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	return dir
}
