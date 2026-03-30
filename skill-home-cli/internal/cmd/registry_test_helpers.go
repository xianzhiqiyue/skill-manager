package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/skill-home/cli/internal/registry"
)

func (f *fakeRegistryClient) GetCatalogVersion() (*registry.CatalogVersionResponse, error) {
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
