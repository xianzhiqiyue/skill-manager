package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/registry"
)

func TestWrapRegistryReadErrorAddsLoginHintForForbidden(t *testing.T) {
	err := wrapRegistryReadError(&registry.APIError{
		Code:    "FORBIDDEN",
		Message: "Access denied",
	})
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if !strings.Contains(err.Error(), "该 skill 可能是私有的") {
		t.Fatalf("expected private skill hint, got %v", err)
	}
	if !strings.Contains(err.Error(), "skill-home login") {
		t.Fatalf("expected login hint, got %v", err)
	}
}

func TestWrapRegistryReadErrorLeavesOtherErrorsUnchanged(t *testing.T) {
	original := errors.New("boom")
	if got := wrapRegistryReadError(original); got != original {
		t.Fatalf("expected original error, got %v", got)
	}
}

func TestRunInfoWrapsForbiddenReadError(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	fake := &fakeRegistryClient{
		getSkillErr: &registry.APIError{Code: "FORBIDDEN", Message: "Access denied"},
	}
	restore := swapRegistryClientFactory(func() registryClient {
		return fake
	})
	defer restore()

	err := runInfo("@team/private-skill", &infoOptions{format: "json"})
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	if !strings.Contains(err.Error(), "该 skill 可能是私有的") {
		t.Fatalf("expected private skill hint, got %v", err)
	}
}

func TestPullSkillRefWrapsForbiddenDownloadError(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Reset()

	fake := &fakeRegistryClient{
		downloadErr: &registry.APIError{Code: "FORBIDDEN", Message: "Access denied"},
	}
	restore := swapRegistryClientFactory(func() registryClient {
		return fake
	})
	defer restore()

	_, err := pullSkillRef("@team/private-skill@1.0.0", &pullOptions{
		outputDir: filepath.Join(t.TempDir(), "pulled-skill"),
		extract:   false,
	})
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	if !strings.Contains(err.Error(), "该 skill 可能是私有的") {
		t.Fatalf("expected private skill hint, got %v", err)
	}
}
