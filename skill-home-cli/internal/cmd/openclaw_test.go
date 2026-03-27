package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/skill-home/cli/internal/config"
)

func TestGetTargetIDEsIncludesOpenClawWhenEnabled(t *testing.T) {
	t.Cleanup(func() {
		config.C = nil
	})

	config.C = &config.Config{
		IDE: config.IDEConfig{
			OpenClaw: config.IDE{Enabled: true},
		},
	}

	got := getTargetIDEs(&syncOptions{})
	want := []string{"openclaw"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("unexpected target IDEs: got %v want %v", got, want)
	}
}

func TestIdeFlagUsageMentionsOpenClaw(t *testing.T) {
	tests := []struct {
		name  string
		usage string
	}{
		{name: "install", usage: newInstallCmd().Flags().Lookup("ide").Usage},
		{name: "sync", usage: newSyncCmd().Flags().Lookup("ide").Usage},
		{name: "uninstall", usage: newUninstallCmd().Flags().Lookup("ide").Usage},
		{name: "update", usage: newUpdateCmd().Flags().Lookup("ide").Usage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.usage, "openclaw") {
				t.Fatalf("usage does not mention openclaw: %q", tt.usage)
			}
		})
	}
}

func TestRunDoctorReportsOpenClawPathsAndSymlinkSupport(t *testing.T) {
	t.Cleanup(func() {
		viper.Reset()
		config.C = nil
	})
	viper.Reset()

	tempDir := t.TempDir()
	workspaceDir := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(filepath.Join(workspaceDir, ".git"), 0755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(workspaceDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore cwd returned error: %v", err)
		}
	})
	globalPath := filepath.Join(tempDir, ".openclaw", "skills")

	config.C = &config.Config{
		Local: config.Local{SkillsDir: tempDir},
		IDE: config.IDEConfig{
			OpenClaw: config.IDE{
				Enabled:     true,
				ProjectPath: "skills",
				GlobalPath:  globalPath,
			},
		},
	}

	projectPath := filepath.Join(workspaceDir, "skills")

	restore := swapRegistryClientFactory(func() registryClient {
		return &fakeRegistryClient{}
	})
	defer restore()

	output := captureStdout(t, func() {
		if err := runDoctor(); err != nil {
			t.Fatalf("runDoctor returned error: %v", err)
		}
	})

	if !strings.Contains(output, "openclaw 项目路径: "+projectPath) {
		t.Fatalf("doctor output missing openclaw project path: %s", output)
	}
	if !strings.Contains(output, "openclaw 全局路径: "+globalPath) {
		t.Fatalf("doctor output missing openclaw global path: %s", output)
	}
	if !strings.Contains(output, "symlink=true") {
		t.Fatalf("doctor output missing symlink support: %s", output)
	}
}

func TestCaptureStdoutRestoresStdoutOnPanic(t *testing.T) {
	oldStdout := os.Stdout

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()

		_ = captureStdout(t, func() {
			panic("boom")
		})
	}()

	if os.Stdout != oldStdout {
		t.Fatal("captureStdout did not restore os.Stdout")
	}
}

func captureStdout(t *testing.T, fn func()) (output string) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe returned error: %v", err)
	}

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		_ = r.Close()
		done <- buf.String()
	}()

	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
		_ = w.Close()
		output = <-done
	}()

	fn()
	return output
}
